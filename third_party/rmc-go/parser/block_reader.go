package parser

import (
	"bufio"
	"fmt"
	"io"
)

// BlockInfo contains metadata about a block
type BlockInfo struct {
	Offset         int64
	Size           uint32
	BlockType      uint8
	MinVersion     uint8
	CurrentVersion uint8
}

// TaggedBlockReader reads tagged blocks from a remarkable v6 file
type TaggedBlockReader struct {
	baseReader    *bufio.Reader
	data          *DataStream
	reader        *bufio.Reader
	currentBlock  *BlockInfo
	limitedReader *LimitedBufReader
}

// NewTaggedBlockReader creates a new TaggedBlockReader
func NewTaggedBlockReader(r io.Reader) *TaggedBlockReader {
	br := bufio.NewReader(r)
	return &TaggedBlockReader{
		baseReader: br,
		data:       NewDataStream(br),
		reader:     br,
	}
}

// ReadHeader reads the file header
func (tbr *TaggedBlockReader) ReadHeader() error {
	return tbr.data.ReadHeader()
}

// ReadBlock reads a top-level block header
func (tbr *TaggedBlockReader) ReadBlock() (*BlockInfo, error) {
	if tbr.currentBlock != nil {
		return nil, fmt.Errorf("already in a block")
	}

	blockLength, err := tbr.data.ReadUint32()
	if err == io.EOF {
		return nil, io.EOF
	}
	if err != nil {
		return nil, err
	}

	unknown, err := tbr.data.ReadUint8()
	if err != nil {
		return nil, err
	}
	_ = unknown // Always 0 in known files, but don't enforce

	minVersion, err := tbr.data.ReadUint8()
	if err != nil {
		return nil, err
	}

	currentVersion, err := tbr.data.ReadUint8()
	if err != nil {
		return nil, err
	}

	blockType, err := tbr.data.ReadUint8()
	if err != nil {
		return nil, err
	}

	tbr.currentBlock = &BlockInfo{
		Size:           blockLength,
		BlockType:      blockType,
		MinVersion:     minVersion,
		CurrentVersion: currentVersion,
	}

	// Create a limited reader for this block
	tbr.limitedReader = NewLimitedBufReader(tbr.baseReader, int64(blockLength))
	// Create a fresh buffered reader for this block's data
	// Use a buffer size equal to the block size to avoid issues
	tbr.reader = bufio.NewReaderSize(tbr.limitedReader, 4096)
	tbr.data = NewDataStream(tbr.reader)

	return tbr.currentBlock, nil
}

// EndBlock finishes reading a block and skips any remaining data
func (tbr *TaggedBlockReader) EndBlock() error {
	if tbr.currentBlock == nil {
		return nil
	}

	// Skip any remaining data
	if tbr.limitedReader != nil {
		if err := tbr.limitedReader.Skip(); err != nil {
			return err
		}
	}

	// Reset to base reader
	tbr.reader = bufio.NewReader(tbr.baseReader)
	tbr.data = NewDataStream(tbr.baseReader)
	tbr.currentBlock = nil
	tbr.limitedReader = nil

	return nil
}

// ReadSubblock reads a subblock
func (tbr *TaggedBlockReader) ReadSubblock(index int) (uint32, error) {
	if err := tbr.data.ReadTag(index, TagTypeLength4); err != nil {
		return 0, err
	}

	length, err := tbr.data.ReadUint32()
	if err != nil {
		return 0, err
	}

	return length, nil
}

// HasSubblock checks if a subblock with the given index exists
func (tbr *TaggedBlockReader) HasSubblock(index int) bool {
	// Peek at the next bytes
	peek, err := tbr.reader.Peek(10) // enough to read a varuint tag
	if err != nil {
		return false
	}

	// Parse the tag from peeked bytes
	var result uint64
	var shift uint
	for i := 0; i < len(peek); i++ {
		b := peek[i]
		result |= uint64(b&0x7F) << shift
		shift += 7
		if (b & 0x80) == 0 {
			break
		}
	}

	tagIndex := int(result >> 4)
	tagType := TagType(result & 0xF)

	return tagIndex == index && tagType == TagTypeLength4
}

// ReadID reads a tagged CRDT ID
func (tbr *TaggedBlockReader) ReadID(index int) (CrdtID, error) {
	if err := tbr.data.ReadTag(index, TagTypeID); err != nil {
		return CrdtID{}, err
	}
	return tbr.data.ReadCrdtID()
}

// ReadBool reads a tagged bool
func (tbr *TaggedBlockReader) ReadBool(index int) (bool, error) {
	if err := tbr.data.ReadTag(index, TagTypeByte1); err != nil {
		return false, err
	}
	return tbr.data.ReadBool()
}

// ReadByte reads a tagged byte
func (tbr *TaggedBlockReader) ReadByte(index int) (uint8, error) {
	if err := tbr.data.ReadTag(index, TagTypeByte1); err != nil {
		return 0, err
	}
	return tbr.data.ReadUint8()
}

// ReadInt reads a tagged 4-byte unsigned integer
func (tbr *TaggedBlockReader) ReadInt(index int) (uint32, error) {
	if err := tbr.data.ReadTag(index, TagTypeByte4); err != nil {
		return 0, err
	}
	return tbr.data.ReadUint32()
}

// ReadFloat reads a tagged 4-byte float
func (tbr *TaggedBlockReader) ReadFloat(index int) (float32, error) {
	if err := tbr.data.ReadTag(index, TagTypeByte4); err != nil {
		return 0, err
	}
	return tbr.data.ReadFloat32()
}

// ReadDouble reads a tagged 8-byte double
func (tbr *TaggedBlockReader) ReadDouble(index int) (float64, error) {
	if err := tbr.data.ReadTag(index, TagTypeByte8); err != nil {
		return 0, err
	}
	return tbr.data.ReadFloat64()
}

// readLww is a generic helper function for reading Last-Write-Wins values
func readLww[T any](tbr *TaggedBlockReader, index int, readFn func(int) (T, error)) (LwwValue[T], error) {
	if _, err := tbr.ReadSubblock(index); err != nil {
		return LwwValue[T]{}, err
	}

	timestamp, err := tbr.ReadID(1)
	if err != nil {
		return LwwValue[T]{}, err
	}

	value, err := readFn(2)
	if err != nil {
		return LwwValue[T]{}, err
	}

	return LwwValue[T]{Timestamp: timestamp, Value: value}, nil
}

// ReadLwwBool reads a last-write-wins bool
func (tbr *TaggedBlockReader) ReadLwwBool(index int) (LwwValue[bool], error) {
	return readLww(tbr, index, tbr.ReadBool)
}

// ReadLwwByte reads a last-write-wins byte
func (tbr *TaggedBlockReader) ReadLwwByte(index int) (LwwValue[uint8], error) {
	return readLww(tbr, index, tbr.ReadByte)
}

// ReadLwwFloat reads a last-write-wins float
func (tbr *TaggedBlockReader) ReadLwwFloat(index int) (LwwValue[float32], error) {
	return readLww(tbr, index, tbr.ReadFloat)
}

// ReadLwwID reads a last-write-wins ID
func (tbr *TaggedBlockReader) ReadLwwID(index int) (LwwValue[CrdtID], error) {
	return readLww(tbr, index, tbr.ReadID)
}

// ReadLwwString reads a last-write-wins string
func (tbr *TaggedBlockReader) ReadLwwString(index int) (LwwValue[string], error) {
	return readLww(tbr, index, tbr.ReadString)
}

// ReadString reads a string block
func (tbr *TaggedBlockReader) ReadString(index int) (string, error) {
	if _, err := tbr.ReadSubblock(index); err != nil {
		return "", err
	}

	return tbr.data.ReadString()
}

// RemainingInBlock returns the number of bytes remaining in the current block
func (tbr *TaggedBlockReader) RemainingInBlock() int64 {
	if tbr.limitedReader == nil {
		return 0
	}
	// Account for bytes buffered in the reader but not yet consumed
	buffered := int64(tbr.reader.Buffered())
	remaining := tbr.limitedReader.Remaining() + buffered
	return remaining
}

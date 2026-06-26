package parser

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const HeaderV6 = "reMarkable .lines file, version=6          "

// DataStream provides low-level reading of remarkable v6 file format
type DataStream struct {
	reader io.Reader
}

// NewDataStream creates a new DataStream
func NewDataStream(r io.Reader) *DataStream {
	return &DataStream{reader: r}
}

// ReadHeader reads and validates the file header
func (ds *DataStream) ReadHeader() error {
	header := make([]byte, len(HeaderV6))
	if _, err := io.ReadFull(ds.reader, header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}
	if string(header) != HeaderV6 {
		return fmt.Errorf("invalid header: %q", string(header))
	}
	return nil
}

// ReadBytes reads exactly n bytes
func (ds *DataStream) ReadBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(ds.reader, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// ReadBool reads a boolean value
func (ds *DataStream) ReadBool() (bool, error) {
	b, err := ds.ReadUint8()
	return b != 0, err
}

// ReadUint8 reads a uint8
func (ds *DataStream) ReadUint8() (uint8, error) {
	buf, err := ds.ReadBytes(1)
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

// ReadUint16 reads a little-endian uint16
func (ds *DataStream) ReadUint16() (uint16, error) {
	buf, err := ds.ReadBytes(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(buf), nil
}

// ReadUint32 reads a little-endian uint32
func (ds *DataStream) ReadUint32() (uint32, error) {
	buf, err := ds.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}

// ReadFloat32 reads a little-endian float32
func (ds *DataStream) ReadFloat32() (float32, error) {
	buf, err := ds.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	bits := binary.LittleEndian.Uint32(buf)
	return math.Float32frombits(bits), nil
}

// ReadFloat64 reads a little-endian float64
func (ds *DataStream) ReadFloat64() (float64, error) {
	buf, err := ds.ReadBytes(8)
	if err != nil {
		return 0, err
	}
	bits := binary.LittleEndian.Uint64(buf)
	return math.Float64frombits(bits), nil
}

// ReadVarUint reads a variable-length unsigned integer
func (ds *DataStream) ReadVarUint() (uint64, error) {
	var result uint64
	var shift uint

	for {
		b, err := ds.ReadUint8()
		if err != nil {
			return 0, err
		}

		result |= uint64(b&0x7F) << shift
		shift += 7

		if (b & 0x80) == 0 {
			break
		}
	}

	return result, nil
}

// ReadCrdtID reads a CRDT ID
func (ds *DataStream) ReadCrdtID() (CrdtID, error) {
	part1, err := ds.ReadUint8()
	if err != nil {
		return CrdtID{}, err
	}

	part2, err := ds.ReadVarUint()
	if err != nil {
		return CrdtID{}, err
	}

	return CrdtID{Part1: uint(part1), Part2: part2}, nil
}

// ReadString reads a length-prefixed string
func (ds *DataStream) ReadString() (string, error) {
	length, err := ds.ReadVarUint()
	if err != nil {
		return "", err
	}

	// Read the "is_ascii" flag
	isAscii, err := ds.ReadBool()
	if err != nil {
		return "", err
	}
	_ = isAscii

	if length == 0 {
		return "", nil
	}

	buf, err := ds.ReadBytes(int(length))
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

// CheckTag checks if the next tag matches without consuming it
func (ds *DataStream) CheckTag(expectedIndex int, expectedType TagType) (bool, error) {
	// We need to peek, but io.Reader doesn't support that
	// For now, we'll use a different approach in TaggedBlockReader
	return false, fmt.Errorf("CheckTag requires buffered reader")
}

// ReadTag reads and validates a tag
func (ds *DataStream) ReadTag(expectedIndex int, expectedType TagType) error {
	x, err := ds.ReadVarUint()
	if err != nil {
		return err
	}

	index := int(x >> 4)
	tagType := TagType(x & 0xF)

	if index != expectedIndex {
		return fmt.Errorf("expected index %d, got %d", expectedIndex, index)
	}

	if tagType != expectedType {
		return fmt.Errorf("expected tag type %s (0x%X), got 0x%X", expectedType, expectedType, tagType)
	}

	return nil
}

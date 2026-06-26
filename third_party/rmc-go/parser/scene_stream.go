package parser

import (
	"fmt"
	"io"
	"math"
)

const (
	BlockTypeMigrationInfo  = 0x00
	BlockTypeSceneTree      = 0x01
	BlockTypeTreeNode       = 0x02
	BlockTypeSceneGlyphItem = 0x03
	BlockTypeSceneGroupItem = 0x04
	BlockTypeSceneLineItem  = 0x05
	BlockTypeSceneTextItem  = 0x06
	BlockTypeRootText       = 0x07
	BlockTypeSceneTombstone = 0x08
	BlockTypeAuthorIDs      = 0x09
	BlockTypePageInfo       = 0x0A
	BlockTypeSceneInfo      = 0x0D

	// Point structure sizes for different versions
	PointSizeV2 = 0x0E // 14 bytes per point (version 2)
	PointSizeV1 = 0x18 // 24 bytes per point (version 1)
)

// SceneTree represents the complete scene with all layers and content
type SceneTree struct {
	Root     *Group
	RootText *Text
	Nodes    map[CrdtID]*Group
}

// NewSceneTree creates a new empty scene tree
func NewSceneTree() *SceneTree {
	rootID := CrdtID{Part1: 0, Part2: 1}
	root := NewEmptyGroup(rootID)

	return &SceneTree{
		Root:  root,
		Nodes: map[CrdtID]*Group{rootID: root},
	}
}

// ReadSceneTree reads a complete scene tree from a reader
func ReadSceneTree(r io.Reader) (*SceneTree, error) {
	reader := NewTaggedBlockReader(r)

	if err := reader.ReadHeader(); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	tree := NewSceneTree()

	for {
		blockInfo, err := reader.ReadBlock()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read block: %w", err)
		}

		if err := tree.processBlock(reader, blockInfo); err != nil {
			// Log the error but continue processing
			// This makes the parser more robust to unknown or malformed blocks
			fmt.Printf("Warning: failed to process block type 0x%02X: %v\n", blockInfo.BlockType, err)
		}

		if err := reader.EndBlock(); err != nil {
			return nil, fmt.Errorf("failed to end block: %w", err)
		}
	}

	return tree, nil
}

// processBlock processes a single block based on its type
func (st *SceneTree) processBlock(reader *TaggedBlockReader, blockInfo *BlockInfo) error {
	switch blockInfo.BlockType {
	case BlockTypeSceneTree:
		return st.readSceneTreeBlock(reader)
	case BlockTypeTreeNode:
		return st.readTreeNodeBlock(reader)
	case BlockTypeSceneGroupItem:
		return st.readSceneGroupItemBlock(reader)
	case BlockTypeSceneLineItem:
		return st.readSceneLineItemBlock(reader, blockInfo.CurrentVersion)
	case BlockTypeRootText:
		return st.readRootTextBlock(reader)
	case BlockTypeMigrationInfo, BlockTypeAuthorIDs, BlockTypePageInfo:
		// Skip these blocks for now
		return nil
	case BlockTypeSceneInfo:
		// Skip SceneInfo block for now
		return nil

	default:
		// Unknown block type - skip
		return nil
	}
}

// readSceneTreeBlock reads a scene tree block
func (st *SceneTree) readSceneTreeBlock(reader *TaggedBlockReader) error {
	treeID, err := reader.ReadID(1)
	if err != nil {
		return fmt.Errorf("failed to read tree ID: %w", err)
	}

	_, err = reader.ReadID(2) // nodeID - not currently used
	if err != nil {
		return fmt.Errorf("failed to read node ID: %w", err)
	}

	_, err = reader.ReadBool(3) // isUpdate
	if err != nil {
		return fmt.Errorf("failed to read isUpdate flag: %w", err)
	}

	_, err = reader.ReadSubblock(4)
	if err != nil {
		return fmt.Errorf("failed to read subblock 4: %w", err)
	}

	parentID, err := reader.ReadID(1)
	if err != nil {
		return fmt.Errorf("failed to read parent ID: %w", err)
	}

	// The tree_id is the ID of the node to create/add
	// Create node with tree_id if it doesn't exist
	if _, exists := st.Nodes[treeID]; !exists {
		st.Nodes[treeID] = NewEmptyGroup(treeID)
	}

	// Create parent if it doesn't exist
	if _, exists := st.Nodes[parentID]; !exists {
		st.Nodes[parentID] = NewEmptyGroup(parentID)
	}

	// Add tree_id node to parent
	parent := st.Nodes[parentID]
	parent.Children.Add(CrdtSequenceItem{
		ItemID: treeID,
		Value:  st.Nodes[treeID],
	})

	return nil
}

// readTreeNodeBlock reads a tree node block
func (st *SceneTree) readTreeNodeBlock(reader *TaggedBlockReader) error {
	nodeID, err := reader.ReadID(1)
	if err != nil {
		return err
	}

	label, err := reader.ReadLwwString(2)
	if err != nil {
		return err
	}

	visible, err := reader.ReadLwwBool(3)
	if err != nil {
		return err
	}

	node, exists := st.Nodes[nodeID]
	if !exists {
		// Create node if it doesn't exist
		node = &Group{
			NodeID:   nodeID,
			Children: NewCrdtSequence(),
			Label:    label,
			Visible:  visible,
		}
		st.Nodes[nodeID] = node
		return nil
	}

	node.Label = label
	node.Visible = visible

	// Check for anchor data
	if reader.HasSubblock(7) {
		anchorID, err := reader.ReadLwwID(7)
		if err != nil {
			return err
		}
		node.AnchorID = &anchorID

		anchorType, err := reader.ReadLwwByte(8)
		if err != nil {
			return err
		}
		node.AnchorType = &anchorType

		anchorThreshold, err := reader.ReadLwwFloat(9)
		if err != nil {
			return err
		}
		node.AnchorThreshold = &anchorThreshold

		anchorOriginX, err := reader.ReadLwwFloat(10)
		if err != nil {
			return err
		}
		node.AnchorOriginX = &anchorOriginX
	}

	return nil
}

// readSceneGroupItemBlock reads a scene group item block
func (st *SceneTree) readSceneGroupItemBlock(reader *TaggedBlockReader) error {
	parentID, err := reader.ReadID(1)
	if err != nil {
		return err
	}

	itemID, err := reader.ReadID(2)
	if err != nil {
		return err
	}

	leftID, err := reader.ReadID(3)
	if err != nil {
		return err
	}

	rightID, err := reader.ReadID(4)
	if err != nil {
		return err
	}

	deletedLength, err := reader.ReadInt(5)
	if err != nil {
		return err
	}

	var nodeID *CrdtID
	if reader.HasSubblock(6) {
		_, err := reader.ReadSubblock(6)
		if err != nil {
			return err
		}

		itemType, err := reader.data.ReadUint8()
		if err != nil {
			return err
		}
		_ = itemType // Should be 0x02 for group item

		id, err := reader.ReadID(2)
		if err != nil {
			return err
		}
		nodeID = &id
	}

	if nodeID == nil {
		return nil
	}

	// Add to parent's children
	parent, exists := st.Nodes[parentID]
	if !exists {
		// Create parent if it doesn't exist
		parent = NewEmptyGroup(parentID)
		st.Nodes[parentID] = parent
	}

	childNode, exists := st.Nodes[*nodeID]
	if !exists {
		// Create child if it doesn't exist
		childNode = NewEmptyGroup(*nodeID)
		st.Nodes[*nodeID] = childNode
	}

	parent.Children.Add(CrdtSequenceItem{
		ItemID:        itemID,
		LeftID:        leftID,
		RightID:       rightID,
		DeletedLength: deletedLength,
		Value:         childNode,
	})

	return nil
}

// readSceneLineItemBlock reads a scene line item block
func (st *SceneTree) readSceneLineItemBlock(reader *TaggedBlockReader, version uint8) error {
	parentID, err := reader.ReadID(1)
	if err != nil {
		return err
	}

	itemID, err := reader.ReadID(2)
	if err != nil {
		return err
	}

	leftID, err := reader.ReadID(3)
	if err != nil {
		return err
	}

	rightID, err := reader.ReadID(4)
	if err != nil {
		return err
	}

	deletedLength, err := reader.ReadInt(5)
	if err != nil {
		return err
	}

	var line *Line
	if reader.HasSubblock(6) {
		_, err := reader.ReadSubblock(6)
		if err != nil {
			return err
		}

		itemType, err := reader.data.ReadUint8()
		if err != nil {
			return err
		}
		_ = itemType // Should be 0x03 for line item

		line, err = readLine(reader, version)
		if err != nil {
			return fmt.Errorf("failed to read line: %w", err)
		}
	}

	if line == nil {
		return nil
	}

	// Add to parent's children
	parent, exists := st.Nodes[parentID]
	if !exists {
		// Create parent if it doesn't exist
		parent = NewEmptyGroup(parentID)
		st.Nodes[parentID] = parent
	}

	parent.Children.Add(CrdtSequenceItem{
		ItemID:        itemID,
		LeftID:        leftID,
		RightID:       rightID,
		DeletedLength: deletedLength,
		Value:         line,
	})

	return nil
}

// readLineMetadata reads the basic line metadata (tool, color, thickness, length)
func readLineMetadata(reader *TaggedBlockReader) (toolID, colorID uint32, thicknessScale float64, startingLength float32, err error) {
	toolID, err = reader.ReadInt(1)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to read tool ID: %w", err)
	}

	colorID, err = reader.ReadInt(2)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to read color ID: %w", err)
	}

	thicknessScale, err = reader.ReadDouble(3)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to read thickness scale: %w", err)
	}

	startingLength, err = reader.ReadFloat(4)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to read starting length: %w", err)
	}

	return toolID, colorID, thicknessScale, startingLength, nil
}

// readLinePoints reads all points for a line from the stream
func readLinePoints(reader *TaggedBlockReader, version uint8) ([]Point, error) {
	subblockLen, err := reader.ReadSubblock(5)
	if err != nil {
		return nil, fmt.Errorf("failed to read points subblock: %w", err)
	}

	pointSize := PointSizeV2
	if version == 1 {
		pointSize = PointSizeV1
	}

	numPoints := int(subblockLen) / pointSize
	extraBytesInSubblock := int(subblockLen) % pointSize

	points := make([]Point, numPoints)
	for i := 0; i < numPoints; i++ {
		point, err := readPoint(reader.data, version)
		if err != nil {
			return nil, fmt.Errorf("failed to read point %d: %w", i, err)
		}
		points[i] = point
	}

	// Check if there are extra bytes at the end of the points subblock
	if extraBytesInSubblock > 0 {
		extra, _ := reader.data.ReadBytes(extraBytesInSubblock)
		fmt.Printf("Debug: Extra bytes in points subblock: %v\n", extra)
	}

	return points, nil
}

// parseColorOverride checks for and parses optional RGBA color override data
func parseColorOverride(reader *TaggedBlockReader) (*RGBA, bool) {
	remaining := reader.RemainingInBlock()
	if remaining < 6 {
		return nil, false
	}

	// Read 2-byte prefix
	_, err := reader.data.ReadBytes(2)
	if err != nil {
		return nil, false
	}

	// Read RGBA color (BGRA order in file)
	b, errB := reader.data.ReadUint8()
	g, errG := reader.data.ReadUint8()
	r, errR := reader.data.ReadUint8()
	a, errA := reader.data.ReadUint8()

	if errB != nil || errG != nil || errR != nil || errA != nil {
		return nil, false
	}

	rgba := RGBA{R: r, G: g, B: b, A: a}
	return &rgba, true
}

// readLine reads a line (stroke) from the stream
func readLine(reader *TaggedBlockReader, version uint8) (*Line, error) {
	// Read basic metadata
	toolID, colorID, thicknessScale, startingLength, err := readLineMetadata(reader)
	if err != nil {
		return nil, err
	}

	// Read all points
	points, err := readLinePoints(reader, version)
	if err != nil {
		return nil, err
	}

	// Read timestamp (unused)
	_, err = reader.ReadID(6)
	if err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}

	// Try to read move_id (optional)
	var moveID *CrdtID
	if reader.HasSubblock(7) {
		id, err := reader.ReadID(7)
		if err == nil {
			moveID = &id
		}
	}

	// Check for color override (highlight/shader colors)
	var colorOverride *RGBA
	if rgba, hasOverride := parseColorOverride(reader); hasOverride {
		colorOverride = rgba
	}

	return &Line{
		Color:          PenColor(colorID),
		ColorOverride:  colorOverride,
		Tool:           Pen(toolID),
		Points:         points,
		ThicknessScale: thicknessScale,
		StartingLength: startingLength,
		MoveID:         moveID,
	}, nil
}

// readPoint reads a point from the stream
func readPoint(ds *DataStream, version uint8) (Point, error) {
	x, err := ds.ReadFloat32()
	if err != nil {
		return Point{}, err
	}

	y, err := ds.ReadFloat32()
	if err != nil {
		return Point{}, err
	}

	var speed uint16
	var width uint16
	var direction uint8
	var pressure uint8

	if version == 1 {
		// Version 1 format
		speedF, err := ds.ReadFloat32()
		if err != nil {
			return Point{}, err
		}
		speed = uint16(speedF * 4)

		dirF, err := ds.ReadFloat32()
		if err != nil {
			return Point{}, err
		}
		direction = uint8(255 * dirF / (math.Pi * 2))

		widthF, err := ds.ReadFloat32()
		if err != nil {
			return Point{}, err
		}
		width = uint16(widthF * 4)

		pressureF, err := ds.ReadFloat32()
		if err != nil {
			return Point{}, err
		}
		pressure = uint8(pressureF * 255)
	} else {
		// Version 2 format
		speed, err = ds.ReadUint16()
		if err != nil {
			return Point{}, err
		}

		width, err = ds.ReadUint16()
		if err != nil {
			return Point{}, err
		}

		direction, err = ds.ReadUint8()
		if err != nil {
			return Point{}, err
		}

		pressure, err = ds.ReadUint8()
		if err != nil {
			return Point{}, err
		}
	}

	return Point{
		X:         x,
		Y:         y,
		Speed:     speed,
		Width:     width,
		Direction: direction,
		Pressure:  pressure,
	}, nil
}

// readTextItems reads all text items from a CRDT sequence
func readTextItems(reader *TaggedBlockReader) (*CrdtSequence, error) {
	// Navigate to text items section
	_, err := reader.ReadSubblock(1)
	if err != nil {
		return nil, fmt.Errorf("failed to read text items subblock 1: %w", err)
	}

	_, err = reader.ReadSubblock(1)
	if err != nil {
		return nil, fmt.Errorf("failed to read text items subblock 2: %w", err)
	}

	numTextItems, err := reader.data.ReadVarUint()
	if err != nil {
		return nil, fmt.Errorf("failed to read number of text items: %w", err)
	}

	textItems := NewCrdtSequence()
	for i := 0; i < int(numTextItems); i++ {
		item, err := readTextItem(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read text item %d: %w", i, err)
		}
		textItems.Add(item)
	}

	return textItems, nil
}

// readTextFormatting reads paragraph style formatting information
func readTextFormatting(reader *TaggedBlockReader) (map[CrdtID]LwwValue[ParagraphStyle], error) {
	_, err := reader.ReadSubblock(2)
	if err != nil {
		return nil, fmt.Errorf("failed to read formatting subblock 1: %w", err)
	}

	_, err = reader.ReadSubblock(1)
	if err != nil {
		return nil, fmt.Errorf("failed to read formatting subblock 2: %w", err)
	}

	numFormats, err := reader.data.ReadVarUint()
	if err != nil {
		return nil, fmt.Errorf("failed to read number of formats: %w", err)
	}

	styles := make(map[CrdtID]LwwValue[ParagraphStyle])
	for i := 0; i < int(numFormats); i++ {
		charID, style, err := readTextFormat(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read text format %d: %w", i, err)
		}
		styles[charID] = style
	}

	return styles, nil
}

// readTextPosition reads the position and width of the text block
func readTextPosition(reader *TaggedBlockReader) (posX, posY float64, width float32, err error) {
	_, err = reader.ReadSubblock(3)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read position subblock: %w", err)
	}

	posX, err = reader.data.ReadFloat64()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read posX: %w", err)
	}

	posY, err = reader.data.ReadFloat64()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read posY: %w", err)
	}

	width, err = reader.ReadFloat(4)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read width: %w", err)
	}

	return posX, posY, width, nil
}

// readRootTextBlock reads the root text block
func (st *SceneTree) readRootTextBlock(reader *TaggedBlockReader) error {
	blockID, err := reader.ReadID(1)
	if err != nil {
		return fmt.Errorf("failed to read block ID: %w", err)
	}
	_ = blockID

	_, err = reader.ReadSubblock(2)
	if err != nil {
		return fmt.Errorf("failed to read root text subblock: %w", err)
	}

	// Read text items
	textItems, err := readTextItems(reader)
	if err != nil {
		return err
	}

	// Read formatting
	styles, err := readTextFormatting(reader)
	if err != nil {
		return err
	}

	// Read position and width
	posX, posY, width, err := readTextPosition(reader)
	if err != nil {
		return err
	}

	st.RootText = &Text{
		Items:  textItems,
		Styles: styles,
		PosX:   posX,
		PosY:   posY,
		Width:  width,
	}

	return nil
}

// readTextItem reads a text item from the stream
func readTextItem(reader *TaggedBlockReader) (CrdtSequenceItem, error) {
	_, err := reader.ReadSubblock(0)
	if err != nil {
		return CrdtSequenceItem{}, err
	}

	itemID, err := reader.ReadID(2)
	if err != nil {
		return CrdtSequenceItem{}, err
	}

	leftID, err := reader.ReadID(3)
	if err != nil {
		return CrdtSequenceItem{}, err
	}

	rightID, err := reader.ReadID(4)
	if err != nil {
		return CrdtSequenceItem{}, err
	}

	deletedLength, err := reader.ReadInt(5)
	if err != nil {
		return CrdtSequenceItem{}, err
	}

	var value interface{} = ""
	if reader.HasSubblock(6) {
		text, err := reader.ReadString(6)
		if err != nil {
			return CrdtSequenceItem{}, err
		}
		value = text
	}

	return CrdtSequenceItem{
		ItemID:        itemID,
		LeftID:        leftID,
		RightID:       rightID,
		DeletedLength: deletedLength,
		Value:         value,
	}, nil
}

// readTextFormat reads text format information
func readTextFormat(reader *TaggedBlockReader) (CrdtID, LwwValue[ParagraphStyle], error) {
	charID, err := reader.data.ReadCrdtID()
	if err != nil {
		return CrdtID{}, LwwValue[ParagraphStyle]{}, err
	}

	timestamp, err := reader.ReadID(1)
	if err != nil {
		return CrdtID{}, LwwValue[ParagraphStyle]{}, err
	}

	_, err = reader.ReadSubblock(2)
	if err != nil {
		return CrdtID{}, LwwValue[ParagraphStyle]{}, err
	}

	c, err := reader.data.ReadUint8()
	if err != nil {
		return CrdtID{}, LwwValue[ParagraphStyle]{}, err
	}
	_ = c // Should be 17

	formatCode, err := reader.data.ReadUint8()
	if err != nil {
		return CrdtID{}, LwwValue[ParagraphStyle]{}, err
	}

	return charID, LwwValue[ParagraphStyle]{
		Timestamp: timestamp,
		Value:     ParagraphStyle(formatCode),
	}, nil
}

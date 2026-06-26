package parser

import "fmt"

// TagType represents the type of following data in a tagged block
type TagType uint8

const (
	TagTypeID      TagType = 0xF
	TagTypeLength4 TagType = 0xC
	TagTypeByte8   TagType = 0x8
	TagTypeByte4   TagType = 0x4
	TagTypeByte1   TagType = 0x1
)

func (t TagType) String() string {
	switch t {
	case TagTypeID:
		return "ID"
	case TagTypeLength4:
		return "Length4"
	case TagTypeByte8:
		return "Byte8"
	case TagTypeByte4:
		return "Byte4"
	case TagTypeByte1:
		return "Byte1"
	default:
		return fmt.Sprintf("Unknown(0x%X)", uint8(t))
	}
}

// CrdtID is a CRDT identifier or timestamp
type CrdtID struct {
	Part1 uint
	Part2 uint64
}

func (c CrdtID) String() string {
	return fmt.Sprintf("CrdtID(%d, %d)", c.Part1, c.Part2)
}

// LwwValue is a last-write-wins value with a timestamp
type LwwValue[T any] struct {
	Timestamp CrdtID
	Value     T
}

// PenColor represents pen color indices
type PenColor uint32

const (
	ColorBlack        PenColor = 0
	ColorGray         PenColor = 1
	ColorWhite        PenColor = 2
	ColorYellow       PenColor = 3
	ColorGreen        PenColor = 4
	ColorPink         PenColor = 5
	ColorBlue         PenColor = 6
	ColorRed          PenColor = 7
	ColorGrayOverlap  PenColor = 8
	ColorHighlight    PenColor = 9
	ColorGreen2       PenColor = 10
	ColorCyan         PenColor = 11
	ColorMagenta      PenColor = 12
	ColorYellow2      PenColor = 13

	// Highlight colors
	ColorHighlightYellow PenColor = 14
	ColorHighlightBlue   PenColor = 15
	ColorHighlightPink   PenColor = 16
	ColorHighlightOrange PenColor = 17
	ColorHighlightGreen  PenColor = 18
	ColorHighlightGray   PenColor = 19

	// Shader colors
	ColorShaderGray    PenColor = 20
	ColorShaderOrange  PenColor = 21
	ColorShaderMagenta PenColor = 22
	ColorShaderBlue    PenColor = 23
	ColorShaderRed     PenColor = 24
	ColorShaderGreen   PenColor = 25
	ColorShaderYellow  PenColor = 26
	ColorShaderCyan    PenColor = 27
)

// RGBA represents an RGBA color from the file
type RGBA struct {
	R, G, B, A uint8
}

// Pen represents different pen/tool types
type Pen uint32

const (
	PenPaintbrush1        Pen = 0
	PenPencil1            Pen = 1
	PenBallpoint1         Pen = 2
	PenMarker1            Pen = 3
	PenFineliner1         Pen = 4
	PenHighlighter1       Pen = 5
	PenEraser             Pen = 6
	PenMechanicalPencil1  Pen = 7
	PenEraserArea         Pen = 8
	PenPaintbrush2        Pen = 12
	PenMechanicalPencil2  Pen = 13
	PenPencil2            Pen = 14
	PenBallpoint2         Pen = 15
	PenMarker2            Pen = 16
	PenFineliner2         Pen = 17
	PenHighlighter2       Pen = 18
	PenCalligraphy        Pen = 21
	PenShader             Pen = 23
)

// IsHighlighter returns true if the pen is a highlighter
func (p Pen) IsHighlighter() bool {
	return p == PenHighlighter1 || p == PenHighlighter2
}

// ParagraphStyle represents text paragraph styles
type ParagraphStyle uint32

const (
	StyleBasic           ParagraphStyle = 0
	StylePlain           ParagraphStyle = 1
	StyleHeading         ParagraphStyle = 2
	StyleBold            ParagraphStyle = 3
	StyleBullet          ParagraphStyle = 4
	StyleBullet2         ParagraphStyle = 5
	StyleCheckbox        ParagraphStyle = 6
	StyleCheckboxChecked ParagraphStyle = 7
	// Additional styles found in newer reMarkable software
	StyleNumbered        ParagraphStyle = 10 // Numbered list (1., 2., 3., etc.)
)

// Point represents a point in a stroke with pressure/speed data
type Point struct {
	X         float32
	Y         float32
	Speed     uint16
	Direction uint8
	Width     uint16
	Pressure  uint8
}

// Line represents a drawn stroke
type Line struct {
	Color          PenColor
	ColorOverride  *RGBA // RGBA color override from file (for highlights/shaders)
	Tool           Pen
	Points         []Point
	ThicknessScale float64
	StartingLength float32
	MoveID         *CrdtID
}

// Rectangle represents a rectangular area
type Rectangle struct {
	X float64
	Y float64
	W float64
	H float64
}

// GlyphRange represents highlighted text in a PDF
type GlyphRange struct {
	Start      *uint32
	Length     uint32
	Text       string
	Color      PenColor
	Rectangles []Rectangle
}

// Text represents a text block
type Text struct {
	Items  *CrdtSequence
	Styles map[CrdtID]LwwValue[ParagraphStyle]
	PosX   float64
	PosY   float64
	Width  float32
}

// Group represents a layer or group of scene items
type Group struct {
	NodeID          CrdtID
	Children        *CrdtSequence
	Label           LwwValue[string]
	Visible         LwwValue[bool]
	AnchorID        *LwwValue[CrdtID]
	AnchorType      *LwwValue[uint8]
	AnchorThreshold *LwwValue[float32]
	AnchorOriginX   *LwwValue[float32]
}

// NewEmptyGroup creates a new empty group with default values
func NewEmptyGroup(id CrdtID) *Group {
	return &Group{
		NodeID:   id,
		Children: NewCrdtSequence(),
		Label:    LwwValue[string]{Timestamp: CrdtID{}, Value: ""},
		Visible:  LwwValue[bool]{Timestamp: CrdtID{}, Value: true},
	}
}

// CrdtSequenceItem represents an item in a CRDT sequence
type CrdtSequenceItem struct {
	ItemID        CrdtID
	LeftID        CrdtID
	RightID       CrdtID
	DeletedLength uint32
	Value         interface{} // Can be string, int, *Group, *Line, etc.
}

// CrdtSequence represents a CRDT sequence
type CrdtSequence struct {
	Items []CrdtSequenceItem
}

// NewCrdtSequence creates a new empty CRDT sequence
func NewCrdtSequence() *CrdtSequence {
	return &CrdtSequence{Items: make([]CrdtSequenceItem, 0)}
}

// Add adds an item to the sequence
func (cs *CrdtSequence) Add(item CrdtSequenceItem) {
	cs.Items = append(cs.Items, item)
}

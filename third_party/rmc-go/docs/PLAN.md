# Text Rendering Implementation Plan

## Overview

This document outlines the plan to implement full text rendering support in rmc-go. Currently, text data is parsed from .rm files but not rendered to SVG output. This plan is based on analysis of the Python rmc implementation and the current state of rmc-go.

## Current State

### What's Already Implemented ✅

1. **Text Parsing** (`internal/rmscene/scene_stream.go:562-656`)
   - `readRootTextBlock()` successfully parses text blocks
   - Reads text items from CRDT sequence
   - Parses paragraph styles/formatting
   - Extracts text position (posX, posY) and width
   - Stores in `SceneTree.RootText`

2. **Data Structures** (`internal/rmscene/types.go`)
   - `Text` struct (lines 191-197): Items, Styles, PosX, PosY, Width
   - `CrdtSequence` (lines 220-233): CRDT sequence container
   - `CrdtSequenceItem` (lines 212-218): Individual sequence items
   - `ParagraphStyle` enum (lines 140-151): Basic, Plain, Heading, Bold, Bullet, Checkbox, etc.
   - `GlyphRange` struct (lines 181-188): For PDF highlights (not yet parsed)
   - `LwwValue` (lines 43-47): Last-Write-Wins values with timestamps

3. **SVG Export Scaffolding** (`internal/export/svg.go`)
   - `buildAnchorPos()` (lines 58-68): Hardcoded special anchors
   - `lineHeights` map (lines 18-26): Style-specific line heights
   - `drawGroup()` (lines 121-142): Group rendering framework

### What's Missing ❌

1. **No SVG text output generation** - `drawGroup()` only handles Group and Line, not Text
2. **No CRDT-to-string conversion** - Text items stored but never reconstructed into readable text
3. **No text-based anchor building** - TODO comment at svg.go:65
4. **No CSS styling output** - No `<style>` block for text formatting
5. **No GlyphRange parsing or rendering** - Type exists but not used
6. **No character-level properties** - Not in Python either (acknowledged limitation)

## Implementation Phases

---

## Phase 1: Basic Text Display

**Goal:** Render text as simple SVG `<text>` elements with correct positioning and paragraph styles.

### Tasks

#### 1.1 Implement CRDT Sequence to String Conversion

**File:** `internal/rmscene/text.go` (new file)

Create helper functions to reconstruct text from CRDT sequences:

```go
package rmscene

// Paragraph represents a text paragraph with style
type Paragraph struct {
    Text    string
    Style   ParagraphStyle
    StartID CrdtID
}

// TextDocument represents a structured text document
type TextDocument struct {
    Paragraphs []Paragraph
}

// BuildTextDocument converts a Text object into a structured document
// by reconstructing strings from CRDT sequences and grouping by style
func BuildTextDocument(text *Text) (*TextDocument, error) {
    // TODO: Implement
    // 1. Iterate through text.Items.Items
    // 2. For each item, extract Value (should be string or rune)
    // 3. Skip items with DeletedLength > 0
    // 4. Group consecutive items by their style (look up in text.Styles)
    // 5. Build Paragraph objects with concatenated text
    return nil, nil
}
```

**Python Reference:** `rmscene` library's `TextDocument.from_scene_item()` (not directly visible, but used in svg.py:268)

**Implementation Notes:**
- CRDT items may be single characters or strings
- Must handle deleted items (DeletedLength > 0)
- Must apply LWW semantics for styles (use timestamp)
- Group consecutive characters with same style into paragraphs

#### 1.2 Create drawText Function

**File:** `internal/export/svg.go`

Add text rendering function:

```go
func drawText(text *rmscene.Text, w io.Writer, indent string) {
    // 1. Convert text to TextDocument
    doc, err := rmscene.BuildTextDocument(text)
    if err != nil {
        return
    }

    // 2. Write opening group tag
    fmt.Fprintf(w, "%s<g class=\"root-text\" style=\"display:inline\">\n", indent)

    // 3. Write CSS style block
    writeTextStyles(w, indent+"\t")

    // 4. Iterate through paragraphs
    yOffset := -88.0 // TEXT_TOP_Y constant
    for _, p := range doc.Paragraphs {
        // Get line height for this style
        lineHeight := lineHeights[p.Style]
        if lineHeight == 0 {
            lineHeight = 70 // default
        }
        yOffset += lineHeight

        // Calculate position
        xPos := text.PosX
        yPos := text.PosY + yOffset

        // Get CSS class name
        className := getStyleClassName(p.Style)

        // Write text element
        trimmedText := strings.TrimSpace(p.Text)
        if trimmedText != "" {
            fmt.Fprintf(w, "%s<text x=\"%.3f\" y=\"%.3f\" class=\"%s\">%s</text>\n",
                indent+"\t", scale(xPos), scale(yPos), className, htmlEscape(trimmedText))
        }
    }

    // 5. Close group
    fmt.Fprintf(w, "%s</g>\n", indent)
}

func writeTextStyles(w io.Writer, indent string) {
    fmt.Fprintf(w, "%s<style>\n", indent)
    fmt.Fprintf(w, "%s\ttext.heading { font: 14pt serif; }\n", indent)
    fmt.Fprintf(w, "%s\ttext.bold { font: 8pt sans-serif bold; }\n", indent)
    fmt.Fprintf(w, "%s\ttext, text.plain { font: 7pt sans-serif; }\n", indent)
    fmt.Fprintf(w, "%s</style>\n", indent)
}

func getStyleClassName(style rmscene.ParagraphStyle) string {
    // Map ParagraphStyle enum to CSS class name
    switch style {
    case rmscene.StyleHeading:
        return "heading"
    case rmscene.StyleBold:
        return "bold"
    case rmscene.StylePlain:
        return "plain"
    case rmscene.StyleBullet:
        return "bullet"
    case rmscene.StyleBullet2:
        return "bullet2"
    case rmscene.StyleCheckbox:
        return "checkbox"
    case rmscene.StyleCheckboxChecked:
        return "checkbox-checked"
    default:
        return "plain"
    }
}

func htmlEscape(s string) string {
    // Escape HTML special characters: <, >, &, ", '
    s = strings.ReplaceAll(s, "&", "&amp;")
    s = strings.ReplaceAll(s, "<", "&lt;")
    s = strings.ReplaceAll(s, ">", "&gt;")
    s = strings.ReplaceAll(s, "\"", "&quot;")
    s = strings.ReplaceAll(s, "'", "&#39;")
    return s
}
```

**Python Reference:** `draw_text()` in svg.py:251-283

**Constants to add:**
```go
const (
    TextTopY = -88.0  // Base Y offset for text
)
```

#### 1.3 Integrate Text Rendering into drawGroup

**File:** `internal/export/svg.go:121-142`

Modify the `drawGroup()` function to handle Text items:

```go
func drawGroup(group *rmscene.Group, w io.Writer, anchorPos map[rmscene.CrdtID]float64, indent string) {
    anchorX, anchorY := getAnchor(group, anchorPos)
    fmt.Fprintf(w, "%s<g id=\"%s\" transform=\"translate(%.3f, %.3f)\">\n",
        indent, group.NodeID, scale(anchorX), scale(anchorY))

    if group.Children != nil {
        for _, item := range group.Children.Items {
            if item.Value == nil {
                continue
            }

            switch v := item.Value.(type) {
            case *rmscene.Group:
                drawGroup(v, w, anchorPos, indent+"\t")
            case *rmscene.Line:
                drawStroke(v, w, indent+"\t")
            case *rmscene.Text:  // NEW CASE
                drawText(v, w, indent+"\t")
            }
        }
    }

    fmt.Fprintf(w, "%s</g>\n", indent)
}
```

#### 1.4 Handle RootText in ExportToSVG

**File:** `internal/export/svg.go:29-52`

Add RootText rendering before or after the main group:

```go
func ExportToSVG(tree *rmscene.SceneTree, w io.Writer) error {
    // Calculate bounding box
    xMin, xMax, yMin, yMax := getBoundingBox(tree.Root, make(map[rmscene.CrdtID]float64))

    width := scale(xMax - xMin + 1)
    height := scale(yMax - yMin + 1)

    // Write SVG header
    fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" height="%.1f" width="%.1f" viewBox="%.1f %.1f %.1f %.1f">
`, height, width, scale(xMin), scale(yMin), width, height)

    fmt.Fprintf(w, "\t<g id=\"p1\" style=\"display:inline\">\n")

    // Render RootText if it exists (NEW)
    if tree.RootText != nil {
        drawText(tree.RootText, w, "\t\t")
    }

    // Draw content
    anchorPos := buildAnchorPos(tree.RootText)
    drawGroup(tree.Root, w, anchorPos, "\t\t")

    // Close
    fmt.Fprintf(w, "\t</g>\n")
    fmt.Fprintf(w, "</svg>\n")

    return nil
}
```

#### 1.5 Testing

**Test files to use:**
- Any .rm files with handwritten text (not just strokes)
- Check if rmc/tests/rm/ has text examples

**Validation:**
1. SVG should contain `<text>` elements
2. Text should be positioned correctly
3. Paragraph styles should apply (different fonts)
4. No crashes on text-less files

---

## Phase 2: Proper Layout and Anchoring

**Goal:** Implement correct text-based anchor positioning so layers can anchor to text paragraphs.

### Tasks

#### 2.1 Update buildAnchorPos to Process Text

**File:** `internal/export/svg.go:58-68`

Replace the TODO with actual implementation:

```go
func buildAnchorPos(text *rmscene.Text) map[rmscene.CrdtID]float64 {
    anchorPos := make(map[rmscene.CrdtID]float64)

    // Special anchors (hardcoded in Python too)
    anchorPos[rmscene.CrdtID{Part1: 0, Part2: 281474976710654}] = 100
    anchorPos[rmscene.CrdtID{Part1: 0, Part2: 281474976710655}] = 100

    // Build anchors from text paragraphs
    if text != nil {
        doc, err := rmscene.BuildTextDocument(text)
        if err == nil {
            yOffset := TextTopY
            for _, p := range doc.Paragraphs {
                lineHeight := lineHeights[p.Style]
                if lineHeight == 0 {
                    lineHeight = 70
                }
                yOffset += lineHeight

                // Map paragraph start ID to Y position
                anchorPos[p.StartID] = text.PosY + yOffset
            }
        }
    }

    return anchorPos
}
```

**Python Reference:** `build_anchor_pos()` in svg.py:112-135

**Note:** This allows layers/groups with `AnchorID` to position themselves relative to specific text paragraphs.

#### 2.2 Improve Bounding Box for Text

**File:** `internal/export/svg.go:70-105`

Add text dimensions to bounding box calculation:

```go
func getBoundingBox(group *rmscene.Group, anchorPos map[rmscene.CrdtID]float64) (float64, float64, float64, float64) {
    xMin := -float64(ScreenWidth) / 2
    xMax := float64(ScreenWidth) / 2
    yMin := 0.0
    yMax := float64(ScreenHeight)

    if group.Children == nil {
        return xMin, xMax, yMin, yMax
    }

    for _, item := range group.Children.Items {
        if item.Value == nil {
            continue
        }

        switch v := item.Value.(type) {
        case *rmscene.Group:
            anchorX, anchorY := getAnchor(v, anchorPos)
            xMinT, xMaxT, yMinT, yMaxT := getBoundingBox(v, anchorPos)
            xMin = math.Min(xMin, xMinT+anchorX)
            xMax = math.Max(xMax, xMaxT+anchorX)
            yMin = math.Min(yMin, yMinT+anchorY)
            yMax = math.Max(yMax, yMaxT+anchorY)

        case *rmscene.Line:
            for _, p := range v.Points {
                xMin = math.Min(xMin, float64(p.X))
                xMax = math.Max(xMax, float64(p.X))
                yMin = math.Min(yMin, float64(p.Y))
                yMax = math.Max(yMax, float64(p.Y))
            }

        case *rmscene.Text:  // NEW CASE
            // Approximate text bounding box
            doc, err := rmscene.BuildTextDocument(v)
            if err == nil {
                xMin = math.Min(xMin, v.PosX)
                xMax = math.Max(xMax, v.PosX+float64(v.Width))

                totalHeight := TextTopY
                for _, p := range doc.Paragraphs {
                    lineHeight := lineHeights[p.Style]
                    if lineHeight == 0 {
                        lineHeight = 70
                    }
                    totalHeight += lineHeight
                }

                yMin = math.Min(yMin, v.PosY+TextTopY)
                yMax = math.Max(yMax, v.PosY+totalHeight)
            }
        }
    }

    return xMin, xMax, yMin, yMax
}
```

#### 2.3 Testing

**Validation:**
1. Layers anchored to text should position correctly
2. SVG viewBox should include all text content
3. Text shouldn't be clipped in output

---

## Phase 3: Advanced Features (Optional)

**Goal:** Add GlyphRange support for PDF highlights and character-level formatting.

### Tasks

#### 3.1 Parse GlyphRange from Scene Items

**File:** `internal/rmscene/scene_stream.go`

Add new block type and parsing:

```go
const (
    // ... existing block types ...
    BlockTypeGlyphRange = 0x?? // TODO: Find actual value from hex dumps
)

func (st *SceneTree) processBlock(blockType uint8, reader *TaggedBlockReader) error {
    switch blockType {
    // ... existing cases ...
    case BlockTypeGlyphRange:
        return st.readGlyphRangeBlock(reader)
    // ...
    }
}

func (st *SceneTree) readGlyphRangeBlock(reader *TaggedBlockReader) error {
    // TODO: Parse GlyphRange data
    // - Start position
    // - Length
    // - Text content
    // - Color
    // - Rectangles
    return nil
}
```

**Research needed:**
- Find actual block type ID for GlyphRange
- Analyze .rm files with PDF highlights to understand format
- May need to look at Python rmscene source or do hex dumps

#### 3.2 Render GlyphRange as SVG Rectangles

**File:** `internal/export/svg.go`

Add rendering for PDF highlights:

```go
func drawGlyphRange(gr *rmscene.GlyphRange, w io.Writer, indent string) {
    // Get highlight color
    pen := createPen(rmscene.PenHighlighter1, gr.Color, 1.0)
    color := pen.baseColor

    // Draw rectangles for each highlighted region
    for _, rect := range gr.Rectangles {
        fmt.Fprintf(w, "%s<rect x=\"%.3f\" y=\"%.3f\" width=\"%.3f\" height=\"%.3f\" ",
            indent, scale(rect.X), scale(rect.Y), scale(rect.W), scale(rect.H))
        fmt.Fprintf(w, "fill=\"rgb(%d,%d,%d)\" opacity=\"0.3\" />\n",
            color.R, color.G, color.B)
    }
}
```

Add to `drawGroup()` switch:
```go
case *rmscene.GlyphRange:
    drawGlyphRange(v, w, indent+"\t")
```

#### 3.3 Character-Level Formatting (Stretch Goal)

**File:** `internal/rmscene/text.go`

Extend `Paragraph` to include formatting runs:

```go
type FormatRun struct {
    Text       string
    Bold       bool
    Italic     bool
    Underline  bool
}

type Paragraph struct {
    Text    string
    Runs    []FormatRun  // Character-level formatting
    Style   ParagraphStyle
    StartID CrdtID
}
```

Modify `BuildTextDocument()` to:
1. Check `CrdtSequenceItem` for properties field
2. Parse font-weight, font-style attributes
3. Build FormatRun slices

Update `drawText()` to use SVG `<tspan>` for inline formatting:

```go
// Instead of single <text> element:
fmt.Fprintf(w, "%s<text x=\"%.3f\" y=\"%.3f\" class=\"%s\">", ...)
for _, run := range p.Runs {
    style := ""
    if run.Bold {
        style += "font-weight:bold;"
    }
    if run.Italic {
        style += "font-style:italic;"
    }
    if style != "" {
        fmt.Fprintf(w, "<tspan style=\"%s\">%s</tspan>", style, htmlEscape(run.Text))
    } else {
        fmt.Fprintf(w, "%s", htmlEscape(run.Text))
    }
}
fmt.Fprintf(w, "</text>\n")
```

**Note:** The Python implementation doesn't do this either (there's a TODO comment acknowledging it at svg.py:279).

---

## Reference Information

### Key Files in rmc-go

1. **`internal/rmscene/scene_stream.go`**
   - Line 562: `readRootTextBlock()` - parses text from binary format
   - Line 658: `readTextItem()` - reads individual CRDT items
   - Line 703: `readTextFormat()` - reads paragraph styles

2. **`internal/rmscene/types.go`**
   - Line 140: `ParagraphStyle` enum
   - Line 153: `Point` struct
   - Line 164: `Line` struct
   - Line 181: `GlyphRange` struct
   - Line 191: `Text` struct
   - Line 212: `CrdtSequenceItem` struct
   - Line 220: `CrdtSequence` struct

3. **`internal/export/svg.go`**
   - Line 18: `lineHeights` map
   - Line 29: `ExportToSVG()` - main entry point
   - Line 58: `buildAnchorPos()` - TODO for text anchors
   - Line 70: `getBoundingBox()` - needs text case
   - Line 121: `drawGroup()` - needs text case

4. **`internal/export/pen.go`**
   - Line 10: `RGB` struct
   - Line 14: `rmPalette` - color definitions
   - Line 49: `pen` struct
   - Line 60: `createPen()` - pen creation

### Python rmc Reference Files

1. **`./rmc/src/rmc/exporters/svg.py`**
   - Line 40: `LINE_HEIGHTS` dictionary
   - Line 82: `TEXT_TOP_Y = -88`
   - Line 112: `build_anchor_pos()` - builds text anchors
   - Line 251: `draw_text()` - main text rendering
   - Line 256: CSS style definitions

2. **`./rmc/src/rmc/exporters/markdown.py`**
   - Line 22: GlyphRange export example

### Constants and Magic Numbers

```go
// Screen dimensions
ScreenWidth  = 1404
ScreenHeight = 1872
ScreenDPI    = 226
Scale        = 72.0 / ScreenDPI

// Text layout
TextTopY     = -88.0  // Base Y offset for text

// Line heights (already defined in svg.go:18-26)
StylePlain:           70
StyleBullet:          35
StyleBullet2:         35
StyleBold:            70
StyleHeading:         150
StyleCheckbox:        35
StyleCheckboxChecked: 35

// Special anchor IDs (already in buildAnchorPos)
{Part1: 0, Part2: 281474976710654}: 100
{Part1: 0, Part2: 281474976710655}: 100
```

### Testing Strategy

1. **Phase 1 Testing:**
   - Find/create .rm files with handwritten text
   - Verify text appears in SVG output
   - Check positioning and styling
   - Ensure no crashes on stroke-only files

2. **Phase 2 Testing:**
   - Test files with layers anchored to text
   - Verify viewBox includes all text
   - Check anchor positioning accuracy

3. **Phase 3 Testing:**
   - Test PDF files with highlights
   - Verify GlyphRange rectangles appear
   - Test mixed bold/italic text (if implemented)

### Known Limitations to Document

1. Character-level formatting (bold/italic within paragraphs) is not implemented in Python either
2. GlyphRange is only exported to markdown in Python, not rendered in SVG
3. Text bounding box calculation is approximate
4. No support for custom fonts (uses generic serif/sans-serif)
5. Bullet points and checkboxes render as styled text, not as actual bullets/checkboxes

---

## Implementation Notes

### CRDT Sequence Reconstruction

The CRDT (Conflict-free Replicated Data Type) sequence is how reMarkable stores text to support collaborative editing. Key points:

- Items have ItemID, LeftID, RightID for ordering
- DeletedLength > 0 means item was deleted (skip it)
- Value can be string or single character
- Items may be out of order in file (need to reconstruct proper order)
- Styles map CrdtID → (Timestamp, ParagraphStyle) with LWW semantics

**Reconstruction Algorithm:**
1. Build map of ItemID → CrdtSequenceItem
2. Find the root (item with LeftID == 0 or special value)
3. Follow RightID chain to build ordered list
4. Skip items with DeletedLength > 0
5. Concatenate Value strings
6. Group by style changes

### LWW (Last-Write-Wins) Semantics

For styles, when multiple style changes exist for same character:
- Compare Timestamp (CrdtID has Part1 and Part2)
- Use style with highest timestamp
- Part2 is typically larger component for comparison

### HTML Escaping

Text from .rm files may contain characters that are special in XML/HTML:
- `&` → `&amp;`
- `<` → `&lt;`
- `>` → `&gt;`
- `"` → `&quot;`
- `'` → `&#39;`

Always escape before writing to SVG output.

---

## Success Criteria

### Phase 1 Complete When:
- [ ] CRDT sequences convert to readable strings
- [ ] Text appears in SVG output with `<text>` elements
- [ ] CSS styling applies to different paragraph types
- [ ] Text positioning is approximately correct
- [ ] No crashes on existing test files

### Phase 2 Complete When:
- [ ] Layers anchor to text paragraphs correctly
- [ ] Bounding box includes text content
- [ ] Text-based anchors build properly
- [ ] Python and Go outputs match for text positioning

### Phase 3 Complete When:
- [ ] PDF highlights (GlyphRange) render as colored rectangles
- [ ] Character-level formatting works (if implemented)
- [ ] All text features match or exceed Python implementation

---

## Questions to Resolve

1. **CRDT Ordering:** How exactly are CRDT items ordered? Need to analyze actual .rm files with text.
2. **GlyphRange Block Type:** What is the actual block type ID for GlyphRange?
3. **Text in Groups:** Can Text items appear as children in regular Groups, or only as RootText?
4. **Character Encoding:** UTF-8? UTF-16? Need to verify.
5. **Empty Paragraphs:** How are blank lines represented in CRDT sequence?

---

## Next Steps

1. Start with Phase 1, Task 1.1: Create `internal/rmscene/text.go`
2. Analyze actual .rm files with text to understand CRDT structure
3. Implement `BuildTextDocument()` function
4. Test text reconstruction before moving to SVG rendering
5. Proceed through Phase 1 tasks sequentially
6. Test after each task completion
7. Move to Phase 2 only after Phase 1 is fully working

---

## Implementation Status

### ✅ Phase 1 Complete (2025-10-24)

All Phase 1 tasks have been successfully implemented:

1. **CRDT Sequence to String Conversion** ✅
   - Created `internal/rmscene/text.go` with `BuildTextDocument()` function
   - Text reconstruction works correctly with newline handling
   - Style mapping implemented (first paragraph uses CrdtID(0,0) style, rest default to plain)

2. **drawText() Function** ✅
   - Implemented in `internal/export/svg.go`
   - Generates SVG `<text>` elements with correct positioning
   - CSS styling applied for different paragraph types
   - HTML character escaping implemented

3. **Integration** ✅
   - Added Text case to `drawGroup()` function
   - Added RootText rendering in `ExportToSVG()`
   - Text rendering confirmed working with test files

**Test Results:**
- `text_multiple_lines.rm`: All 13 lines render correctly with proper styles and positioning
- `text_and_strokes.rm`: Text and strokes render together correctly
- Output matches Python rmc implementation

### ✅ Phase 2 Complete (2025-10-24)

**Proper Layout and Anchoring** has been successfully implemented:

1. **buildAnchorPos() Enhancement** ✅
   - Implemented character-level CRDT ID mapping
   - Each character position (especially newlines) gets its own anchor
   - Groups can now anchor to specific text positions
   - Uses plain style (70pt) line height for consistent spacing

2. **Anchor Position Calculation Fix** ✅
   - Fixed order: build anchors BEFORE calculating bounding box
   - Anchors are now correctly applied to groups that reference text positions
   - Strokes positioned correctly relative to text

3. **Text-Based Anchor Mapping** ✅
   - Maps CRDT character IDs (e.g., CrdtID(1,24)) to Y positions
   - Handles multi-character text items with incrementing IDs
   - Properly calculates cumulative Y offsets based on line heights

**Test Results:**
- `strokes_colour.rm` and `strokes_and_text_colour.rm`: Strokes position identically
- Groups anchored to text lines render at correct Y positions
- No spacing issues when text is present

### Known Limitations (as expected)
- Character-level formatting (bold/italic within paragraphs) not implemented (Python doesn't have this either)
- GlyphRange parsing/rendering not implemented yet (Phase 3)

### Next Steps
Phase 3 (GlyphRange support) remains an optional enhancement.

---

*Last Updated: 2025-10-24*
*Status: Phase 1 & 2 Complete - Text Rendering and Anchoring Working*

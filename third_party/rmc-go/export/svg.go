package export

import (
	"fmt"
	"html"
	"io"
	"math"

	"github.com/joagonca/rmc-go/parser"
)

const (
	// reMarkable tablet screen specifications
	ScreenWidth  = 1404 // reMarkable screen width in pixels
	ScreenHeight = 1872 // reMarkable screen height in pixels
	ScreenDPI    = 226  // reMarkable screen DPI
	Scale        = 72.0 / ScreenDPI
	TextTopY     = -88.0 // Base Y offset for text from top

	// Special anchor IDs (hardcoded in reMarkable v6 format)
	SpecialAnchorID1  = 281474976710654 // 2^48 - 2
	SpecialAnchorID2  = 281474976710655 // 2^48 - 1
	SpecialAnchorYPos = 100.0           // Y position for special anchors
)

var lineHeights = map[parser.ParagraphStyle]float64{
	parser.StylePlain:           70,
	parser.StyleBullet:          35,
	parser.StyleBullet2:         35,
	parser.StyleBold:            70,
	parser.StyleHeading:         150,
	parser.StyleCheckbox:        35,
	parser.StyleCheckboxChecked: 35,
	parser.StyleNumbered:        35,
}

// ExportToSVG exports a scene tree to SVG format
func ExportToSVG(tree *parser.SceneTree, w io.Writer) error {
	if tree == nil {
		return fmt.Errorf("scene tree cannot be nil")
	}
	if tree.Root == nil {
		return fmt.Errorf("scene tree root cannot be nil")
	}

	// Build anchor positions (including text-based anchors)
	anchorPos := buildAnchorPos(tree.RootText)

	// Calculate bounding box using the anchor positions
	xMin, xMax, yMin, yMax := getBoundingBox(tree.Root, anchorPos)

	// Include text area in bounding box calculation
	if tree.RootText != nil {
		textMinX := tree.RootText.PosX
		textMaxX := tree.RootText.PosX + float64(tree.RootText.Width)

		// Calculate text Y range by going through all paragraphs
		// This matches the actual rendering logic in drawText()
		doc, err := parser.BuildTextDocument(tree.RootText)
		if err == nil && len(doc.Paragraphs) > 0 {
			yOffset := TextTopY
			textMinY := math.MaxFloat64
			textMaxY := -math.MaxFloat64

			for _, p := range doc.Paragraphs {
				lineHeight := lineHeights[p.Style]
				if lineHeight == 0 {
					lineHeight = 70
				}
				yOffset += lineHeight
				yPos := tree.RootText.PosY + yOffset

				// Track min and max Y positions
				textMinY = math.Min(textMinY, yPos)
				textMaxY = math.Max(textMaxY, yPos)
			}

			xMin = math.Min(xMin, textMinX)
			xMax = math.Max(xMax, textMaxX)
			yMin = math.Min(yMin, textMinY)
			yMax = math.Max(yMax, textMaxY)
		}
	}

	width := scale(xMax - xMin + 1)
	height := scale(yMax - yMin + 1)

	// Write SVG header
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" height="%.1f" width="%.1f" viewBox="%.1f %.1f %.1f %.1f">
`, height, width, scale(xMin), scale(yMin), width, height)

	fmt.Fprintf(w, "\t<g id=\"p1\" style=\"display:inline\">\n")

	// Render RootText if it exists
	if tree.RootText != nil {
		if err := drawText(tree.RootText, w, "\t\t"); err != nil {
			return fmt.Errorf("failed to draw root text: %w", err)
		}
	}

	// Draw content (use anchor positions without text for strokes)
	if err := drawGroup(tree.Root, w, anchorPos, "\t\t"); err != nil {
		return fmt.Errorf("failed to draw group: %w", err)
	}

	// Close
	fmt.Fprintf(w, "\t</g>\n")
	fmt.Fprintf(w, "</svg>\n")

	return nil
}

func scale(v float64) float64 {
	return v * Scale
}

func buildAnchorPos(text *parser.Text) map[parser.CrdtID]float64 {
	anchorPos := make(map[parser.CrdtID]float64)

	// Special anchors (hardcoded in reMarkable v6 format specification)
	anchorPos[parser.CrdtID{Part1: 0, Part2: SpecialAnchorID1}] = SpecialAnchorYPos
	anchorPos[parser.CrdtID{Part1: 0, Part2: SpecialAnchorID2}] = SpecialAnchorYPos

	// Build anchors from text - we need to map EVERY character position
	// because groups can anchor to specific character positions
	if text != nil && text.Items != nil {
		yOffset := TextTopY

		// Process each text item
		for _, item := range text.Items.Items {
			if item.DeletedLength > 0 || item.Value == nil {
				continue
			}

			str, ok := item.Value.(string)
			if !ok {
				continue
			}

			// Each character in the CRDT has its own ID
			// The ItemID is the ID of the first character,
			// and each subsequent character increments by 1
			currentID := item.ItemID
			for i, ch := range str {
				// Calculate the CRDT ID for this character
				charID := parser.CrdtID{
					Part1: currentID.Part1,
					Part2: currentID.Part2 + uint64(i),
				}

				// Look up the style for this character position
				// For simplicity, use plain style for all lines except explicitly styled ones
				// The reMarkable seems to use plain (70pt) line height for regular text
				currentStyle := parser.StylePlain

				// Only increment on newlines (not on the first character)
				if ch == '\n' {
					// Get line height for current style
					lineHeight := lineHeights[currentStyle]
					if lineHeight == 0 {
						lineHeight = 70
					}
					yOffset += lineHeight

					// Map this character's ID to its Y position
					anchorPos[charID] = text.PosY + yOffset
				} else if i == 0 {
					// For the first character, just map it to current position
					// without incrementing (we already incremented on the previous newline)
					anchorPos[charID] = text.PosY + yOffset
				}
			}
		}
	}

	return anchorPos
}

func getBoundingBox(group *parser.Group, anchorPos map[parser.CrdtID]float64) (float64, float64, float64, float64) {
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
		case *parser.Group:
			anchorX, anchorY := getAnchor(v, anchorPos)
			xMinT, xMaxT, yMinT, yMaxT := getBoundingBox(v, anchorPos)
			xMin = math.Min(xMin, xMinT+anchorX)
			xMax = math.Max(xMax, xMaxT+anchorX)
			yMin = math.Min(yMin, yMinT+anchorY)
			yMax = math.Max(yMax, yMaxT+anchorY)

		case *parser.Line:
			for _, p := range v.Points {
				xMin = math.Min(xMin, float64(p.X))
				xMax = math.Max(xMax, float64(p.X))
				yMin = math.Min(yMin, float64(p.Y))
				yMax = math.Max(yMax, float64(p.Y))
			}
		}
	}

	return xMin, xMax, yMin, yMax
}

func getAnchor(group *parser.Group, anchorPos map[parser.CrdtID]float64) (float64, float64) {
	anchorX := 0.0
	anchorY := 0.0

	if group.AnchorID != nil && group.AnchorOriginX != nil {
		anchorX = float64(group.AnchorOriginX.Value)
		if y, ok := anchorPos[group.AnchorID.Value]; ok {
			anchorY = y
		}
	}

	return anchorX, anchorY
}

func drawGroup(group *parser.Group, w io.Writer, anchorPos map[parser.CrdtID]float64, indent string) error {
	anchorX, anchorY := getAnchor(group, anchorPos)
	fmt.Fprintf(w, "%s<g id=\"%s\" transform=\"translate(%.3f, %.3f)\">\n",
		indent, group.NodeID, scale(anchorX), scale(anchorY))

	if group.Children != nil {
		for _, item := range group.Children.Items {
			if item.Value == nil {
				continue
			}

			switch v := item.Value.(type) {
			case *parser.Group:
				if err := drawGroup(v, w, anchorPos, indent+"\t"); err != nil {
					return err
				}
			case *parser.Line:
				drawStroke(v, w, indent+"\t")
			case *parser.Text:
				if err := drawText(v, w, indent+"\t"); err != nil {
					return err
				}
			}
		}
	}

	fmt.Fprintf(w, "%s</g>\n", indent)
	return nil
}

func drawStroke(line *parser.Line, w io.Writer, indent string) {
	pen := createPen(line.Tool, line.Color, line.ColorOverride, line.ThicknessScale)

	lastXPos := -1.0
	lastYPos := -1.0
	lastSegmentWidth := 0.0

	for i, point := range line.Points {
		xPos := float64(point.X)
		yPos := float64(point.Y)

		if i%pen.segmentLength == 0 {
			// End previous segment
			if lastXPos != -1.0 {
				fmt.Fprintf(w, "\"/>\n")
			}

			segmentColor := pen.getSegmentColor(point, lastSegmentWidth)
			segmentWidth := pen.getSegmentWidth(point, lastSegmentWidth)
			segmentOpacity := pen.getSegmentOpacity(point, lastSegmentWidth)

			fmt.Fprintf(w, "%s<polyline ", indent)
			fmt.Fprintf(w, "style=\"fill:none; stroke:%s; stroke-width:%.3f; opacity:%.3f\" ",
				segmentColor, scale(segmentWidth), segmentOpacity)
			fmt.Fprintf(w, "stroke-linecap=\"%s\" ", pen.strokeLinecap)
			fmt.Fprintf(w, "points=\"")

			if lastXPos != -1.0 {
				fmt.Fprintf(w, "%.3f,%.3f ", scale(lastXPos), scale(lastYPos))
			}

			lastSegmentWidth = segmentWidth
		}

		lastXPos = xPos
		lastYPos = yPos

		fmt.Fprintf(w, "%.3f,%.3f ", scale(xPos), scale(yPos))
	}

	fmt.Fprintf(w, "\" />\n")
}

func drawText(text *parser.Text, w io.Writer, indent string) error {
	// Convert text to TextDocument
	doc, err := parser.BuildTextDocument(text)
	if err != nil {
		return fmt.Errorf("failed to build text document: %w", err)
	}

	// Write opening group tag
	fmt.Fprintf(w, "%s<g class=\"root-text\" style=\"display:inline\">\n", indent)

	// Write CSS style block
	writeTextStyles(w, indent+"\t")

	// Iterate through paragraphs
	yOffset := TextTopY
	bulletNumber := 1 // Counter for numbered bullets (StyleBullet2)
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

		// Write text element (skip empty lines as they just add spacing)
		trimmedText := p.Text // Don't trim - preserve spacing
		if trimmedText != "" {
			// Add appropriate prefix based on style
			prefix := getParagraphPrefix(p.Style, &bulletNumber)
			displayText := prefix + trimmedText

			fmt.Fprintf(w, "%s<text x=\"%.3f\" y=\"%.3f\" class=\"%s\">%s</text>\n",
				indent+"\t", scale(xPos), scale(yPos), className, htmlEscape(displayText))
		}
	}

	// Close group
	fmt.Fprintf(w, "%s</g>\n", indent)
	return nil
}

func writeTextStyles(w io.Writer, indent string) {
	fmt.Fprintf(w, "%s<style>\n", indent)
	fmt.Fprintf(w, "%s\ttext.heading { font: 14pt serif; }\n", indent)
	fmt.Fprintf(w, "%s\ttext.bold { font: 8pt sans-serif; font-weight: bold; }\n", indent)
	fmt.Fprintf(w, "%s\ttext, text.plain { font: 7pt sans-serif; }\n", indent)
	fmt.Fprintf(w, "%s\ttext.bullet { font: 7pt sans-serif; }\n", indent)
	fmt.Fprintf(w, "%s\ttext.bullet2 { font: 7pt sans-serif; }\n", indent)
	fmt.Fprintf(w, "%s\ttext.checkbox { font: 7pt sans-serif; }\n", indent)
	fmt.Fprintf(w, "%s\ttext.checkbox-checked { font: 7pt sans-serif; }\n", indent)
	fmt.Fprintf(w, "%s\ttext.numbered { font: 7pt sans-serif; }\n", indent)
	fmt.Fprintf(w, "%s</style>\n", indent)
}

func getStyleClassName(style parser.ParagraphStyle) string {
	switch style {
	case parser.StyleHeading:
		return "heading"
	case parser.StyleBold:
		return "bold"
	case parser.StylePlain:
		return "plain"
	case parser.StyleBullet:
		return "bullet"
	case parser.StyleBullet2:
		return "bullet2"
	case parser.StyleCheckbox:
		return "checkbox"
	case parser.StyleCheckboxChecked:
		return "checkbox-checked"
	case parser.StyleNumbered:
		return "numbered"
	default:
		return "plain"
	}
}

func htmlEscape(s string) string {
	// Use standard library for proper HTML escaping
	return html.EscapeString(s)
}

// getParagraphPrefix returns the appropriate prefix for a paragraph based on its style
func getParagraphPrefix(style parser.ParagraphStyle, bulletNumber *int) string {
	switch style {
	case parser.StyleBullet:
		// Regular bullet point
		return "• "
	case parser.StyleBullet2:
		// Alternative bullet (could also be numbered in some contexts)
		return "• "
	case parser.StyleNumbered:
		// Numbered list
		prefix := fmt.Sprintf("%d. ", *bulletNumber)
		*bulletNumber++
		return prefix
	case parser.StyleCheckbox:
		// Unchecked checkbox - using ballot box Unicode character
		return "☐ "
	case parser.StyleCheckboxChecked:
		// Checked checkbox - using ballot box with check Unicode character
		return "☑ "
	default:
		return ""
	}
}

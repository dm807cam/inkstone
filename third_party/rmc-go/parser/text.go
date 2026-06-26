package parser

import (
	"fmt"
	"strings"
)

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
	if text == nil || text.Items == nil {
		return &TextDocument{Paragraphs: []Paragraph{}}, nil
	}

	// Build ordered list of text items and track their start positions
	var allText strings.Builder
	var itemStarts []struct {
		itemID CrdtID
		start  int
	}

	for _, item := range text.Items.Items {
		// Skip deleted items
		if item.DeletedLength > 0 {
			continue
		}

		// Extract text value
		if item.Value != nil {
			if str, ok := item.Value.(string); ok {
				startPos := allText.Len()
				allText.WriteString(str)
				itemStarts = append(itemStarts, struct {
					itemID CrdtID
					start  int
				}{itemID: item.ItemID, start: startPos})
			}
		}
	}

	fullText := allText.String()
	if fullText == "" {
		return &TextDocument{Paragraphs: []Paragraph{}}, nil
	}

	// Split by newlines to get paragraphs
	lines := strings.Split(fullText, "\n")

	// Default style is always plain - CrdtID(0, 0) is just a placeholder
	// and shouldn't be used as the default for all text
	defaultStyle := StylePlain

	// Build paragraphs
	doc := &TextDocument{
		Paragraphs: make([]Paragraph, 0, len(lines)),
	}

	charPos := 0
	for i, line := range lines {
		// Find the item ID that corresponds to the start of this line
		// But for styling purposes, we need the ID of the previous newline character
		// (or for the first line, the ID of the first character)
		var styleID CrdtID
		var startID CrdtID
		styleFound := false
		startFound := false

		if i == 0 {
			// First line: use the first item's ID
			if len(itemStarts) > 0 {
				startID = itemStarts[0].itemID
				styleID = startID
				styleFound = true
				startFound = true
			}
		} else {
			// For subsequent lines, the style is determined by the newline character
			// that ended the previous line (charPos - 1)
			styleCharPos := charPos - 1

			// Find the CRDT ID for both positions
			for j, is := range itemStarts {
				// Check if the style position (newline) is within this item's range
				itemEnd := len(fullText)
				if j+1 < len(itemStarts) {
					itemEnd = itemStarts[j+1].start
				}

				if !styleFound && is.start <= styleCharPos && styleCharPos < itemEnd {
					offset := styleCharPos - is.start
					styleID = CrdtID{
						Part1: is.itemID.Part1,
						Part2: is.itemID.Part2 + uint64(offset),
					}
					styleFound = true
				}

				// Get startID for the current line start
				if !startFound && is.start <= charPos && charPos < itemEnd {
					offset := charPos - is.start
					startID = CrdtID{
						Part1: is.itemID.Part1,
						Part2: is.itemID.Part2 + uint64(offset),
					}
					startFound = true
				}

				if styleFound && startFound {
					break
				}
			}
		}

		// Determine style for this paragraph by looking it up in the styles map
		// Each line's style is determined by the CRDT ID of the newline before it
		paraStyle := defaultStyle
		if styleFound {
			if styleValue, exists := text.Styles[styleID]; exists {
				paraStyle = styleValue.Value
			}
		}

		// Even if the line is empty, we want to preserve it for layout
		// (empty lines still take up vertical space)
		para := Paragraph{
			Text:    line,
			Style:   paraStyle,
			StartID: startID,
		}

		doc.Paragraphs = append(doc.Paragraphs, para)

		// Move to next line (line length + newline)
		charPos += len(line) + 1
	}

	return doc, nil
}

// String returns a string representation of the text document
func (doc *TextDocument) String() string {
	var sb strings.Builder
	for i, para := range doc.Paragraphs {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(para.Text)
	}
	return sb.String()
}

// GetStyleName returns a human-readable name for a paragraph style
func GetStyleName(style ParagraphStyle) string {
	switch style {
	case StyleBasic:
		return "basic"
	case StylePlain:
		return "plain"
	case StyleHeading:
		return "heading"
	case StyleBold:
		return "bold"
	case StyleBullet:
		return "bullet"
	case StyleBullet2:
		return "bullet2"
	case StyleCheckbox:
		return "checkbox"
	case StyleCheckboxChecked:
		return "checkbox-checked"
	case StyleNumbered:
		return "numbered"
	default:
		return fmt.Sprintf("unknown-%d", style)
	}
}

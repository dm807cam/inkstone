//go:build cairo
// +build cairo

package export

/*
#cgo pkg-config: cairo
#include <stdlib.h>
#include <cairo.h>
#include <cairo-pdf.h>
*/
import "C"

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"unsafe"

	"github.com/joagonca/rmc-go/parser"
	"github.com/ungerik/go-cairo"
)

// pageDimensions holds the calculated dimensions and anchor positions for a page
type pageDimensions struct {
	width, height float64
	xMin, yMin    float64
	anchorPos     map[parser.CrdtID]float64
}

// setPDFPageSize sets the size for the current page in a PDF surface
// This wraps the cairo_pdf_surface_set_size C function that isn't exposed in go-cairo
func setPDFPageSize(surface *cairo.Surface, width, height float64) {
	surfacePtr, _ := surface.Native()
	C.cairo_pdf_surface_set_size((*C.cairo_surface_t)(unsafe.Pointer(surfacePtr)), C.double(width), C.double(height))
}

// calculatePageDimensions computes the bounding box and dimensions for a scene tree
func calculatePageDimensions(tree *parser.SceneTree) (pageDimensions, error) {
	if tree == nil || tree.Root == nil {
		return pageDimensions{}, fmt.Errorf("scene tree or root cannot be nil")
	}

	// Build anchor positions (including text-based anchors)
	anchorPos := buildAnchorPos(tree.RootText)

	// Calculate bounding box using the anchor positions
	xMin, xMax, yMin, yMax := getBoundingBox(tree.Root, anchorPos)

	// Include text area in bounding box calculation
	if tree.RootText != nil {
		wrapWidth := textWrapWidth(tree.RootText)
		textMinX := tree.RootText.PosX
		textMaxX := tree.RootText.PosX + wrapWidth

		// Calculate text Y range by going through all paragraphs. Each paragraph can
		// wrap into several visual lines, so advance once per wrapped line to keep the
		// page tall enough for the same layout drawTextCairo produces.
		doc, err := parser.BuildTextDocument(tree.RootText)
		if err == nil && len(doc.Paragraphs) > 0 {
			yOffset := TextTopY
			textMinY := math.MaxFloat64
			textMaxY := -math.MaxFloat64
			bulletNumber := 1

			for _, p := range doc.Paragraphs {
				lineHeight := lineHeights[p.Style]
				if lineHeight == 0 {
					lineHeight = 70
				}
				for range paragraphLines(p, wrapWidth, &bulletNumber) {
					yOffset += lineHeight
					yPos := tree.RootText.PosY + yOffset

					textMinY = math.Min(textMinY, yPos)
					textMaxY = math.Max(textMaxY, yPos)
				}
			}

			xMin = math.Min(xMin, textMinX)
			xMax = math.Max(xMax, textMaxX)
			yMin = math.Min(yMin, textMinY)
			yMax = math.Max(yMax, textMaxY)
		}
	}

	width := scale(xMax - xMin + 1)
	height := scale(yMax - yMin + 1)

	return pageDimensions{
		width:     width,
		height:    height,
		xMin:      xMin,
		yMin:      yMin,
		anchorPos: anchorPos,
	}, nil
}

// renderPageToCairo renders a scene tree to a Cairo surface
func renderPageToCairo(tree *parser.SceneTree, surface *cairo.Surface, dims pageDimensions) error {
	// Set up coordinate system
	surface.Save()
	defer surface.Restore()

	surface.Translate(-scale(dims.xMin), -scale(dims.yMin))

	// Draw text first (if it exists)
	if tree.RootText != nil {
		if err := drawTextCairo(tree.RootText, surface); err != nil {
			return fmt.Errorf("failed to draw root text: %w", err)
		}
	}

	// Draw strokes/groups
	if err := drawGroupCairo(tree.Root, surface, dims.anchorPos); err != nil {
		return fmt.Errorf("failed to draw group: %w", err)
	}

	return nil
}

// ExportToPDFCairo exports a scene tree directly to PDF using Cairo
func ExportToPDFCairo(tree *parser.SceneTree, w io.Writer) error {
	// Calculate page dimensions
	dims, err := calculatePageDimensions(tree)
	if err != nil {
		return err
	}

	// Create a temporary file for PDF output
	// Cairo requires a file path, so we write to temp and then copy
	tmpFile, err := os.CreateTemp("", "rmc-cairo-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Create a Cairo PDF surface with the temp file
	pdfSurface := cairo.NewPDFSurface(tmpPath, dims.width, dims.height, cairo.PDF_VERSION_1_5)
	defer pdfSurface.Finish()

	// Render the page
	if err := renderPageToCairo(tree, pdfSurface, dims); err != nil {
		return err
	}

	// Finish the surface to flush all drawing operations
	pdfSurface.Finish()

	// Read the temporary PDF file and write to the output
	pdfData, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read generated PDF: %w", err)
	}

	if _, err := w.Write(pdfData); err != nil {
		return fmt.Errorf("failed to write PDF output: %w", err)
	}

	return nil
}

func drawGroupCairo(group *parser.Group, surface *cairo.Surface, anchorPos map[parser.CrdtID]float64) error {
	surface.Save()

	anchorX, anchorY := getAnchor(group, anchorPos)
	surface.Translate(scale(anchorX), scale(anchorY))

	if group.Children != nil {
		for _, item := range group.Children.Items {
			if item.Value == nil {
				continue
			}

			switch v := item.Value.(type) {
			case *parser.Group:
				if err := drawGroupCairo(v, surface, anchorPos); err != nil {
					return err
				}
			case *parser.Line:
				drawStrokeCairo(v, surface)
			case *parser.Text:
				if err := drawTextCairo(v, surface); err != nil {
					return err
				}
			}
		}
	}

	surface.Restore()
	return nil
}

func drawStrokeCairo(line *parser.Line, surface *cairo.Surface) {
	pen := createPen(line.Tool, line.Color, line.ColorOverride, line.ThicknessScale)

	lastSegmentWidth := 0.0

	for i, point := range line.Points {
		xPos := float64(point.X)
		yPos := float64(point.Y)

		if i%pen.segmentLength == 0 {
			// Start new segment with updated properties
			segmentColor := pen.getSegmentColorRGB(point, lastSegmentWidth)
			segmentWidth := pen.getSegmentWidth(point, lastSegmentWidth)
			segmentOpacity := pen.getSegmentOpacity(point, lastSegmentWidth)

			// Set color with opacity
			surface.SetSourceRGBA(
				float64(segmentColor.R)/255.0,
				float64(segmentColor.G)/255.0,
				float64(segmentColor.B)/255.0,
				segmentOpacity,
			)

			// Set line width
			surface.SetLineWidth(scale(segmentWidth))

			// Set line cap
			if pen.strokeLinecap == "round" {
				surface.SetLineCap(cairo.LINE_CAP_ROUND)
			} else if pen.strokeLinecap == "square" {
				surface.SetLineCap(cairo.LINE_CAP_SQUARE)
			} else {
				surface.SetLineCap(cairo.LINE_CAP_BUTT)
			}

			surface.SetLineJoin(cairo.LINE_JOIN_ROUND)

			// Start new path
			if i == 0 {
				surface.MoveTo(scale(xPos), scale(yPos))
			}

			lastSegmentWidth = segmentWidth
		}

		if i > 0 {
			surface.LineTo(scale(xPos), scale(yPos))
		}

		// Stroke at segment boundaries
		if i > 0 && (i+1)%pen.segmentLength == 0 {
			surface.Stroke()
			// Move to current position to continue
			surface.MoveTo(scale(xPos), scale(yPos))
		}
	}

	// Stroke any remaining path
	surface.Stroke()
}

// textWrapWidth returns the width (in reMarkable units) to wrap a text block to.
// It honors the block's own declared width, falling back to the full screen width
// when the block stores no width (0).
func textWrapWidth(text *parser.Text) float64 {
	if w := float64(text.Width); w > 0 {
		return w
	}
	return ScreenWidth
}

// fontPointSize is the point size used to render a paragraph of the given style,
// matching setTextFontCairo and the SVG style sheet.
func fontPointSize(style parser.ParagraphStyle) float64 {
	switch style {
	case parser.StyleHeading:
		return 14.0
	case parser.StyleBold:
		return 8.0
	default:
		return 7.0
	}
}

// avgCharWidthRatio approximates mean glyph advance as a fraction of the font size
// for the faces used by typed text. Budgeting by character count (instead of
// measuring every glyph) lets page sizing and drawing share one wrap calculation
// without a separate measuring pass, and errs narrow so wrapped lines stay clear of
// the block's right edge.
const avgCharWidthRatio = 0.5

// maxCharsPerLine is how many characters of the given style fit within availWidth
// reMarkable units. The width and the font size both pass through scale(), so the
// ratio is independent of the global point scale. Returns 0 to disable wrapping.
func maxCharsPerLine(style parser.ParagraphStyle, availWidth float64) int {
	if availWidth <= 0 {
		return 0
	}
	charWidth := fontPointSize(style) * avgCharWidthRatio
	if charWidth <= 0 {
		return 0
	}
	if n := int(scale(availWidth) / charWidth); n > 0 {
		return n
	}
	return 1
}

// paragraphLines returns the visual lines a paragraph renders as: its style prefix
// plus text, wrapped to the block width. An empty paragraph yields a single blank
// line (vertical spacing). bulletNumber is advanced only for non-empty paragraphs,
// matching the original numbering. Page sizing and drawing both call this so they
// agree on line counts.
func paragraphLines(p parser.Paragraph, wrapWidth float64, bulletNumber *int) []string {
	if p.Text == "" {
		return []string{""}
	}
	prefix := getParagraphPrefix(p.Style, bulletNumber)
	return wrapParagraph(prefix+p.Text, maxCharsPerLine(p.Style, wrapWidth))
}

// wrapParagraph greedily breaks text into lines of at most maxChars runes, breaking
// at spaces; a word longer than maxChars is hard-split so no single token overflows.
// When maxChars <= 0, or the text already fits, the text is returned unchanged as one
// line so spacing in lines that already fit is preserved exactly.
func wrapParagraph(text string, maxChars int) []string {
	if maxChars <= 0 || len([]rune(text)) <= maxChars {
		return []string{text}
	}

	var lines []string
	var line []rune
	flush := func() {
		lines = append(lines, string(line))
		line = line[:0]
	}
	for _, word := range strings.Fields(text) {
		w := []rune(word)
		// Hard-split a word that cannot fit on a line by itself.
		for len(w) > maxChars {
			if len(line) > 0 {
				flush()
			}
			lines = append(lines, string(w[:maxChars]))
			w = w[maxChars:]
		}
		need := len(w)
		if len(line) > 0 {
			need++ // separating space
		}
		if len(line)+need > maxChars {
			flush()
		}
		if len(line) > 0 {
			line = append(line, ' ')
		}
		line = append(line, w...)
	}
	if len(line) > 0 {
		flush()
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func drawTextCairo(text *parser.Text, surface *cairo.Surface) error {
	// Convert text to TextDocument
	doc, err := parser.BuildTextDocument(text)
	if err != nil {
		return fmt.Errorf("failed to build text document: %w", err)
	}

	// Iterate through paragraphs, wrapping each to the text block's width so a long
	// line (e.g. handwriting converted into a single paragraph) stays on the page
	// instead of running off the right edge.
	wrapWidth := textWrapWidth(text)
	yOffset := TextTopY
	bulletNumber := 1
	for _, p := range doc.Paragraphs {
		// Get line height for this style
		lineHeight := lineHeights[p.Style]
		if lineHeight == 0 {
			lineHeight = 70
		}

		// Font and color are constant across a paragraph's wrapped lines.
		setTextFontCairo(surface, p.Style)
		surface.SetSourceRGB(0, 0, 0)

		// Each wrapped line advances one line height; a blank line only adds spacing.
		for _, line := range paragraphLines(p, wrapWidth, &bulletNumber) {
			yOffset += lineHeight
			if line == "" {
				continue
			}
			surface.MoveTo(scale(text.PosX), scale(text.PosY+yOffset))
			surface.ShowText(line)
		}
	}

	return nil
}

func setTextFontCairo(surface *cairo.Surface, style parser.ParagraphStyle) {
	switch style {
	case parser.StyleHeading:
		surface.SelectFontFace("serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_NORMAL)
		surface.SetFontSize(14.0)
	case parser.StyleBold:
		surface.SelectFontFace("sans-serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
		surface.SetFontSize(8.0)
	default:
		surface.SelectFontFace("sans-serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_NORMAL)
		surface.SetFontSize(7.0)
	}
}

// ExportToMultipagePDFCairo exports multiple scene trees directly to a multipage PDF using Cairo
func ExportToMultipagePDFCairo(trees []*parser.SceneTree, w io.Writer) error {
	if len(trees) == 0 {
		return fmt.Errorf("no scene trees provided")
	}

	// Calculate dimensions for the first page to initialize the PDF surface
	firstDims, err := calculatePageDimensions(trees[0])
	if err != nil {
		return fmt.Errorf("page 1: %w", err)
	}

	// Create a temporary file for PDF output
	// Cairo requires a file path, so we write to temp and then copy
	tmpFile, err := os.CreateTemp("", "rmc-cairo-multipage-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Create PDF surface with first page dimensions
	pdfSurface := cairo.NewPDFSurface(tmpPath, firstDims.width, firstDims.height, cairo.PDF_VERSION_1_5)
	defer pdfSurface.Finish()

	// Render each page
	for pageIdx, tree := range trees {
		// Calculate dimensions for this page
		var dims pageDimensions
		if pageIdx == 0 {
			dims = firstDims
		} else {
			dims, err = calculatePageDimensions(tree)
			if err != nil {
				return fmt.Errorf("page %d: %w", pageIdx+1, err)
			}
			// Set the page size for this page (pages after the first)
			setPDFPageSize(pdfSurface, dims.width, dims.height)
		}

		// Render the page
		if err := renderPageToCairo(tree, pdfSurface, dims); err != nil {
			return fmt.Errorf("page %d: %w", pageIdx+1, err)
		}

		// Show the page (this finalizes the current page and prepares for next)
		if pageIdx < len(trees)-1 {
			pdfSurface.ShowPage()
		}
	}

	// Finish the surface to flush all drawing operations
	pdfSurface.Finish()

	// Read the temporary PDF file and write to the output
	pdfData, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read generated PDF: %w", err)
	}

	if _, err := w.Write(pdfData); err != nil {
		return fmt.Errorf("failed to write PDF output: %w", err)
	}

	return nil
}

// Helper method to get RGB color for Cairo (instead of CSS string)
func (p *pen) getSegmentColorRGB(point parser.Point, lastWidth float64) RGB {
	switch p.name {
	case "Ballpoint":
		speed := float64(point.Speed) / 4.0
		pressure := float64(point.Pressure) / 255.0
		intensity := (0.1 * -(speed / 35.0)) + (1.2 * pressure) + 0.5
		intensity = clamp(intensity)
		factor := math.Min(math.Abs(intensity-1), 0.235)
		r := int(float64(p.baseColor.R) * (1 - factor))
		g := int(float64(p.baseColor.G) * (1 - factor))
		b := int(float64(p.baseColor.B) * (1 - factor))
		return RGB{R: r, G: g, B: b}

	case "Brush":
		speed := float64(point.Speed) / 4.0
		pressure := float64(point.Pressure) / 255.0
		intensity := math.Pow(pressure, 1.5) - 0.2*(speed/50.0)
		intensity = clamp(intensity)
		r := int(float64(p.baseColor.R) * intensity)
		g := int(float64(p.baseColor.G) * intensity)
		b := int(float64(p.baseColor.B) * intensity)
		return RGB{R: r, G: g, B: b}

	default:
		return p.baseColor
	}
}

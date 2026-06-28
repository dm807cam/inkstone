package exporter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ddvk/rmfakecloud/internal/hwr"
	rmcparser "github.com/joagonca/rmc-go/parser"
)

// OCRFormat selects the document shape produced from transcribed handwriting.
type OCRFormat string

const (
	// OCRFormatTxt produces plain text, preserving line breaks and reading order.
	OCRFormatTxt OCRFormat = "txt"
	// OCRFormatMarkdown produces GitHub-flavored Markdown, mapping the visual
	// structure of the notes (headings, lists, checkboxes, tables) to Markdown.
	OCRFormatMarkdown OCRFormat = "md"
)

// Transcription prompts. They are deliberately strict about emitting only the
// transcription (no commentary, labels, or surrounding code fences) so the pages
// concatenate into a clean document.
const (
	ocrTxtPrompt = "Transcribe all handwritten text in this image to plain text. " +
		"Preserve line breaks and the natural reading order. " +
		"Do not add commentary, headings, labels, or code fences. Output only the transcription."

	ocrMarkdownPrompt = "Transcribe all handwritten text in this image into well-structured " +
		"GitHub-flavored Markdown that mirrors the visual structure of the handwriting: " +
		"use '#'/'##' headings for titles, '- ' for bullet lists, '1.' for numbered lists, " +
		"'- [ ]'/'- [x]' for unchecked/checked boxes, **bold** for emphasized words, and a " +
		"Markdown table where the layout is tabular. Preserve the natural reading order. " +
		"Do not wrap the whole answer in a code fence or add commentary. Output only the Markdown."
)

// errNoStrokes is returned by renderRmPageToPNG for a page with no ink (e.g. a blank or
// typed-text-only page); the OCR loop treats it as a page to skip, not a failure.
var errNoStrokes = errors.New("page has no strokes to transcribe")

// Transcriber turns a rendered page image into recognized text using the given prompt.
// *hwr.LLMClient satisfies this; the indirection keeps OCRDocument unit-testable.
type Transcriber interface {
	Transcribe(png []byte, prompt string) (string, error)
}

// OCRDocument renders each page's handwriting to an image, transcribes it with the vision
// model, and writes a single plain-text or Markdown document covering the whole notebook.
// pages holds raw v6 .rm page bytes in page order; nil/empty entries (un-annotated pages)
// and pages with no ink are skipped. It errors only if nothing at all could be transcribed.
func OCRDocument(pages [][]byte, format OCRFormat, t Transcriber, w io.Writer) error {
	prompt := ocrTxtPrompt
	if format == OCRFormatMarkdown {
		prompt = ocrMarkdownPrompt
	}

	var doc strings.Builder
	transcribed := 0
	for i, page := range pages {
		if len(page) == 0 {
			continue
		}
		png, err := renderRmPageToPNG(page)
		if err != nil {
			if errors.Is(err, errNoStrokes) {
				continue
			}
			return fmt.Errorf("render page %d: %w", i+1, err)
		}
		text, err := t.Transcribe(png, prompt)
		if err != nil {
			return fmt.Errorf("transcribe page %d: %w", i+1, err)
		}
		if text = strings.TrimSpace(text); text == "" {
			continue
		}
		if doc.Len() > 0 {
			doc.WriteString(pageSeparator(format))
		}
		doc.WriteString(text)
		doc.WriteString("\n")
		transcribed++
	}

	if transcribed == 0 {
		return fmt.Errorf("no handwriting could be transcribed (only v6 notebooks are supported for OCR export)")
	}
	_, err := io.WriteString(w, doc.String())
	return err
}

// pageSeparator joins consecutive pages. Markdown gets a thematic break so pages stay
// visually distinct; plain text just gets a blank line.
func pageSeparator(format OCRFormat) string {
	if format == OCRFormatMarkdown {
		return "\n---\n\n"
	}
	return "\n"
}

// renderRmPageToPNG parses a v6 .rm page and rasterizes its ink strokes to a PNG so a
// vision model can read the handwriting. Typed-text blocks are not rasterized; only ink
// strokes are. Returns errNoStrokes when the page contains no strokes.
func renderRmPageToPNG(rmData []byte) ([]byte, error) {
	tree, err := rmcparser.ReadSceneTree(bytes.NewReader(rmData))
	if err != nil {
		return nil, fmt.Errorf("parse rm page: %w", err)
	}

	var strokes []hwr.Stroke
	if tree != nil && tree.Root != nil {
		collectStrokes(tree.Root, 0, 0, &strokes)
	}
	if len(strokes) == 0 {
		return nil, errNoStrokes
	}
	return hwr.RenderStrokesToPNG(strokes)
}

// collectStrokes walks the scene tree depth-first, accumulating each group's horizontal
// anchor offset (AnchorOriginX, used by the "move selection" tool), and appends every
// line's points translated into page space. Vertical text-anchored offsets are not
// resolved here; for OCR a small placement error on moved selections is harmless.
func collectStrokes(group *rmcparser.Group, offX, offY float64, out *[]hwr.Stroke) {
	if group.AnchorOriginX != nil {
		offX += float64(group.AnchorOriginX.Value)
	}
	if group.Children == nil {
		return
	}
	for _, item := range group.Children.Items {
		switch v := item.Value.(type) {
		case *rmcparser.Group:
			collectStrokes(v, offX, offY, out)
		case *rmcparser.Line:
			s := hwr.Stroke{
				X: make([]float64, 0, len(v.Points)),
				Y: make([]float64, 0, len(v.Points)),
			}
			for _, p := range v.Points {
				s.X = append(s.X, float64(p.X)+offX)
				s.Y = append(s.Y, float64(p.Y)+offY)
			}
			if len(s.X) > 0 {
				*out = append(*out, s)
			}
		}
	}
}

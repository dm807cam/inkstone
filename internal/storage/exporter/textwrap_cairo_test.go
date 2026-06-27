//go:build cairo

package exporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/joagonca/rmc-go/export"
	"github.com/joagonca/rmc-go/parser"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// sceneWithText builds a minimal scene tree whose root text holds a single
// paragraph rendered inside a text block of the given width (reMarkable units).
// PosY is placed below the screen so every wrapped line extends the page height,
// letting a test read line count off the page height.
func sceneWithText(label string, width float32) *parser.SceneTree {
	tree := parser.NewSceneTree()
	tree.RootText = &parser.Text{
		Items: &parser.CrdtSequence{Items: []parser.CrdtSequenceItem{
			{ItemID: parser.CrdtID{Part1: 1, Part2: 1}, Value: label},
		}},
		Styles: map[parser.CrdtID]parser.LwwValue[parser.ParagraphStyle]{},
		PosX:   -float64(width) / 2,
		PosY:   2000,
		Width:  width,
	}
	return tree
}

func pdfPageHeight(t *testing.T, pdf []byte) float64 {
	t.Helper()
	dims, err := api.PageDims(bytes.NewReader(pdf), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatalf("read page dims: %v", err)
	}
	if len(dims) == 0 {
		t.Fatal("pdf has no pages")
	}
	return dims[0].Height
}

// TestConvertedTextWraps proves the cairo renderer wraps a long typed-text
// paragraph to the block width. The same text in a narrow block must wrap into
// more lines — and so a taller page — than in a wide block. Before the wrap patch,
// the block width did not affect line count, so the two pages were the same height
// and this test would fail.
func TestConvertedTextWraps(t *testing.T) {
	// A long single paragraph, like handwriting converted to one line of text.
	long := strings.TrimSpace(strings.Repeat("This is a test on how the OCR works. ", 8))

	var narrow, wide bytes.Buffer
	if err := export.ExportToPDFCairo(sceneWithText(long, 300), &narrow); err != nil {
		t.Fatalf("render narrow block: %v", err)
	}
	if err := export.ExportToPDFCairo(sceneWithText(long, 1400), &wide); err != nil {
		t.Fatalf("render wide block: %v", err)
	}

	hNarrow := pdfPageHeight(t, narrow.Bytes())
	hWide := pdfPageHeight(t, wide.Bytes())
	if hNarrow <= hWide {
		t.Errorf("narrow block should wrap into a taller page than wide; got narrow=%.1f wide=%.1f", hNarrow, hWide)
	}
}

// TestShortTextDoesNotWrap guards the common case: text that already fits the block
// is left untouched (one visual line), so wrapping never reflows lines that fit.
func TestShortTextDoesNotWrap(t *testing.T) {
	short := "Short line."

	var narrow, wide bytes.Buffer
	if err := export.ExportToPDFCairo(sceneWithText(short, 300), &narrow); err != nil {
		t.Fatalf("render narrow block: %v", err)
	}
	if err := export.ExportToPDFCairo(sceneWithText(short, 1400), &wide); err != nil {
		t.Fatalf("render wide block: %v", err)
	}

	if h1, h2 := pdfPageHeight(t, narrow.Bytes()), pdfPageHeight(t, wide.Bytes()); h1 != h2 {
		t.Errorf("short text should occupy one line regardless of width; got narrow=%.1f wide=%.1f", h1, h2)
	}
}

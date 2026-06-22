package exporter

import (
	"bytes"
	"fmt"
	"io"
	"os"

	rmc "github.com/joagonca/rmc-go"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ExportV6ToPdfNative converts v6 .rm file to PDF using rmc-go library (in-process)
// This uses the Cairo renderer for native PDF generation
func ExportV6ToPdfNative(rmData []byte, output io.Writer) error {
	opts := &rmc.Options{
		UseLegacy: false, // Always use Cairo renderer (not Inkscape)
	}

	// Convert from bytes to PDF bytes
	pdfData, err := rmc.ConvertFromBytes(rmData, rmc.FormatPDF, opts)
	if err != nil {
		return fmt.Errorf("failed to convert v6 rm to PDF: %w", err)
	}

	// Write to output
	_, err = io.Copy(output, bytes.NewReader(pdfData))
	if err != nil {
		return fmt.Errorf("failed to write PDF output: %w", err)
	}

	return nil
}

// ExportV6ToSvgNative converts v6 .rm file to SVG using rmc-go library
func ExportV6ToSvgNative(rmData []byte, output io.Writer) error {
	opts := &rmc.Options{}

	svgData, err := rmc.ConvertFromBytes(rmData, rmc.FormatSVG, opts)
	if err != nil {
		return fmt.Errorf("failed to convert v6 rm to SVG: %w", err)
	}

	_, err = io.Copy(output, bytes.NewReader(svgData))
	if err != nil {
		return fmt.Errorf("failed to write SVG output: %w", err)
	}

	return nil
}

// ExportV6MultiPageToPdfNative converts multiple v6 .rm pages to a single PDF
func ExportV6MultiPageToPdfNative(pages [][]byte, output io.Writer) error {
	if len(pages) == 0 {
		return fmt.Errorf("no pages provided")
	}

	opts := &rmc.Options{
		UseLegacy: false, // Use Cairo renderer
	}

	// Use rmc-go's multipage function
	pdfData, err := rmc.ConvertMultipleFromBytes(pages, opts)
	if err != nil {
		return fmt.Errorf("failed to convert multiple v6 pages to PDF: %w", err)
	}

	_, err = io.Copy(output, bytes.NewReader(pdfData))
	return err
}

// writeTempPDF writes b to a uniquely named temp file and returns its path.
func writeTempPDF(prefix string, b []byte) (string, error) {
	f, err := os.CreateTemp("", prefix)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// ExportV6MultiPageOverBackground renders v6 annotation pages and overlays each
// on top of the corresponding page of the original imported PDF (background),
// preserving the background's pages, sizes and ordering.
//
// pages is indexed by background page number: pages[i] holds the raw v6 .rm
// bytes for page i, or nil if that page has no annotations. Pages without
// annotations are emitted unchanged from the background.
//
// Alignment note: the v6 annotation surface is sized to the reMarkable page
// (clamped to the full screen by rmc-go); it is stamped scaled-to-fit and
// centered on each background page. This is correct for the common case where
// annotations stay within the page bounds and aspect ratios match.
func ExportV6MultiPageOverBackground(pages [][]byte, background io.Reader, output io.Writer) error {
	if len(pages) == 0 {
		return fmt.Errorf("no pages provided")
	}

	bgBytes, err := io.ReadAll(background)
	if err != nil {
		return fmt.Errorf("failed to read background PDF: %w", err)
	}
	bgPath, err := writeTempPDF("rmfakecloud-bg-*.pdf", bgBytes)
	if err != nil {
		return fmt.Errorf("failed to buffer background PDF: %w", err)
	}
	defer os.Remove(bgPath)

	// Build a per-page stamp map: background page (1-based) -> annotation overlay.
	opts := &rmc.Options{UseLegacy: false}
	const stampDesc = "scale:1.0 rel, pos:c, rot:0"
	wmMap := make(map[int]*model.Watermark)
	var tmpStamps []string
	defer func() {
		for _, p := range tmpStamps {
			os.Remove(p)
		}
	}()

	for idx, data := range pages {
		if len(data) == 0 {
			continue
		}
		annPDF, err := rmc.ConvertFromBytes(data, rmc.FormatPDF, opts)
		if err != nil {
			return fmt.Errorf("failed to render v6 annotation page %d: %w", idx, err)
		}
		annPath, err := writeTempPDF("rmfakecloud-ann-*.pdf", annPDF)
		if err != nil {
			return fmt.Errorf("failed to buffer annotation page %d: %w", idx, err)
		}
		tmpStamps = append(tmpStamps, annPath)

		wm, err := api.PDFWatermark(annPath, stampDesc, true, false, types.POINTS)
		if err != nil {
			return fmt.Errorf("failed to build overlay for page %d: %w", idx, err)
		}
		wmMap[idx+1] = wm
	}

	// No annotations anywhere: return the background unchanged.
	if len(wmMap) == 0 {
		_, err = io.Copy(output, bytes.NewReader(bgBytes))
		return err
	}

	outPath, err := writeTempPDF("rmfakecloud-merged-*.pdf", nil)
	if err != nil {
		return fmt.Errorf("failed to create merged output: %w", err)
	}
	defer os.Remove(outPath)

	conf := model.NewDefaultConfiguration()
	if err := api.AddWatermarksMapFile(bgPath, outPath, wmMap, conf); err != nil {
		return fmt.Errorf("failed to overlay annotations onto background: %w", err)
	}

	merged, err := os.Open(outPath)
	if err != nil {
		return err
	}
	defer merged.Close()
	_, err = io.Copy(output, merged)
	return err
}

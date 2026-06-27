package export

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joagonca/rmc-go/parser"
)

// ExportToPDF exports a scene tree to PDF format
// If useLegacy is true, uses Inkscape via SVG conversion. Otherwise uses Cairo directly (default).
func ExportToPDF(tree *parser.SceneTree, w io.Writer, useLegacy bool) error {
	// Use legacy Inkscape renderer if requested
	if useLegacy {
		return exportToPDFInkscape(tree, w)
	}

	// Otherwise use native Cairo-based export (default)
	return ExportToPDFCairo(tree, w)
}

// exportToPDFInkscape exports a scene tree to PDF format via SVG conversion using Inkscape
func exportToPDFInkscape(tree *parser.SceneTree, w io.Writer) error {
	// Create temporary SVG
	svgBuf := &bytes.Buffer{}
	if err := ExportToSVG(tree, svgBuf); err != nil {
		return fmt.Errorf("failed to generate SVG: %w", err)
	}

	// Create temp files
	svgFile, err := os.CreateTemp("", "rmc-*.svg")
	if err != nil {
		return fmt.Errorf("failed to create temp SVG file: %w", err)
	}
	// Ensure cleanup happens in correct order: close before remove
	defer func() {
		svgFile.Close()
		os.Remove(svgFile.Name())
	}()

	if _, err := svgFile.Write(svgBuf.Bytes()); err != nil {
		return fmt.Errorf("failed to write SVG: %w", err)
	}
	svgFile.Close()

	pdfFile, err := os.CreateTemp("", "rmc-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp PDF file: %w", err)
	}
	pdfName := pdfFile.Name()
	pdfFile.Close()
	// Remove PDF temp file after we're done reading it
	defer os.Remove(pdfName)

	// Convert with inkscape
	cmd := exec.Command("inkscape", svgFile.Name(), "--export-filename", pdfName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("inkscape conversion failed: %w\n"+
			"  Ensure 'inkscape' is installed and available in PATH\n"+
			"  Install: https://inkscape.org/release/\n"+
			"  Or use SVG output with: -t svg", err)
	}

	// Read and write PDF
	pdfData, err := os.ReadFile(pdfName)
	if err != nil {
		return fmt.Errorf("failed to read PDF: %w", err)
	}

	if _, err := w.Write(pdfData); err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	return nil
}

// ExportToMultipagePDF exports multiple scene trees to a multipage PDF format
// If useLegacy is true, uses Inkscape via SVG conversion. Otherwise uses Cairo directly (default).
func ExportToMultipagePDF(trees []*parser.SceneTree, w io.Writer, useLegacy bool) error {
	if len(trees) == 0 {
		return fmt.Errorf("no scene trees provided")
	}

	// Use legacy Inkscape renderer if requested
	if useLegacy {
		return exportToMultipagePDFInkscape(trees, w)
	}

	// Otherwise use native Cairo-based export (default)
	return ExportToMultipagePDFCairo(trees, w)
}

// exportToMultipagePDFInkscape exports multiple scene trees to a multipage PDF via SVG conversion using Inkscape
func exportToMultipagePDFInkscape(trees []*parser.SceneTree, w io.Writer) error {
	// Create temporary directory for intermediate files
	tempDir, err := os.MkdirTemp("", "rmc-multipage-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate SVG and PDF for each page
	var pdfFiles []string
	for i, tree := range trees {
		// Generate SVG
		svgBuf := &bytes.Buffer{}
		if err := ExportToSVG(tree, svgBuf); err != nil {
			return fmt.Errorf("failed to generate SVG for page %d: %w", i+1, err)
		}

		// Write SVG to temp file
		svgPath := filepath.Join(tempDir, fmt.Sprintf("page_%03d.svg", i))
		if err := os.WriteFile(svgPath, svgBuf.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write temp SVG for page %d: %w", i+1, err)
		}

		// Convert SVG to PDF using Inkscape
		pdfPath := filepath.Join(tempDir, fmt.Sprintf("page_%03d.pdf", i))
		cmd := exec.Command("inkscape", svgPath, "--export-filename", pdfPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("inkscape conversion failed for page %d: %w\n"+
				"  Ensure 'inkscape' is installed and available in PATH\n"+
				"  Install: https://inkscape.org/release/", i+1, err)
		}

		pdfFiles = append(pdfFiles, pdfPath)
	}

	// Merge PDFs using pdfunite (part of poppler-utils)
	// Alternative: gs (Ghostscript) if pdfunite is not available
	outputPdfPath := filepath.Join(tempDir, "output.pdf")

	// Try pdfunite first
	args := append([]string{}, pdfFiles...)
	args = append(args, outputPdfPath)
	cmd := exec.Command("pdfunite", args...)
	err = cmd.Run()

	if err != nil {
		// Try Ghostscript as fallback
		gsArgs := []string{
			"-dBATCH", "-dNOPAUSE", "-q", "-sDEVICE=pdfwrite",
			"-sOutputFile=" + outputPdfPath,
		}
		gsArgs = append(gsArgs, pdfFiles...)
		cmd = exec.Command("gs", gsArgs...)
		if err = cmd.Run(); err != nil {
			return fmt.Errorf("PDF merging failed (install pdfunite or ghostscript): %w\n"+
				"  Ubuntu/Debian: sudo apt-get install poppler-utils\n"+
				"  macOS: brew install poppler\n"+
				"  Or: brew install ghostscript", err)
		}
	}

	// Read merged PDF and write to output
	pdfData, err := os.ReadFile(outputPdfPath)
	if err != nil {
		return fmt.Errorf("failed to read merged PDF: %w", err)
	}

	if _, err := w.Write(pdfData); err != nil {
		return fmt.Errorf("failed to write PDF output: %w", err)
	}

	return nil
}

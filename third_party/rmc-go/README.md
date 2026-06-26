# rmc-go

A Go implementation for converting reMarkable tablet v6 format files (`.rm`) to PDF and SVG.

This began as a port of the Python [rmc](https://github.com/ricklupton/rmc) tool, which uses [rmscene](https://github.com/ricklupton/rmscene) to read the reMarkable v6 file format, but was already extended in functionality.

**rmc-go can be used both as a command-line tool and as a Go library.**

## Features

- Read reMarkable v6 format files (software version 3+)
- Export to SVG format
- Export to PDF format
  - Default: direct PDF rendering using Cairo (requires CGo build)
  - Legacy: via Inkscape (requires Inkscape installation)
- Multipage PDF support: combine multiple .rm files from a folder into a single PDF
- Handles strokes/drawings with different pen types and colors
- Support for all pen colors including highlights and shaders
- Command-line interface

## Installation

### Prerequisites

- Go 1.25 or later
- Make (for building with Makefile)
- **For PDF export (choose one):**
  - Cairo development libraries + pkg-config (for default PDF export)
  - Inkscape (for legacy PDF export via SVG with `--legacy` flag)
- **For multipage PDF (legacy Inkscape method only):**
  - `pdfunite` (from poppler-utils) or `ghostscript` for merging PDFs

### Build from source

#### Standard build (with Cairo PDF support)

```bash
# Install Cairo libraries first:
# macOS: brew install cairo pkg-config
# Ubuntu/Debian: sudo apt-get install libcairo2-dev
# Fedora: sudo dnf install cairo-devel

git clone <repository-url>
cd rmc-go
make build-cairo
```

This creates the `rmc` binary with native Cairo PDF export (default) and legacy Inkscape support.

#### Build without Cairo (Inkscape-only)

If you don't want to install Cairo dependencies:

```bash
git clone <repository-url>
cd rmc-go
make build
```

This creates the `rmc` binary with Inkscape-based PDF export only. Note: PDF export will only work with the `--legacy` flag.

### As a Go Library

To use rmc-go as a library in your Go application:

```bash
go get github.com/joagonca/rmc-go
```

Then import it in your code:

```go
import "github.com/joagonca/rmc-go"
```

See the [Library Usage](#library-usage) section below for examples.

## Usage

### Command-Line Interface

#### Export to PDF

```bash
# Default: using Cairo (requires build-cairo)
./rmc file.rm -o output.pdf

# Legacy: using Inkscape (requires Inkscape installed)
./rmc file.rm -o output.pdf --legacy
```

#### Export to SVG

```bash
./rmc file.rm -o output.svg
```

#### Multipage PDF from folder

```bash
# Combine all .rm files in a folder into a single multipage PDF
# With .content file for reliable page ordering (default: Cairo renderer)
./rmc folder/ -o output.pdf --content folder.content

# Without .content file (uses modification time - may be unreliable)
./rmc folder/ -o output.pdf

# With legacy Inkscape renderer
./rmc folder/ -o output.pdf --content folder.content --legacy
```

**Important:** When using folders without a `.content` file, pages are ordered by file modification time, which may produce incorrect ordering if pages were edited after creation. Use the `--content` flag with a reMarkable `.content` file for reliable page ordering.

**Note:** Multipage output is only supported for PDF format. Attempting to export a folder to SVG will result in an error.

#### Export to stdout

```bash
./rmc file.rm -t svg > output.svg
./rmc file.rm -t pdf > output.pdf
```

#### Command-line options

```
Usage:
  rmc [input.rm|folder] [flags]

Flags:
      --content string  Path to .content file for page ordering (only used with folders)
  -h, --help            help for rmc
      --legacy          Use legacy Inkscape renderer for PDF export (requires Inkscape)
  -o, --output string   Output file (default: stdout)
  -t, --type string     Output type: svg or pdf (default: guess from filename)
```

**Input:**
- Single `.rm` file: Exports the file to the specified format
- Folder: Combines all `.rm` files in the folder into a multipage PDF (only PDF format supported)

**Page Ordering:**
- With `--content` flag: Uses the `.content` JSON file to determine correct page order
- Without `--content` flag: Falls back to file modification time (may be unreliable if pages edited after creation)

### Library Usage

rmc-go can be imported and used as a library in your Go applications. The package provides both high-level convenience functions and low-level APIs for fine-grained control.

#### Quick Start

```go
package main

import (
    "log"
    "github.com/joagonca/rmc-go"
)

func main() {
    // Simple file-to-file conversion
    err := rmc.ConvertFile("input.rm", "output.pdf", nil)
    if err != nil {
        log.Fatal(err)
    }
}
```

#### Convert from Binary Data

```go
// Read .rm file into memory
rmData, err := os.ReadFile("input.rm")
if err != nil {
    log.Fatal(err)
}

// Convert to PDF bytes
pdfData, err := rmc.ConvertFromBytes(rmData, rmc.FormatPDF, nil)
if err != nil {
    log.Fatal(err)
}

// Use pdfData as needed (write to file, send over HTTP, etc.)
os.WriteFile("output.pdf", pdfData, 0644)
```

#### Convert with Options

```go
opts := &rmc.Options{
    UseLegacy: true, // Use Inkscape renderer instead of Cairo
}

err := rmc.ConvertFile("input.rm", "output.pdf", opts)
if err != nil {
    log.Fatal(err)
}
```

#### Available Functions

**Single Page Conversion:**
- `ConvertFile(inputPath, outputPath, opts)` - Convert a file on disk
- `Convert(reader, writer, format, opts)` - Convert using io.Reader/Writer
- `ConvertFromBytes(data, format, opts)` - Convert from byte slice to byte slice
- `ConvertToBytes(data, format, opts)` - Alias for ConvertFromBytes
- `ConvertFileToBytes(inputPath, format, opts)` - Read file and convert to bytes
- `ConvertBytesToFile(data, outputPath, format, opts)` - Convert bytes and write to file

**Multipage PDF Conversion:**
- `ConvertFiles(inputPaths, outputPath, opts)` - Convert multiple files to multipage PDF
- `ConvertMultipleFromBytes(pages, opts)` - Convert multiple byte slices to multipage PDF
- `ConvertFilesToBytes(inputPaths, opts)` - Read multiple files and convert to multipage PDF bytes
- `ConvertMultipleBytesToFile(pages, outputPath, opts)` - Convert multiple byte slices and write to PDF file

#### Multipage PDF Examples

```go
// Convert multiple files to a multipage PDF
files := []string{"page1.rm", "page2.rm", "page3.rm"}
err := rmc.ConvertFiles(files, "output.pdf", nil)
if err != nil {
    log.Fatal(err)
}

// Convert from byte slices (e.g., from database or HTTP requests)
pages := [][]byte{page1Data, page2Data, page3Data}
pdfData, err := rmc.ConvertMultipleFromBytes(pages, nil)
if err != nil {
    log.Fatal(err)
}
os.WriteFile("output.pdf", pdfData, 0644)
```

#### Low-Level API

For more control, you can use the `parser` and `export` packages directly:

```go
import (
    "os"
    "github.com/joagonca/rmc-go/parser"
    "github.com/joagonca/rmc-go/export"
)

func main() {
    // Parse .rm file
    f, _ := os.Open("input.rm")
    tree, err := parser.ReadSceneTree(f)
    f.Close()
    if err != nil {
        log.Fatal(err)
    }

    // Export to PDF
    out, _ := os.Create("output.pdf")
    defer out.Close()

    useLegacy := false
    err = export.ExportToPDF(tree, out, useLegacy)
    if err != nil {
        log.Fatal(err)
    }
}
```

See `example_library_usage.go` for complete working examples.

## Development

### Building

Use the Makefile for all build operations:

```bash
# Build the rmc binary (without Cairo)
make build

# Build with native Cairo PDF support
make build-cairo

# Run integration tests with test files
make test

# Run Go unit tests
make test-unit

# Clean build artifacts and test outputs
make clean

# Show all available targets
make help
```

### Testing

The project includes test `.rm` files in the `tests/` directory. Run `make test` to verify that both SVG and PDF export work correctly with these files. Test outputs are saved to `test_output/` for inspection.

## Project Structure

```
rmc-go/
├── cmd/rmc-go/          # CLI application
│   └── main.go                # Main entry point with --legacy flag support
├── parser/              # v6 file format parser (public API)
│   ├── datastream.go          # Binary data stream reader
│   ├── block_reader.go        # Tagged block reader
│   ├── limited_reader.go      # Limited reader utility
│   ├── scene_stream.go        # Scene block parser
│   ├── text.go                # Text document processing
│   ├── content.go             # Content file parsing
│   └── types.go               # Data structures
├── export/              # Export functionality (public API)
│   ├── svg.go                 # SVG export
│   ├── pen.go                 # Pen rendering (shared by SVG/PDF)
│   ├── pdf.go                 # PDF export (Inkscape method)
│   ├── pdf_cairo.go           # Native PDF export using Cairo (build tag: cairo)
│   └── pdf_cairo_stub.go      # Stub for builds without Cairo
├── rmc.go               # High-level convenience API for library usage
├── example_library_usage.go   # Example code for library users
├── tests/               # Test .rm files
├── Makefile             # Build automation (build, build-cairo targets)
├── go.mod
└── README.md
```

## How it Works

The reMarkable v6 file format is a binary format that contains:

1. **Header**: File format identifier
2. **Blocks**: Tagged blocks containing different types of data:
   - Scene tree blocks (layers/groups)
   - Scene item blocks (lines, text, etc.)
   - Line/stroke data with points, pressure, speed
   - Text blocks with formatting

This implementation:

1. **Parses** the binary format using a DataStream and TaggedBlockReader
2. **Builds** a scene tree with groups (layers) and items (strokes/text)
3. **Exports** to output formats:
   - **SVG**: Direct rendering of strokes and text with appropriate pen styles
   - **PDF (Cairo)**: Direct rendering to PDF using Cairo graphics library (default, requires Cairo build)
   - **PDF (Inkscape)**: Converts SVG to PDF using Inkscape (legacy, requires `--legacy` flag)

## Supported Pen Types

- Ballpoint (v1 & v2)
- Fineliner (v1 & v2)
- Marker (v1 & v2)
- Pencil (v1 & v2)
- Mechanical Pencil (v1 & v2)
- Paintbrush/Brush (v1 & v2)
- Highlighter (v1 & v2)
- Eraser & Eraser Area
- Calligraphy
- Shader

## Supported Colors

### Standard Colors
- Black, Gray, White
- Yellow, Green, Pink, Blue, Red
- Cyan, Magenta

### Highlight Colors (6 variants)
- Yellow, Blue, Pink, Orange, Green, Gray

### Shader Colors (8 variants)
- Gray, Orange, Magenta, Blue, Red, Green, Yellow, Cyan

All pen colors are rendered with accurate RGB values and appropriate opacity for highlighters and shaders.

## PDF Export Methods

This tool supports two methods for PDF export:

### Cairo Method (Default)

- Renders PDF directly using Cairo graphics library
- **Pros:** No external dependencies at runtime, faster rendering
- **Cons:** Requires CGo and Cairo libraries at build time
- **Usage:** `./rmc file.rm -o output.pdf`
- **Build:** `make build-cairo`

### Inkscape Method (Legacy)

- Converts to SVG first, then uses Inkscape to generate PDF
- **Pros:** No CGo dependencies, easier to build and deploy
- **Cons:** Requires Inkscape to be installed on the system, slower
- **Usage:** `./rmc file.rm -o output.pdf --legacy`
- **Build:** `make build`

Both methods produce equivalent PDF output with full support for all pen types, colors, and text rendering.

## Limitations

- Character-level text formatting (bold/italic within paragraphs) is not implemented
- GlyphRange (PDF highlights) are not yet rendered
- Some newer block types may not be supported
- Parser is tolerant of errors and will skip unrecognized blocks

## Acknowledgements

This project is based on:

- [rmc](https://github.com/ricklupton/rmc) - Python converter for reMarkable files
- [rmscene](https://github.com/ricklupton/rmscene) - Python library for reading v6 format
- [ddvk's reader](https://github.com/ddvk/reader) - helped understand the file format

## License

MIT License (same as the original rmc project)

## Development Status

This is a work-in-progress implementation. The core functionality for reading stroke data and exporting to SVG/PDF is working, including:

- ✅ All pen types and colors (including highlights and shaders)
- ✅ Pressure-sensitive stroke rendering
- ✅ Layer support
- ✅ Text rendering with paragraph styles
- ⚠️  Some newer block types may not be fully supported

### Recent Updates

- **Text Rendering**: Implemented full text rendering support with paragraph styles (heading, plain, bold, bullet, checkbox). Text from .rm files is now properly rendered to SVG output with correct positioning and styling.
- **Highlight & Shader Support**: Added full support for all 14 highlight and shader color variants, including accurate RGBA color parsing from v6 format files.

Contributions and bug reports are welcome!

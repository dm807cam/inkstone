# Using rmc-go as a Library

rmc-go can be used both as a command-line tool and as a Go library. This document provides detailed examples of library usage.

## Installation

```bash
go get github.com/joagonca/rmc-go
```

## Basic Usage

### Simple File Conversion

```go
package main

import (
    "log"
    "github.com/joagonca/rmc-go"
)

func main() {
    // Convert .rm file to PDF
    err := rmc.ConvertFile("input.rm", "output.pdf", nil)
    if err != nil {
        log.Fatal(err)
    }

    // Convert .rm file to SVG
    err = rmc.ConvertFile("input.rm", "output.svg", nil)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Working with Binary Data

### Convert from Byte Slice

```go
package main

import (
    "log"
    "os"
    "github.com/joagonca/rmc-go"
)

func main() {
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

    // Use pdfData as needed
    // - Send over HTTP
    // - Store in database
    // - Write to file
    os.WriteFile("output.pdf", pdfData, 0644)
}
```

### Using io.Reader and io.Writer

```go
package main

import (
    "bytes"
    "log"
    "github.com/joagonca/rmc-go"
)

func main() {
    // Your .rm data from somewhere
    rmData := []byte("...")

    input := bytes.NewReader(rmData)
    output := &bytes.Buffer{}

    err := rmc.Convert(input, output, rmc.FormatPDF, nil)
    if err != nil {
        log.Fatal(err)
    }

    pdfData := output.Bytes()
    // Use pdfData...
}
```

## Options

### Using Legacy Inkscape Renderer

```go
package main

import (
    "log"
    "github.com/joagonca/rmc-go"
)

func main() {
    opts := &rmc.Options{
        UseLegacy: true, // Use Inkscape instead of Cairo
    }

    err := rmc.ConvertFile("input.rm", "output.pdf", opts)
    if err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### High-Level Functions

#### Single Page Conversion

##### `ConvertFile(inputPath, outputPath string, opts *Options) error`

Convert a .rm file on disk to the specified output format.
- Format is inferred from output file extension
- Uses default options if `opts` is `nil`

##### `Convert(input io.Reader, output io.Writer, format Format, opts *Options) error`

Convert from a reader to a writer.
- Useful for streaming or in-memory conversions
- Format must be specified explicitly (`rmc.FormatPDF` or `rmc.FormatSVG`)

##### `ConvertFromBytes(data []byte, format Format, opts *Options) ([]byte, error)`

Convert from byte slice to byte slice (fully in-memory).
- Most convenient for working with binary data
- Returns converted data as byte slice

##### `ConvertToBytes(data []byte, format Format, opts *Options) ([]byte, error)`

Alias for `ConvertFromBytes` (same functionality).

##### `ConvertFileToBytes(inputPath string, format Format, opts *Options) ([]byte, error)`

Read a file and convert to bytes in one step.

##### `ConvertBytesToFile(data []byte, outputPath string, format Format, opts *Options) error`

Convert bytes and write to file in one step.

#### Multipage PDF Conversion

##### `ConvertFiles(inputPaths []string, outputPath string, opts *Options) error`

Convert multiple ordered .rm files to a multipage PDF.
- Files are processed in the order they appear in the slice
- Only PDF format is supported for multipage output
- Uses default options if `opts` is `nil`

##### `ConvertMultipleFromBytes(pages [][]byte, opts *Options) ([]byte, error)`

Convert multiple ordered .rm files from binary data to a multipage PDF.
- Pages are processed in the order they appear in the slice
- Returns the multipage PDF as a byte slice
- Only PDF format is supported for multipage output

##### `ConvertFilesToBytes(inputPaths []string, opts *Options) ([]byte, error)`

Read multiple ordered .rm files and convert to a multipage PDF as bytes.
- Combines file reading and conversion in one step
- Files are processed in the order they appear in the slice

##### `ConvertMultipleBytesToFile(pages [][]byte, outputPath string, opts *Options) error`

Convert multiple ordered .rm files from binary data to a multipage PDF and write to file.
- Combines conversion and file writing in one step
- Pages are processed in the order they appear in the slice

### Types

#### `Format`

```go
type Format string

const (
    FormatPDF Format = "pdf"
    FormatSVG Format = "svg"
)
```

#### `Options`

```go
type Options struct {
    UseLegacy bool // Use Inkscape renderer instead of Cairo (default: false)
}
```

## Low-Level API

For fine-grained control, use the `parser` and `export` packages directly:

```go
package main

import (
    "log"
    "os"
    "github.com/joagonca/rmc-go/parser"
    "github.com/joagonca/rmc-go/export"
)

func main() {
    // Parse .rm file
    f, err := os.Open("input.rm")
    if err != nil {
        log.Fatal(err)
    }

    tree, err := parser.ReadSceneTree(f)
    f.Close()
    if err != nil {
        log.Fatal(err)
    }

    // Export to PDF
    out, err := os.Create("output.pdf")
    if err != nil {
        log.Fatal(err)
    }
    defer out.Close()

    useLegacy := false
    err = export.ExportToPDF(tree, out, useLegacy)
    if err != nil {
        log.Fatal(err)
    }

    // Or export to SVG
    svgOut, _ := os.Create("output.svg")
    defer svgOut.Close()
    err = export.ExportToSVG(tree, svgOut)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Multipage PDF Examples

### Convert Multiple Files

```go
package main

import (
    "log"
    "github.com/joagonca/rmc-go"
)

func main() {
    // Files are processed in the order they appear
    files := []string{
        "page1.rm",
        "page2.rm",
        "page3.rm",
    }

    // Convert to multipage PDF
    err := rmc.ConvertFiles(files, "multipage_output.pdf", nil)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Created multipage PDF successfully")
}
```

### Convert from Byte Slices

```go
package main

import (
    "log"
    "os"
    "github.com/joagonca/rmc-go"
)

func main() {
    // Read pages (e.g., from database, HTTP requests, etc.)
    page1Data, _ := os.ReadFile("page1.rm")
    page2Data, _ := os.ReadFile("page2.rm")
    page3Data, _ := os.ReadFile("page3.rm")

    // Create ordered slice of pages
    pages := [][]byte{page1Data, page2Data, page3Data}

    // Convert to multipage PDF bytes
    pdfData, err := rmc.ConvertMultipleFromBytes(pages, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Write to file or send over network
    os.WriteFile("output.pdf", pdfData, 0644)

    // Or use convenience function to write directly
    err = rmc.ConvertMultipleBytesToFile(pages, "output2.pdf", nil)
    if err != nil {
        log.Fatal(err)
    }
}
```

### With Legacy Inkscape Renderer

```go
package main

import (
    "log"
    "github.com/joagonca/rmc-go"
)

func main() {
    files := []string{"page1.rm", "page2.rm", "page3.rm"}

    opts := &rmc.Options{
        UseLegacy: true, // Use Inkscape instead of Cairo
    }

    err := rmc.ConvertFiles(files, "multipage_legacy.pdf", opts)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Use Cases

### HTTP Server Example

```go
package main

import (
    "io"
    "net/http"
    "github.com/joagonca/rmc-go"
)

func convertHandler(w http.ResponseWriter, r *http.Request) {
    // Read .rm file from request body
    rmData, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Convert to PDF
    pdfData, err := rmc.ConvertFromBytes(rmData, rmc.FormatPDF, nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Send PDF back
    w.Header().Set("Content-Type", "application/pdf")
    w.Write(pdfData)
}

func main() {
    http.HandleFunc("/convert", convertHandler)
    http.ListenAndServe(":8080", nil)
}
```

### Batch Conversion

```go
package main

import (
    "log"
    "os"
    "path/filepath"
    "github.com/joagonca/rmc-go"
)

func main() {
    files, _ := filepath.Glob("*.rm")

    for _, file := range files {
        outputFile := file[:len(file)-3] + ".pdf"

        err := rmc.ConvertFile(file, outputFile, nil)
        if err != nil {
            log.Printf("Failed to convert %s: %v", file, err)
            continue
        }

        log.Printf("Converted %s -> %s", file, outputFile)
    }
}
```

### Multipage HTTP Server

```go
package main

import (
    "encoding/json"
    "io"
    "net/http"
    "github.com/joagonca/rmc-go"
)

// Request format: array of base64-encoded .rm files
type MultipageRequest struct {
    Pages [][]byte `json:"pages"`
}

func multipageConvertHandler(w http.ResponseWriter, r *http.Request) {
    // Parse JSON request
    var req MultipageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if len(req.Pages) == 0 {
        http.Error(w, "no pages provided", http.StatusBadRequest)
        return
    }

    // Convert to multipage PDF
    pdfData, err := rmc.ConvertMultipleFromBytes(req.Pages, nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Send PDF back
    w.Header().Set("Content-Type", "application/pdf")
    w.Header().Set("Content-Disposition", "attachment; filename=\"output.pdf\"")
    w.Write(pdfData)
}

func main() {
    http.HandleFunc("/convert-multipage", multipageConvertHandler)
    http.ListenAndServe(":8080", nil)
}
```

## Complete Example

See `example_library_usage.go` in the root of the repository for a complete, runnable example demonstrating all features.

## Requirements

- Go 1.21 or later
- For PDF export: Cairo libraries (unless using `UseLegacy: true`)
  - macOS: `brew install cairo pkg-config`
  - Ubuntu/Debian: `sudo apt-get install libcairo2-dev`
  - Fedora: `sudo dnf install cairo-devel`
- For legacy Inkscape PDF export: Inkscape installed

## Building Your Application

If using the default Cairo renderer, build with CGo enabled:

```bash
go build -tags cairo yourapp.go
```

Or use the standard build (which will fall back to the Cairo stub if not available):

```bash
go build yourapp.go
```

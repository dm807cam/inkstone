package hwr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Rendering parameters for turning strokes into an image a vision model can read.
const (
	// renderMaxDim caps the longer side of the rendered image (in px). The reMarkable
	// screen is 1404px wide; staying near that keeps detail without wasting vision tokens.
	renderMaxDim = 1280.0
	// renderPadding is the white margin around the strokes, in output px.
	renderPadding = 24.0
	// renderStrokeRadius is half the stroke thickness, in output px.
	renderStrokeRadius = 2.0
	// renderMaxScale prevents tiny selections from being blown up into a blurry mess.
	renderMaxScale = 4.0
)

// iinkStroke is a single pen stroke: parallel arrays of point coordinates. The tablet also
// sends timestamps (t) and pressure (p), which we don't need to redraw the shape.
type iinkStroke struct {
	X []float64 `json:"x"`
	Y []float64 `json:"y"`
}

// iinkBatch is the subset of the MyScript iink-batch request the tablet POSTs that we need to
// reconstruct the handwriting. Unknown fields (configuration, contentType, DPI, ...) are ignored.
type iinkBatch struct {
	StrokeGroups []struct {
		Strokes []iinkStroke `json:"strokes"`
	} `json:"strokeGroups"`
}

// allStrokes flattens the stroke groups into a single slice.
func (b *iinkBatch) allStrokes() []iinkStroke {
	var out []iinkStroke
	for _, g := range b.StrokeGroups {
		out = append(out, g.Strokes...)
	}
	return out
}

// renderStrokesPNG parses an iink-batch payload and rasterizes its strokes to a PNG: black
// ink on a white background, scaled so the longer side is at most renderMaxDim.
func renderStrokesPNG(iinkJSON []byte) ([]byte, error) {
	var batch iinkBatch
	if err := json.Unmarshal(iinkJSON, &batch); err != nil {
		return nil, fmt.Errorf("parse iink batch: %w", err)
	}
	strokes := batch.allStrokes()
	if len(strokes) == 0 {
		return nil, fmt.Errorf("no strokes in iink batch")
	}

	minX, minY, maxX, maxY, points := bounds(strokes)
	if points == 0 {
		return nil, fmt.Errorf("no points in strokes")
	}

	contentW := math.Max(maxX-minX, 1)
	contentH := math.Max(maxY-minY, 1)
	scale := math.Min(renderMaxScale, renderMaxDim/math.Max(contentW, contentH))

	imgW := int(contentW*scale + 2*renderPadding)
	imgH := int(contentH*scale + 2*renderPadding)
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	fill(img, color.White)

	tx := func(x, y float64) (float64, float64) {
		return (x-minX)*scale + renderPadding, (y-minY)*scale + renderPadding
	}

	black := color.RGBA{A: 255}
	for _, s := range strokes {
		n := min(len(s.X), len(s.Y))
		if n == 0 {
			continue
		}
		px, py := tx(s.X[0], s.Y[0])
		stampDisc(img, px, py, renderStrokeRadius, black)
		for i := 1; i < n; i++ {
			qx, qy := tx(s.X[i], s.Y[i])
			drawThickSegment(img, px, py, qx, qy, renderStrokeRadius, black)
			px, py = qx, qy
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

// bounds returns the coordinate extent and total point count across all strokes.
func bounds(strokes []iinkStroke) (minX, minY, maxX, maxY float64, points int) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, s := range strokes {
		n := min(len(s.X), len(s.Y))
		for i := 0; i < n; i++ {
			x, y := s.X[i], s.Y[i]
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, y), math.Max(maxY, y)
			points++
		}
	}
	return
}

// drawThickSegment stamps discs along the segment so the line stays continuous and rounded.
func drawThickSegment(img *image.RGBA, x0, y0, x1, y1, r float64, col color.Color) {
	dist := math.Hypot(x1-x0, y1-y0)
	steps := int(dist) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		stampDisc(img, x0+(x1-x0)*t, y0+(y1-y0)*t, r, col)
	}
}

// stampDisc fills a filled circle of radius r centered at (cx, cy).
func stampDisc(img *image.RGBA, cx, cy, r float64, col color.Color) {
	r2 := r * r
	for dy := -int(r); dy <= int(r); dy++ {
		for dx := -int(r); dx <= int(r); dx++ {
			if float64(dx*dx+dy*dy) <= r2 {
				img.Set(int(cx)+dx, int(cy)+dy, col)
			}
		}
	}
}

// fill paints the whole image a single color.
func fill(img *image.RGBA, col color.Color) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.Set(x, y, col)
		}
	}
}

package export

import (
	"fmt"
	"math"

	"github.com/joagonca/rmc-go/parser"
)

type RGB struct {
	R, G, B int
}

var rmPalette = map[parser.PenColor]RGB{
	parser.ColorBlack:       {0, 0, 0},
	parser.ColorGray:        {144, 144, 144},
	parser.ColorWhite:       {255, 255, 255},
	parser.ColorYellow:      {251, 247, 25},
	parser.ColorGreen:       {0, 255, 0},
	parser.ColorPink:        {255, 192, 203},
	parser.ColorBlue:        {78, 105, 201},
	parser.ColorRed:         {179, 62, 57},
	parser.ColorGrayOverlap: {125, 125, 125},
	parser.ColorHighlight:   {255, 237, 117}, // Default highlight color (yellow)
	parser.ColorGreen2:      {161, 216, 125},
	parser.ColorCyan:        {139, 208, 229},
	parser.ColorMagenta:     {183, 130, 205},
	parser.ColorYellow2:     {247, 232, 81},
	// Note: Highlight and shader color variants are now read directly from .rm files as RGBA overrides
}

type pen struct {
	name           string
	baseWidth      float64
	baseColor      RGB
	segmentLength  int
	baseOpacity    float64
	strokeLinecap  string
	strokeOpacity  float64
	thicknessScale float64
}

func createPen(penType parser.Pen, color parser.PenColor, colorOverride *parser.RGBA, thicknessScale float64) *pen {
	var baseColor RGB

	// Use color override if available (for highlights/shaders), otherwise use palette
	if colorOverride != nil {
		baseColor = RGB{
			R: int(colorOverride.R),
			G: int(colorOverride.G),
			B: int(colorOverride.B),
		}
	} else {
		var ok bool
		baseColor, ok = rmPalette[color]
		if !ok {
			baseColor = RGB{0, 0, 0}
		}
	}

	p := &pen{
		baseColor:      baseColor,
		segmentLength:  1000,
		baseOpacity:    1.0,
		strokeLinecap:  "round",
		strokeOpacity:  1.0,
		thicknessScale: thicknessScale,
	}

	switch penType {
	case parser.PenBallpoint1, parser.PenBallpoint2:
		p.name = "Ballpoint"
		p.baseWidth = thicknessScale
		p.segmentLength = 5
	case parser.PenFineliner1, parser.PenFineliner2:
		p.name = "Fineliner"
		p.baseWidth = thicknessScale * 1.8
	case parser.PenMarker1, parser.PenMarker2:
		p.name = "Marker"
		p.baseWidth = thicknessScale
		p.segmentLength = 3
	case parser.PenPencil1, parser.PenPencil2:
		p.name = "Pencil"
		p.baseWidth = thicknessScale
		p.segmentLength = 2
	case parser.PenMechanicalPencil1, parser.PenMechanicalPencil2:
		p.name = "MechanicalPencil"
		p.baseWidth = thicknessScale * thicknessScale
		p.baseOpacity = 0.7
	case parser.PenPaintbrush1, parser.PenPaintbrush2:
		p.name = "Brush"
		p.baseWidth = thicknessScale
		p.segmentLength = 2
		p.strokeLinecap = "round"
	case parser.PenHighlighter1, parser.PenHighlighter2:
		p.name = "Highlighter"
		p.baseWidth = 15
		p.strokeLinecap = "square"
		p.baseOpacity = 0.3
		p.strokeOpacity = 0.2
	case parser.PenEraser:
		p.name = "Eraser"
		p.baseWidth = thicknessScale * 2
		p.strokeLinecap = "square"
		p.baseColor = rmPalette[parser.ColorWhite]
	case parser.PenEraserArea:
		p.name = "EraseArea"
		p.baseWidth = thicknessScale
		p.strokeLinecap = "square"
		p.baseOpacity = 0
	case parser.PenCalligraphy:
		p.name = "Calligraphy"
		p.baseWidth = thicknessScale
		p.segmentLength = 2
	case parser.PenShader:
		p.name = "Shader"
		p.baseWidth = 12
		p.strokeLinecap = "round"
		p.baseOpacity = 0.1
	default:
		p.name = "Unknown"
		p.baseWidth = thicknessScale
	}

	return p
}

func (p *pen) getSegmentColor(point parser.Point, lastWidth float64) string {
	switch p.name {
	case "Ballpoint":
		speed := float64(point.Speed) / 4.0
		pressure := float64(point.Pressure) / 255.0
		intensity := (0.1 * -(speed / 35.0)) + (1.2 * pressure) + 0.5
		intensity = clamp(intensity)
		// Apply intensity to the base color instead of always using gray
		factor := math.Min(math.Abs(intensity-1), 0.235) // max darkening of ~60/255
		r := int(float64(p.baseColor.R) * (1 - factor))
		g := int(float64(p.baseColor.G) * (1 - factor))
		b := int(float64(p.baseColor.B) * (1 - factor))
		return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)

	case "Brush":
		speed := float64(point.Speed) / 4.0
		pressure := float64(point.Pressure) / 255.0
		intensity := math.Pow(pressure, 1.5) - 0.2*(speed/50.0)
		intensity = clamp(intensity)
		// Apply intensity to darken/lighten the base color
		r := int(float64(p.baseColor.R) * intensity)
		g := int(float64(p.baseColor.G) * intensity)
		b := int(float64(p.baseColor.B) * intensity)
		return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)

	default:
		return fmt.Sprintf("rgb(%d,%d,%d)", p.baseColor.R, p.baseColor.G, p.baseColor.B)
	}
}

func (p *pen) getSegmentWidth(point parser.Point, lastWidth float64) float64 {
	speed := float64(point.Speed) / 4.0
	pressure := float64(point.Pressure) / 255.0
	width := float64(point.Width) / 4.0
	tilt := directionToTilt(point.Direction)

	switch p.name {
	case "Ballpoint":
		return (0.5 + pressure) + width - 0.5*(speed/50.0)

	case "Marker":
		return 0.9*(width-0.4*tilt) + (0.1 * lastWidth)

	case "Pencil":
		segWidth := 0.7 * ((((0.8 * p.baseWidth) + (0.5 * pressure)) * width) -
			(0.25 * math.Pow(tilt, 1.8)) - (0.6 * (speed / 50.0)))
		maxWidth := p.baseWidth * 10
		if segWidth > maxWidth {
			return maxWidth
		}
		return segWidth

	case "Brush":
		return 0.7 * (((1 + (1.4 * pressure)) * width) - (0.5 * tilt) - (speed / 50.0))

	case "Calligraphy":
		return 0.9*(((1+pressure)*width)-(0.3*tilt)) + (0.1 * lastWidth)

	default:
		return p.baseWidth
	}
}

func (p *pen) getSegmentOpacity(point parser.Point, lastWidth float64) float64 {
	speed := float64(point.Speed) / 4.0
	pressure := float64(point.Pressure) / 255.0

	switch p.name {
	case "Pencil":
		opacity := (0.1 * -(speed / 35.0)) + pressure
		return clamp(opacity) - 0.1

	default:
		return p.baseOpacity
	}
}

func directionToTilt(direction uint8) float64 {
	return float64(direction) * (math.Pi * 2) / 255.0
}

func clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

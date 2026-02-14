package opengraph

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/fogleman/gg"
	"github.com/robherley/snips.sh/internal/snips"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	imgWidth  = 1200
	imgHeight = 630
)

var (
	colorBackground = color.NRGBA{0x12, 0x13, 0x18, 0xFF}
	colorBorder     = color.NRGBA{0x27, 0x2D, 0x36, 0xFF}
	colorGray       = color.NRGBA{0x87, 0x8A, 0x8F, 0xFF}
	colorWhite      = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}

	themeColors = []color.NRGBA{
		{0x66, 0xAD, 0xFF, 0xFF}, // blue   hsl(212, 100%, 70%)
		{0xFF, 0x70, 0x82, 0xFF}, // red    hsl(356, 100%, 72%)
		{0xF0, 0xAC, 0x12, 0xFF}, // amber  hsl(42, 92%, 54%)
		{0x6D, 0xC7, 0x8B, 0xFF}, // green  hsl(134, 48%, 62%)
		{0x11, 0xD5, 0xB1, 0xFF}, // teal   hsl(171, 85%, 45%)
		{0xBD, 0x7E, 0xEB, 0xFF}, // purple hsl(278, 76%, 74%)
		{0xF0, 0x6B, 0x8E, 0xFF}, // pink   hsl(344, 88%, 71%)
	}

	colorPrimary = themeColors[0]
)

// Fonts holds the raw TTF data needed for image generation.
type Fonts struct {
	Regular     []byte
	Display     []byte
	DisplayLine []byte
}

// Renderer holds parsed fonts and a pre-loaded logo for generating OG images.
type Renderer struct {
	fontCode      font.Face
	fontTitle     font.Face
	fontTitleLine font.Face
	logo          image.Image
}

// NewRenderer parses fonts and prepares a reusable renderer.
func NewRenderer(fonts *Fonts, logo image.Image) (*Renderer, error) {
	parseFace := func(ttf []byte, size float64) (font.Face, error) {
		f, err := opentype.Parse(ttf)
		if err != nil {
			return nil, err
		}
		return opentype.NewFace(f, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
	}

	fontCode, err := parseFace(fonts.Regular, 36)
	if err != nil {
		return nil, err
	}
	fontTitle, err := parseFace(fonts.Display, 164)
	if err != nil {
		return nil, err
	}
	fontTitleLine, err := parseFace(fonts.DisplayLine, 164)
	if err != nil {
		return nil, err
	}

	return &Renderer{
		fontCode:      fontCode,
		fontTitle:     fontTitle,
		fontTitleLine: fontTitleLine,
		logo:          logo,
	}, nil
}

// GenerateImage creates a 1200x630 PNG open graph image for a snippet.
func (r *Renderer) GenerateImage(file *snips.File) ([]byte, error) {
	dc := gg.NewContext(imgWidth, imgHeight)

	dc.SetColor(colorBackground)
	dc.Clear()

	drawStripeBar(dc, 0, colorPrimary)

	barHeight := 36
	barY := imgHeight - barHeight
	drawStripeBar(dc, barY, colorBorder)

	bh := float64(barHeight)
	sw := 12.0
	step := 28.0
	lastStripeX := imgWidth - bh - 1*step
	firstStripeX := lastStripeX - float64(len(themeColors)-1)*step
	for i, c := range themeColors {
		sx := firstStripeX + float64(i)*step
		yf := float64(barY)
		dc.SetColor(c)
		dc.MoveTo(sx, yf+bh)
		dc.LineTo(sx+sw, yf+bh)
		dc.LineTo(sx+bh+sw, yf)
		dc.LineTo(sx+bh, yf)
		dc.ClosePath()
		dc.Fill()
	}

	titleX, titleY := 60.0, 95.0
	dc.SetFontFace(r.fontTitleLine)
	dc.SetColor(color.NRGBA{0x71, 0x71, 0x74, 0xFF})
	dc.DrawStringAnchored("snips.sh", titleX-10, titleY, 0, 0.5)
	dc.SetFontFace(r.fontTitle)
	dc.SetColor(colorWhite)
	dc.DrawStringAnchored("snips.sh", titleX, titleY, 0, 0.5)

	logoBounds := r.logo.Bounds()
	logoX := imgWidth - logoBounds.Dx() - 40
	logoY := (imgHeight - logoBounds.Dy()) / 2
	draw.Draw(
		dc.Image().(*image.RGBA),
		image.Rect(logoX, logoY, logoX+logoBounds.Dx(), logoY+logoBounds.Dy()),
		r.logo,
		image.Point{},
		draw.Over,
	)

	type token struct {
		text  string
		color color.NRGBA
	}

	props := []struct{ key, value string }{
		{"id", file.ID},
		{"type", strings.ToLower(file.Type)},
		{"size", humanize.Bytes(file.Size)},
		{"modified", humanize.Time(file.UpdatedAt)},
	}

	var lines [][]token
	lines = append(lines, []token{{"{", colorWhite}})
	for i, p := range props {
		comma := ","
		if i == len(props)-1 {
			comma = ""
		}
		lines = append(lines, []token{
			{"  ", colorWhite},
			{`"`, colorGray},
			{p.key, colorWhite},
			{`"`, colorGray},
			{": ", colorGray},
			{`"`, colorGray},
			{p.value, colorPrimary},
			{`"` + comma, colorGray},
		})
	}
	lines = append(lines, []token{{"}", colorWhite}})

	dc.SetFontFace(r.fontCode)
	y := 250.0
	lineHeight := 50.0
	for _, line := range lines {
		x := 60.0
		for _, tok := range line {
			dc.SetColor(tok.color)
			dc.DrawStringAnchored(tok.text, x, y, 0, 0.5)
			w, _ := dc.MeasureString(tok.text)
			x += w
		}
		y += lineHeight
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// drawStripeBar draws a diagonal stripe bar across the full width at the given y position.
func drawStripeBar(dc *gg.Context, y int, c color.NRGBA) {
	drawStripeSection(dc, y, c, -36, imgWidth+36)
}

// drawStripeSection draws diagonal stripes within a horizontal x-range at the given y position.
func drawStripeSection(dc *gg.Context, y int, c color.NRGBA, xMin, xMax float64) {
	bh := 36.0   // bar height
	sw := 10.0   // stripe width
	step := 28.0 // stripe spacing
	yf := float64(y)

	dc.SetColor(c)

	for sx := -bh; sx < imgWidth+bh; sx += step {
		// bottom-left of the parallelogram
		blX := sx
		// top-right of the parallelogram
		trX := sx + bh + sw

		// skip stripes that don't overlap with our section
		if trX < xMin || blX > xMax {
			continue
		}

		dc.MoveTo(sx, yf+bh)
		dc.LineTo(sx+sw, yf+bh)
		dc.LineTo(sx+bh+sw, yf)
		dc.LineTo(sx+bh, yf)
		dc.ClosePath()
		dc.Fill()
	}
}

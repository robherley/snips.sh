package styles

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

// bayer is a 4x4 ordered-dither threshold matrix.
var bayer = [4][4]float64{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

// brailleBits maps a dot position within a braille cell's 2x4 grid to its bit
// in the Unicode braille pattern block (U+2800-U+28FF).
var brailleBits = [4][2]rune{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// Backdrop fills a width x height area with a braille-dot texture that
// thickens toward the edges, ordered-dithered at the braille sub-dot
// resolution (2x4 dots per cell) so the density ramps smoothly. Each dot is a
// pure function of its coordinates, so the pattern is stable across renders.
func Backdrop(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	// braille sub-dots are roughly square, so no cell aspect correction
	cx, cy := float64(width*2)/2, float64(height*4)/2
	maxDist := math.Hypot(cx, cy)

	dots := strings.Builder{}
	for y := range height {
		if y > 0 {
			dots.WriteByte('\n')
		}
		for x := range width {
			cell := rune(0x2800)
			for dy := range 4 {
				for dx := range 2 {
					px, py := x*2+dx, y*4+dy
					dist := math.Hypot(float64(px)-cx, float64(py)-cy) / maxDist
					// cap density so the edges stay speckled rather than solid
					if dist*0.75 > (bayer[py%4][px%4]+0.5)/16 {
						cell |= brailleBits[dy][dx]
					}
				}
			}
			dots.WriteRune(cell)
		}
	}

	return lipgloss.NewStyle().Foreground(Colors.Muted).Render(dots.String())
}

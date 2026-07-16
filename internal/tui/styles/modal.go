package styles

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// ModalMinWidth keeps modal windows from collapsing around short content.
const ModalMinWidth = 44

// Modal centers a window over a Backdrop-filled canvas. The window has no
// border: the body is expected to come from ModalBody, whose header bar marks
// the top edge, and the window layer blanks the backdrop behind the rest.
func Modal(width, height int, body string) string {
	x := max((width-lipgloss.Width(body))/2, 0)
	y := max((height-lipgloss.Height(body))/2, 0)

	// layers must go through a Compositor: composing a Layer directly onto
	// the Canvas ignores its X/Y offset and blanks the cells around it
	return lipgloss.NewCanvas(width, height).
		Compose(lipgloss.NewCompositor(
			lipgloss.NewLayer(Backdrop(width, height)).Z(0),
			lipgloss.NewLayer(body).X(x).Y(y).Z(1),
		)).
		Render()
}

// ModalBody assembles a modal page: a full-width accent-colored header bar
// with the title on the left and an esc hint on the right, a blank line, then
// the rows.
func ModalBody(accent color.Color, title string, rows ...string) string {
	width := ModalMinWidth
	for _, row := range rows {
		width = max(width, lipgloss.Width(row))
	}

	onAccent := lipgloss.NewStyle().Foreground(Colors.Black).Background(accent)
	gap := max(width-lipgloss.Width(title)-len("[esc]"), 1)
	bar := onAccent.Bold(true).Render("  "+title) +
		onAccent.Render(strings.Repeat(" ", gap)+"[esc]  ")

	body := lipgloss.NewStyle().
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Top, rows...))

	return lipgloss.JoinVertical(lipgloss.Top, bar, body)
}

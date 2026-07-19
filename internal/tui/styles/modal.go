package styles

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const ModalMinWidth = 44

func Dim(s string) string {
	return lipgloss.NewStyle().Foreground(Colors.Dim).Render(ansi.Strip(s))
}

func Modal(width, height int, under, body string) string {
	x := max((width-lipgloss.Width(body))/2, 0)
	y := max((height-lipgloss.Height(body))/2, 0)

	// layers must go through a Compositor: composing a Layer directly onto
	// the Canvas ignores its X/Y offset and blanks the cells around it
	return lipgloss.NewCanvas(width, height).
		Compose(lipgloss.NewCompositor(
			lipgloss.NewLayer(Dim(under)).Z(0),
			lipgloss.NewLayer(body).X(x).Y(y).Z(1),
		)).
		Render()
}

func ModalBody(accent color.Color, title string, rows ...string) string {
	width := max(ModalMinWidth, lipgloss.Width(title))
	for _, row := range rows {
		width = max(width, lipgloss.Width(row))
	}

	body := lipgloss.NewStyle().
		Padding(1, 2).
		Render(lipgloss.NewStyle().Width(width).Render(lipgloss.JoinVertical(lipgloss.Top, rows...)))

	return Frame(accent, BC(accent, title), body)
}

func Frame(accent color.Color, title, body string) string {
	pad := max(lipgloss.Width(body)-lipgloss.Width(title)-2, 0)
	titleRow := "  " + title + strings.Repeat(" ", pad)
	titleBar := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true, true, false, true).
		BorderForeground(accent).
		Render(titleRow)

	frame := lipgloss.RoundedBorder()
	frame.TopLeft, frame.TopRight = "├", "┤"

	window := lipgloss.NewStyle().
		Border(frame).
		BorderForeground(accent).
		Render(body)

	return lipgloss.JoinVertical(lipgloss.Top, titleBar, window)
}

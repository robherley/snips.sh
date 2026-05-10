package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

var (
	Colors = struct {
		Primary color.Color
		Green   color.Color
		Red     color.Color
		Cyan    color.Color
		Yellow  color.Color
		Blue    color.Color
		Pink    color.Color
		Purple  color.Color
		White   color.Color
		Muted   color.Color
		Black   color.Color
	}{
		Primary: lipgloss.Color("#0ac5b2"),
		Green:   lipgloss.Color("#63c174"),
		Red:     lipgloss.Color("#ff6368"),
		Yellow:  lipgloss.Color("#f1a10d"),
		Blue:    lipgloss.Color("#52a9ff"),
		Pink:    lipgloss.Color("#f76191"),
		Purple:  lipgloss.Color("#bf7af0"),
		White:   lipgloss.Color("7"),
		Muted:   lipgloss.Color("8"),
		Black:   lipgloss.Color("16"),
	}
)

func C(c color.Color, s string) string {
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

func B(s string) string {
	return lipgloss.NewStyle().Bold(true).Render(s)
}

func BC(c color.Color, s string) string {
	return lipgloss.NewStyle().Foreground(c).Bold(true).Render(s)
}

func U(s string) string {
	return lipgloss.NewStyle().Underline(true).Render(s)
}

func UC(c color.Color, s string) string {
	return lipgloss.NewStyle().Foreground(c).Underline(true).Render(s)
}

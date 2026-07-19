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
		Dim     color.Color
	}{
		Primary: lipgloss.Color("#65adff"),
		Green:   lipgloss.Color("#6fcc85"),
		Red:     lipgloss.Color("#ff7079"),
		Cyan:    lipgloss.Color("#11d4b7"),
		Yellow:  lipgloss.Color("#f5b41d"),
		Blue:    lipgloss.Color("#65adff"),
		Pink:    lipgloss.Color("#f67396"),
		Purple:  lipgloss.Color("#ca8aef"),
		White:   lipgloss.Color("#ffffff"),
		Muted:   lipgloss.Color("#868a91"),
		Black:   lipgloss.Color("#111317"),
		Dim:     lipgloss.Color("#363a41"),
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

type ThemeOption struct {
	Name  string
	Color color.Color
}

var ThemeOptions = []ThemeOption{
	{"blue", Colors.Blue},
	{"red", Colors.Red},
	{"amber", Colors.Yellow},
	{"green", Colors.Green},
	{"teal", Colors.Cyan},
	{"purple", Colors.Purple},
	{"pink", Colors.Pink},
}

func Theme(name string) color.Color {
	for _, opt := range ThemeOptions {
		if opt.Name == name {
			return opt.Color
		}
	}
	return Colors.Primary
}

func IsValidTheme(s string) bool {
	if s == "" {
		return true
	}
	for _, opt := range ThemeOptions {
		if opt.Name == s {
			return true
		}
	}
	return false
}

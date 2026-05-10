package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Colors mirror the web theme's HSL palette (web/static/css/index.css).
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
		Primary: lipgloss.Color("#65adff"), // blue
		Green:   lipgloss.Color("#6fcc85"),
		Red:     lipgloss.Color("#ff7079"),
		Cyan:    lipgloss.Color("#11d4b7"), // teal
		Yellow:  lipgloss.Color("#f5b41d"), // amber
		Blue:    lipgloss.Color("#65adff"),
		Pink:    lipgloss.Color("#f67396"),
		Purple:  lipgloss.Color("#ca8aef"),
		White:   lipgloss.Color("#ffffff"),
		Muted:   lipgloss.Color("#868a91"), // gray
		Black:   lipgloss.Color("#111317"), // surface-0
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

// ThemeOption is a named selectable accent color. The set mirrors the web
// theme palette in web/static/css/index.css.
type ThemeOption struct {
	Name  string
	Color color.Color
}

// ThemeOptions is the ordered list of named accent colors users can choose from.
var ThemeOptions = []ThemeOption{
	{"blue", Colors.Blue},
	{"red", Colors.Red},
	{"amber", Colors.Yellow},
	{"green", Colors.Green},
	{"teal", Colors.Cyan},
	{"purple", Colors.Purple},
	{"pink", Colors.Pink},
}

// Theme resolves a stored theme name to its color.Color. Unknown / empty names
// fall back to the default Primary color.
func Theme(name string) color.Color {
	for _, opt := range ThemeOptions {
		if opt.Name == name {
			return opt.Color
		}
	}
	return Colors.Primary
}

// IsValidTheme reports whether s is empty (use default) or one of the named
// ThemeOptions.
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

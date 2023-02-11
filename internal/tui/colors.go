package tui

import "github.com/charmbracelet/lipgloss"

var (
	Colors = struct {
		Green  lipgloss.TerminalColor
		Red    lipgloss.TerminalColor
		Cyan   lipgloss.TerminalColor
		Yellow lipgloss.TerminalColor
		Blue   lipgloss.TerminalColor
		Pink   lipgloss.TerminalColor
		Purple lipgloss.TerminalColor
		White  lipgloss.TerminalColor
		Muted  lipgloss.TerminalColor
	}{
		Green:  lipgloss.Color("36"),
		Red:    lipgloss.Color("9"),
		Cyan:   lipgloss.Color("6"),
		Yellow: lipgloss.Color("11"),
		Blue:   lipgloss.Color("4"),
		Pink:   lipgloss.Color("169"),
		Purple: lipgloss.Color("56"),
		White:  lipgloss.AdaptiveColor{Light: "#111111", Dark: "#ffffff"},
		Muted:  lipgloss.Color("247"),
	}

	ColorPrimary   = Colors.Green
	ColorSecondary = Colors.Blue
)

func C(c lipgloss.TerminalColor, s string) string {
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

func B(s string) string {
	return lipgloss.NewStyle().Bold(true).Render(s)
}

func BC(c lipgloss.TerminalColor, s string) string {
	return lipgloss.NewStyle().Foreground(c).Bold(true).Render(s)
}

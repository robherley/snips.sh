package styles

import "github.com/charmbracelet/lipgloss"

var (
	Colors = struct {
		Primary lipgloss.TerminalColor
		Green   lipgloss.TerminalColor
		Red     lipgloss.TerminalColor
		Cyan    lipgloss.TerminalColor
		Yellow  lipgloss.TerminalColor
		Blue    lipgloss.TerminalColor
		Pink    lipgloss.TerminalColor
		Purple  lipgloss.TerminalColor
		White   lipgloss.TerminalColor
		Muted   lipgloss.TerminalColor
		Black   lipgloss.TerminalColor
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

func C(c lipgloss.TerminalColor, s string) string {
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

func B(s string) string {
	return lipgloss.NewStyle().Bold(true).Render(s)
}

func BC(c lipgloss.TerminalColor, s string) string {
	return lipgloss.NewStyle().Foreground(c).Bold(true).Render(s)
}

package tui

import "github.com/charmbracelet/lipgloss"

var Colors = struct {
	Green  lipgloss.Color
	Red    lipgloss.Color
	Cyan   lipgloss.Color
	Yellow lipgloss.Color
	Blue   lipgloss.Color
	Pink   lipgloss.Color
	Purple lipgloss.Color
	White  lipgloss.Color
	Muted  lipgloss.Color
}{
	Green:  lipgloss.Color("10"),
	Red:    lipgloss.Color("9"),
	Cyan:   lipgloss.Color("14"),
	Yellow: lipgloss.Color("11"),
	Blue:   lipgloss.Color("12"),
	Pink:   lipgloss.Color("169"),
	Purple: lipgloss.Color("13"),
	White:  lipgloss.Color("15"),
	Muted:  lipgloss.Color("247"),
}

func C(c lipgloss.Color, s string) string {
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

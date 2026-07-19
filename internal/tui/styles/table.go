package styles

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

type TableSection struct {
	Label color.Color
	Rows  [][2]string
}

func Table(sections ...TableSection) string {
	labelWidth, valueWidth := 0, 0
	for _, section := range sections {
		for _, row := range section.Rows {
			labelWidth = max(labelWidth, lipgloss.Width(row[0]))
			valueWidth = max(valueWidth, lipgloss.Width(row[1]))
		}
	}

	border := lipgloss.NewStyle().Foreground(Colors.Muted)
	valueCell := lipgloss.NewStyle().Width(valueWidth)

	renderBorder := func(left, mid, right string) string {
		return border.Render(left + strings.Repeat("─", labelWidth+2) + mid + strings.Repeat("─", valueWidth+2) + right)
	}

	lines := []string{renderBorder("╭", "┬", "╮")}
	for i, section := range sections {
		if i > 0 {
			lines = append(lines, renderBorder("├", "┼", "┤"))
		}
		labelStyle := lipgloss.NewStyle().Foreground(section.Label).Bold(true).Width(labelWidth)
		for _, row := range section.Rows {
			lines = append(lines,
				border.Render("│ ")+labelStyle.Render(row[0])+
					border.Render(" │ ")+valueCell.Render(row[1])+border.Render(" │"))
		}
	}
	lines = append(lines, renderBorder("╰", "┴", "╯"))

	return strings.Join(lines, "\n")
}

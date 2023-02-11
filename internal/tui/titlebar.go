package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type titleBar struct {
	Width int
}

func NewTitleBar(width int) *titleBar {
	return &titleBar{
		Width: width,
	}
}

func (tb *titleBar) Init() tea.Cmd {
	return nil
}

func (tb *titleBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tb.Width = msg.Width
	}
	return tb, nil
}

func (tb *titleBar) View() string {
	title := lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(1).
		Padding(0, 1, 0, 1).
		Background(ColorPrimary).
		Foreground(Colors.White).
		Bold(true).
		Render("snips.sh")

	return title + strings.Repeat(C(ColorPrimary, "┃"), tb.Width-lipgloss.Width((title)))
}

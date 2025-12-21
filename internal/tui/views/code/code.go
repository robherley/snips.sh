package code

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type Code struct {
	viewport *viewport.Model
	file     *snips.File
	content  string
}

func New(width, height int) *Code {
	vp := viewport.New(width, height)
	return &Code{
		viewport: &vp,
	}
}

func (m *Code) Init() tea.Cmd {
	m.viewport.GotoTop()
	m.viewport.SetContent(m.content)
	return nil
}

func (m *Code) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
	case msgs.FileLoaded:
		m.file = msg.File
		m.content = m.renderContent(msg.File)
		m.Init()
	case msgs.FileDeselected:
		m.file = nil
	}

	vp, cmd := m.viewport.Update(msg)
	m.viewport = &vp
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Code) View() string {
	return m.viewport.View()
}

func (m Code) Keys() help.KeyMap {
	return keys
}

func (m *Code) renderContent(file *snips.File) string {
	if file == nil {
		return ""
	}

	fileContent, err := file.GetContent()
	if err != nil {
		slog.Error("unable to get file content", "err", err)
		return "error getting file content"
	}

	content, err := renderer.ToSyntaxHighlightedTerm(file.Type, fileContent)
	if err != nil {
		slog.Warn("failed to render file as syntax highlighted", "err", err)
		content = string(fileContent)
	}

	lines := strings.Split(content, "\n")
	maxDigits := len(fmt.Sprintf("%d", len(lines)))

	// ditch the last newline
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	borderStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Muted).
		Border(lipgloss.NormalBorder(), false).
		BorderForeground(styles.Colors.Muted)

	renderedLines := make([]string, 0, len(lines))

	lineStyle := borderStyle.BorderRight(true).MarginRight(1)
	for i, line := range lines {
		lineNumber := lineStyle.Render(fmt.Sprintf("%*d", maxDigits, i+1))

		scrubbed := strings.ReplaceAll(line, "\t", "    ")
		renderedLines = append(renderedLines, lineNumber+scrubbed)
	}

	return strings.Join(renderedLines, "\n")
}

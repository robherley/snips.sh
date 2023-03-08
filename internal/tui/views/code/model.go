package code

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/rs/zerolog/log"
)

type Model struct {
	viewport *viewport.Model
	file     *db.File
	content  string
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	return &Model{
		viewport: &vp,
	}
}

func (m *Model) Init() tea.Cmd {
	m.viewport.GotoTop()
	m.viewport.SetContent(m.content)
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *Model) View() string {
	return m.viewport.View()
}

func (m *Model) renderContent(file *db.File) string {
	if file == nil {
		return ""
	}

	content, err := renderer.ToSyntaxHighlightedTerm(file.Type, []byte(file.Content))
	if err != nil {
		log.Warn().Err(err).Msg("failed to render file as syntax highlighted")
		content = string(file.Content)
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

	builder := strings.Builder{}

	builder.WriteString(strings.Repeat(" ", int(maxDigits)))
	builder.WriteString(styles.C(styles.Colors.Muted, lipgloss.NormalBorder().TopLeft))
	builder.WriteString(styles.C(styles.Colors.Muted, strings.Repeat(lipgloss.NormalBorder().Bottom, m.viewport.Width-int(maxDigits))))
	builder.WriteRune('\n')

	for i, line := range lines {
		lineNumber := borderStyle.
			Copy().
			BorderRight(true).
			MarginRight(1).
			Render(fmt.Sprintf("%*d", int(maxDigits), i+1))
		builder.WriteString(lineNumber)
		builder.WriteString(strings.ReplaceAll(line, "\t", "    "))
		builder.WriteRune('\n')
	}

	builder.WriteString(strings.Repeat(" ", int(maxDigits)))
	builder.WriteString(styles.C(styles.Colors.Muted, lipgloss.NormalBorder().BottomLeft))
	builder.WriteString(styles.C(styles.Colors.Muted, strings.Repeat(lipgloss.NormalBorder().Top, m.viewport.Width-int(maxDigits))))
	builder.WriteRune('\n')

	return builder.String()
}

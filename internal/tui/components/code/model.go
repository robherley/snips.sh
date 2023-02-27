package code

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/tui/messages"
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
	case messages.FileLoaded:
		m.file = msg.File
		m.content = renderContent(msg.File)
		m.Init()
	}

	vp, cmd := m.viewport.Update(msg)
	m.viewport = &vp
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	return m.viewport.View()
}

func renderContent(file *db.File) string {
	if file == nil {
		return ""
	}

	content, err := renderer.ToSyntaxHighlightedTerm(file.Type, []byte(file.Content))
	if err != nil {
		log.Warn().Err(err).Msg("failed to render file as syntax highlighted")
		content = string(file.Content)
	}

	length := len(content)
	maxDigits := math.Floor(math.Log10(float64(length)))

	builder := strings.Builder{}
	for i, line := range strings.Split(content, "\n") {
		builder.WriteString(fmt.Sprintf("%*d ", int(maxDigits-1), i+1))
		builder.WriteString(strings.ReplaceAll(line, "\t", "    "))
		builder.WriteRune('\n')
	}

	return builder.String()
}

package code

import (
	"fmt"
	"image/color"
	"log/slog"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type Code struct {
	viewport viewport.Model
	file     *snips.File
	content  string
	theme    color.Color
}

func New(width, height int, theme color.Color) Code {
	w, h := frameFit(width, height)
	return Code{
		viewport: viewport.New(viewport.WithWidth(w), viewport.WithHeight(h)),
		theme:    theme,
	}
}

func frameFit(width, height int) (int, int) {
	return max(width-2, 0), max(height-4, 0)
}

func (m Code) Init() tea.Cmd {
	return nil
}

func (m Code) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Top):
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, keys.Bottom):
			m.viewport.GotoBottom()
			return m, nil
		}
	case tea.WindowSizeMsg:
		w, h := frameFit(msg.Width, msg.Height)
		m.viewport.SetWidth(w)
		m.viewport.SetHeight(h)
	case msgs.FileLoaded:
		m.file = msg.File
		m.content = m.renderContent(msg.File)
		m.viewport.GotoTop()
		m.viewport.SetContent(m.content)
	case msgs.FileDeselected:
		m.file = nil
	case msgs.ThemeChanged:
		m.theme = msg.Color
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Code) View() tea.View {
	return tea.NewView(styles.Frame(m.theme, m.titleRow(), m.viewport.View()))
}

func (m Code) Keys() help.KeyMap {
	return keys
}

func (m Code) IsCapturing() bool {
	return false
}

func (m Code) titleRow() string {
	if m.file == nil {
		return ""
	}

	meta := strings.Join([]string{
		strings.ToLower(m.file.Type),
		humanize.Bytes(m.file.Size),
		humanize.Time(m.file.UpdatedAt),
	}, " · ")

	return styles.BC(m.theme, m.file.DisplayName()) + styles.C(styles.Colors.Muted, " · "+meta)
}

func (m Code) renderContent(file *snips.File) string {
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

	numberStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)
	separator := lipgloss.NewStyle().Foreground(styles.Colors.Muted).Render("▏")

	renderedLines := make([]string, 0, len(lines))

	for i, line := range lines {
		lineNumber := numberStyle.Render(fmt.Sprintf(" %*d ", maxDigits, i+1)) + separator

		scrubbed := strings.ReplaceAll(line, "\t", "    ")
		renderedLines = append(renderedLines, lineNumber+scrubbed)
	}

	return strings.Join(renderedLines, "\n")
}

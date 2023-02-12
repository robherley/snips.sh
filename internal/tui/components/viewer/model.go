package viewer

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/rs/zerolog/log"
)

type Model struct {
	viewport viewport.Model
}

func New(width, height int) *Model {
	return &Model{
		viewport: viewport.New(width, height),
	}
}

func (v *Model) Init() tea.Cmd {
	return nil
}

func (v *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

func (v *Model) View() string {
	return v.viewport.View()
}

func (v *Model) SetFile(file *db.File) {
	content, err := renderer.ToSyntaxHighlightedTerm(file.Type, file.Content)
	if err != nil {
		log.Warn().Err(err).Msg("unable to render file")
		// fallback to the plain text version if syntax highlighting fails
		v.viewport.SetContent(content)
	}
	v.viewport.SetContent(content)
}

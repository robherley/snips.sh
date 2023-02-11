package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/tui/components/filelist"
	"github.com/robherley/snips.sh/internal/tui/components/titlebar"
	"github.com/robherley/snips.sh/internal/tui/messages"
	"github.com/rs/zerolog/log"
)

type TUI struct {
	Window      ssh.Window
	UserID      string
	Fingerprint string
	DB          *db.DB

	models map[string]tea.Model
}

func New(window ssh.Window, userID string, fingerPrint string, database *db.DB, files []filelist.ListItem) *TUI {
	models := map[string]tea.Model{
		titlebar.Name: titlebar.New(window.Width),
		filelist.Name: filelist.New(window.Width, window.Height-2, files),
	}

	fv := &TUI{
		Window:      window,
		UserID:      userID,
		Fingerprint: fingerPrint,
		DB:          database,
		models:      models,
	}

	return fv
}

func (fv *TUI) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)

	for k := range fv.models {
		cmd := fv.models[k].Init()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

func (fv *TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case messages.Error:
		log.Error().Err(msg).Msg("encountered error")
		return fv, tea.Quit
	case tea.WindowSizeMsg:
		fv.Window.Height = msg.Height
		fv.Window.Width = msg.Width

		fv.models[titlebar.Name].Update(tea.WindowSizeMsg{Width: msg.Width})
		fv.models[filelist.Name].Update(tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - 2})
		return fv, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return fv, tea.Quit
		}
	}

	for k, v := range fv.models {
		model, cmd := v.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		fv.models[k] = model
	}

	return fv, tea.Batch(cmds...)
}

func (fv *TUI) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		fv.models[titlebar.Name].View(),
		fv.models[filelist.Name].View(),
	)
}

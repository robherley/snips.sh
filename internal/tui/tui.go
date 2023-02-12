package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/tui/components/filelist"
	"github.com/robherley/snips.sh/internal/tui/components/titlebar"
	"github.com/robherley/snips.sh/internal/tui/components/viewer"
	"github.com/robherley/snips.sh/internal/tui/messages"
	"github.com/rs/zerolog/log"
)

type TUI struct {
	Window      ssh.Window
	UserID      string
	Fingerprint string
	DB          *db.DB

	models       map[string]tea.Model
	selectedFile *db.File
}

func New(window ssh.Window, userID string, fingerPrint string, database *db.DB, files []filelist.ListItem) *TUI {
	models := map[string]tea.Model{
		titlebar.Name: titlebar.New(window.Width),
		filelist.Name: filelist.New(window.Width, window.Height-2, files),
		viewer.Name:   viewer.New(window.Width, window.Height),
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

func (t *TUI) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)

	for k := range t.models {
		cmd := t.models[k].Init()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

func (t *TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case messages.Error:
		log.Error().Err(msg).Msg("encountered error")
		return t, tea.Quit
	case tea.WindowSizeMsg:
		t.Window.Height = msg.Height
		t.Window.Width = msg.Width

		t.models[titlebar.Name].Update(tea.WindowSizeMsg{Width: msg.Width})
		t.models[filelist.Name].Update(tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - 2})
		t.models[viewer.Name].Update(tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height})
		return t, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return t, tea.Quit
		}
	case messages.SelectedFile:
		file := db.File{}
		if err := t.DB.Find(&file, "id = ? AND user_id = ?", msg.ID, t.UserID).Error; err != nil {
			return t, func() tea.Msg { return messages.Error{Err: err} }
		}

		t.models[viewer.Name].(*viewer.Model).SetFile(&file)
		t.selectedFile = &file

		return t, nil
	}

	for k, v := range t.models {
		model, cmd := v.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		t.models[k] = model
	}

	return t, tea.Batch(cmds...)
}

func (t *TUI) View() string {
	if t.selectedFile != nil {
		return t.models[viewer.Name].View()
	}

	return lipgloss.JoinVertical(
		lipgloss.Top,
		t.models[titlebar.Name].View(),
		t.models[filelist.Name].View(),
	)
}

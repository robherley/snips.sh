package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/rs/zerolog/log"
)

type fileView struct {
	Window      ssh.Window
	UserID      string
	Fingerprint string
	DB          *db.DB

	models map[string]tea.Model
	files  []db.File
}

func NewFileView(window ssh.Window, userID string, fingerPrint string, database *db.DB) *fileView {
	models := map[string]tea.Model{
		"titleBar": NewTitleBar(window.Width),
		"fileList": NewFileList(window.Width, window.Height-2),
	}

	fv := &fileView{
		Window:      window,
		UserID:      userID,
		Fingerprint: fingerPrint,
		DB:          database,
		models:      models,
	}

	return fv
}

func (fv *fileView) Init() tea.Cmd {
	for k := range fv.models {
		fv.models[k].Init()
	}

	return fv.getFilesCmd
}

func (fv *fileView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case FilesMsg:
		fv.files = msg.Files
	case ErrorMsg:
		log.Error().Err(msg).Msg("encountered error")
		return fv, tea.Quit
	case tea.WindowSizeMsg:
		fv.Window.Height = msg.Height
		fv.Window.Width = msg.Width

		fv.models["titleBar"].Update(tea.WindowSizeMsg{Width: msg.Width})
		fv.models["fileList"].Update(tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - 2})
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

func (fv *fileView) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		fv.models["titleBar"].View(),
		fv.models["fileList"].View(),
	)
}

func (fv *fileView) getFilesCmd() tea.Msg {
	files := []db.File{}
	if err := fv.DB.Where("user_id = ?", fv.UserID).Order("created_at DESC").Find(&files).Error; err != nil {
		return ErrorMsg{err}
	}

	return FilesMsg{files}
}

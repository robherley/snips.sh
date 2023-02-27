package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/tui/commands"
	"github.com/robherley/snips.sh/internal/tui/components/code"
	"github.com/robherley/snips.sh/internal/tui/components/filelist"
	"github.com/robherley/snips.sh/internal/tui/messages"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/rs/zerolog/log"
)

type View int

const (
	ViewFileList View = iota
	ViewCode
)

type TUI struct {
	UserID      string
	Fingerprint string
	DB          *db.DB

	width  int
	height int

	selectedFile *db.File

	currentView View
	views       map[View]tea.Model
}

func New(width, height int, userID string, fingerPrint string, database *db.DB, files []filelist.ListItem) TUI {
	views := map[View]tea.Model{
		ViewFileList: filelist.New(width, height-1, files),
		ViewCode:     code.New(width, height-1),
	}

	return TUI{
		UserID:      userID,
		Fingerprint: fingerPrint,
		DB:          database,

		width:        width,
		height:       height,
		selectedFile: nil,
		currentView:  ViewFileList,
		views:        views,
	}
}

func (t TUI) Init() tea.Cmd {
	return nil
}

func (t TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case messages.Error:
		log.Error().Err(msg).Msg("encountered error")
		return t, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return t, tea.Quit
		case "esc":
			t.currentView = ViewFileList
			t.selectedFile = nil
			return t, nil
		}
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height

		for key, view := range t.views {
			t.views[key], cmd = view.Update(tea.WindowSizeMsg{
				Width:  t.width,
				Height: t.height - 1,
			})
			cmds = append(cmds, cmd)
		}

		return t, tea.Batch(cmds...)
	case messages.FileSelected:
		return t, commands.GetFile(t.DB, msg.ID, t.UserID)
	case messages.FileLoaded:
		t.selectedFile = msg.File
		t.currentView = ViewCode
	}

	for key, view := range t.views {
		t.views[key], cmd = view.Update(msg)
		cmds = append(cmds, cmd)
	}

	return t, tea.Batch(cmds...)
}

func (t TUI) View() string {
	return lipgloss.JoinVertical(lipgloss.Top, t.titleBar(), t.views[t.currentView].View())
}

func (t TUI) titleBar() string {
	titleText := "snips.sh"
	if t.selectedFile != nil {
		titleText = fmt.Sprintf("%s > %s", titleText, t.selectedFile.ID)
	}

	title := lipgloss.NewStyle().
		Padding(0, 1, 0, 1).
		Background(styles.ColorPrimary).
		Foreground(styles.Colors.White).
		Bold(true).
		Render(titleText)

	return title + strings.Repeat(styles.C(styles.ColorPrimary, "┃"), t.width-lipgloss.Width((title)))
}

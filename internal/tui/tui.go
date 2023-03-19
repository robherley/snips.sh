package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/db/models"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/robherley/snips.sh/internal/tui/views"
	"github.com/robherley/snips.sh/internal/tui/views/code"
	"github.com/robherley/snips.sh/internal/tui/views/filelist"
	"github.com/robherley/snips.sh/internal/tui/views/fileoptions"
	"github.com/robherley/snips.sh/internal/tui/views/prompt"
	"github.com/rs/zerolog/log"
)

type TUI struct {
	UserID      string
	Fingerprint string
	DB          *db.DB

	cfg    *config.Config
	width  int
	height int

	file      *models.File
	viewStack []views.View
	views     map[views.View]tea.Model
}

func New(cfg *config.Config, width, height int, userID string, fingerPrint string, database *db.DB, files []filelist.ListItem) TUI {
	return TUI{
		UserID:      userID,
		Fingerprint: fingerPrint,
		DB:          database,

		cfg:       cfg,
		width:     width,
		height:    height,
		file:      nil,
		viewStack: []views.View{views.FileList},
		views: map[views.View]tea.Model{
			views.FileList:    filelist.New(width, height-1, files),
			views.Code:        code.New(width, height-1),
			views.FileOptions: fileoptions.New(cfg),
			views.Prompt:      prompt.New(database),
		},
	}
}

func (t TUI) Init() tea.Cmd {
	return nil
}

func (t TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		batchedCmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case msgs.Error:
		log.Error().Err(msg).Msg("encountered error")
		return t, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if !t.inPrompt() {
				return t, tea.Quit
			}
		case "ctrl+c":
			return t, tea.Quit
		case "esc":
			if len(t.viewStack) == 1 {
				return t, tea.Quit
			} else {
				batchedCmds = append(batchedCmds, cmds.PopView())
				if t.currentView() == views.FileList {
					batchedCmds = append(batchedCmds, cmds.DeselectFile())
				}
				return t, tea.Batch(batchedCmds...)
			}
		}

		// only send key msgs to the current view
		var cmd tea.Cmd
		t.views[t.currentView()], cmd = t.views[t.currentView()].Update(msg)
		return t, cmd
	case msgs.PushView:
		t.pushView(msg.View)
	case msgs.PopView:
		t.popView()
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height

		for key, view := range t.views {
			var cmd tea.Cmd
			t.views[key], cmd = view.Update(tea.WindowSizeMsg{
				Width:  t.width,
				Height: t.height - 1,
			})
			batchedCmds = append(batchedCmds, cmd)
		}

		return t, tea.Batch(batchedCmds...)
	case msgs.FileSelected:
		return t, cmds.GetFile(t.DB, msg.ID, t.UserID)
	case msgs.FileDeselected:
		t.file = nil
	case msgs.FileLoaded:
		t.file = msg.File
	}

	for key, view := range t.views {
		var cmd tea.Cmd
		t.views[key], cmd = view.Update(msg)
		batchedCmds = append(batchedCmds, cmd)
	}

	return t, tea.Batch(batchedCmds...)
}

func (t TUI) View() string {
	return lipgloss.JoinVertical(lipgloss.Top, t.titleBar(), t.views[t.currentView()].View())
}

func (t TUI) titleBar() string {
	titleText := "┃ snips.sh"
	if t.file != nil {
		titleText += fmt.Sprintf(" / %s", t.file.ID)
	}
	titleText += " "

	title := lipgloss.NewStyle().
		Foreground(styles.Colors.Green).
		Bold(false).
		Render(titleText)

	return title + strings.Repeat(styles.C(styles.Colors.Green, "┃"), t.width-lipgloss.Width((title)))
}

func (t TUI) currentView() views.View {
	return t.viewStack[len(t.viewStack)-1]
}

func (t *TUI) pushView(view views.View) {
	t.viewStack = append(t.viewStack, view)
}

func (t *TUI) popView() {
	t.viewStack = t.viewStack[:len(t.viewStack)-1]
}

func (t *TUI) inPrompt() bool {
	return t.currentView() == views.Prompt
}

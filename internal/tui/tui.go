package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/robherley/snips.sh/internal/tui/views"
	"github.com/robherley/snips.sh/internal/tui/views/browser"
	"github.com/robherley/snips.sh/internal/tui/views/code"
	"github.com/robherley/snips.sh/internal/tui/views/prompt"
	"github.com/rs/zerolog/log"
)

type TUI struct {
	UserID      string
	Fingerprint string
	DB          db.DB

	cfg    *config.Config
	width  int
	height int

	file      *snips.File
	viewStack []views.View
	views     map[views.View]tea.Model
}

func New(cfg *config.Config, width, height int, userID string, fingerprint string, database db.DB, files []*snips.File) TUI {
	return TUI{
		UserID:      userID,
		Fingerprint: fingerprint,
		DB:          database,

		cfg:       cfg,
		width:     width,
		height:    height,
		file:      nil,
		viewStack: []views.View{views.Browser},
		views: map[views.View]tea.Model{
			views.Browser: browser.New(cfg, width, height-1, files),
			views.Code:    code.New(width, height-1),
			views.Prompt:  prompt.New(database),
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
				if t.currentView() == views.Browser {
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
		return t, cmds.LoadFile(t.DB, msg.ID)
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
	textStyle := lipgloss.NewStyle().Foreground(styles.Colors.Black).Background(styles.Colors.Primary).Padding(0, 1).Bold(true)
	title := textStyle.Render("snips.sh")
	user := textStyle.Render(fmt.Sprintf("u:%s", t.UserID))
	return title + strings.Repeat(styles.BC(styles.Colors.Primary, "╱"), t.width-lipgloss.Width(title)-lipgloss.Width(user)) + user
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

package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
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

	ctx    context.Context
	cfg    *config.Config
	width  int
	height int

	file      *snips.File
	viewStack []views.Kind
	views     map[views.Kind]views.Model
	help      help.Model
}

func New(ctx context.Context, cfg *config.Config, width, height int, userID string, fingerprint string, database db.DB, files []*snips.File) TUI {
	t := TUI{
		UserID:      userID,
		Fingerprint: fingerprint,
		DB:          database,

		ctx:       ctx,
		cfg:       cfg,
		width:     width,
		height:    height,
		file:      nil,
		viewStack: []views.Kind{views.Browser},
		help:      help.New(),
	}

	t.views = map[views.Kind]views.Model{
		views.Browser: browser.New(cfg, width, t.innerViewHeight(), files),
		views.Code:    code.New(width, t.innerViewHeight()),
		views.Prompt:  prompt.New(ctx, cfg, database, width),
	}

	t.help.Styles = styles.Help

	return t
}

func (t TUI) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, view := range t.views {
		vcmd := view.Init()
		if vcmd != nil {
			cmds = append(cmds, vcmd)
		}
	}

	return tea.Batch(cmds...)
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
		case "?":
			t.help.ShowAll = !t.help.ShowAll
			t.updateViewSize()
		case "q":
			if !t.inPrompt() {
				return t, tea.Quit
			}
		case "ctrl+c":
			return t, tea.Quit
		case "esc":
			if t.currentViewKind() == views.Browser && t.views[views.Browser].(browser.Browser).IsOptionsFocused() {
				// special case where options focused, also allow escape to unfocus too
				// allows browser to capture the escape key
				break
			}

			if len(t.viewStack) == 1 {
				return t, tea.Quit
			}

			batchedCmds = append(batchedCmds, cmds.PopView())
			if t.currentViewKind() == views.Browser {
				batchedCmds = append(batchedCmds, cmds.DeselectFile())
			}
			return t, tea.Batch(batchedCmds...)
		}

		// otherwise, send key msgs to the current view
		view, cmd := t.views[t.currentViewKind()].Update(msg)
		t.views[t.currentViewKind()] = view.(views.Model)
		return t, cmd
	case msgs.PushView:
		t.pushView(msg.View)
	case msgs.PopView:
		t.popView()
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height

		batchedCmds = append(batchedCmds, t.updateViewSize()...)
		t.help.Width = msg.Width

		return t, tea.Batch(batchedCmds...)
	case msgs.FileSelected:
		return t, cmds.LoadFile(t.DB, msg.ID)
	case msgs.FileDeselected, msgs.ReloadFiles:
		t.file = nil
	case msgs.FileLoaded:
		t.file = msg.File
	}

	for key, view := range t.views {
		newView, cmd := view.Update(msg)
		t.views[key] = newView.(views.Model)
		batchedCmds = append(batchedCmds, cmd)
	}

	return t, tea.Batch(batchedCmds...)
}

func (t TUI) View() string {
	return lipgloss.JoinVertical(lipgloss.Top, t.titleBar(), t.currentViewModel().View(), t.helpBar())
}

func (t TUI) titleBar() string {
	textStyle := lipgloss.NewStyle().Foreground(styles.Colors.Black).Background(styles.Colors.Primary).Padding(0, 1).Bold(true)
	title := textStyle.Render("snips.sh")
	user := textStyle.Render(fmt.Sprintf("u:%s", t.UserID))
	return title + strings.Repeat(styles.BC(styles.Colors.Primary, "╱"), t.width-lipgloss.Width(title)-lipgloss.Width(user)) + user
}

func (t TUI) helpBar() string {
	if len(t.views) == 0 {
		return ""
	}

	return t.help.View(t.currentViewModel().Keys())
}

func (t TUI) currentViewKind() views.Kind {
	return t.viewStack[len(t.viewStack)-1]
}

func (t TUI) currentViewModel() views.Model {
	return t.views[t.currentViewKind()]
}

func (t *TUI) pushView(view views.Kind) {
	t.viewStack = append(t.viewStack, view)
}

func (t *TUI) popView() {
	t.viewStack = t.viewStack[:len(t.viewStack)-1]
}

func (t TUI) inPrompt() bool {
	return t.currentViewKind() == views.Prompt
}

func (t TUI) innerViewHeight() int {
	height := t.height - (lipgloss.Height(t.titleBar()) + lipgloss.Height(t.helpBar()))
	if height < 0 {
		return 0
	}

	return height
}

func (t *TUI) updateViewSize() []tea.Cmd {
	batchedCmds := make([]tea.Cmd, 0, len(t.views))

	for key, view := range t.views {
		newView, cmd := view.Update(tea.WindowSizeMsg{
			Width:  t.width,
			Height: t.innerViewHeight(),
		})
		t.views[key] = newView.(views.Model)
		batchedCmds = append(batchedCmds, cmd)
	}

	return batchedCmds
}

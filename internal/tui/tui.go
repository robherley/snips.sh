package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
)

type TUI struct {
	UserID      string
	Fingerprint string
	DB          db.DB

	ctx    context.Context
	cfg    *config.Config
	width  int
	height int

	file   *snips.File
	views  []views.Kind  // navigation stack
	models []views.Model // indexed by views.Kind; deterministic iteration order
	help   help.Model
}

func New(ctx context.Context, cfg *config.Config, width, height int, userID string, fingerprint string, database db.DB, files []*snips.File) TUI {
	t := TUI{
		UserID:      userID,
		Fingerprint: fingerprint,
		DB:          database,

		ctx:    ctx,
		cfg:    cfg,
		width:  width,
		height: height,
		file:   nil,
		views:  []views.Kind{views.Browser},
		help:   help.New(),
	}

	t.models = []views.Model{
		views.Browser: browser.New(cfg, width, t.innerViewHeight(), files),
		views.Code:    code.New(width, t.innerViewHeight()),
		views.Prompt:  prompt.New(ctx, cfg, database, width),
	}

	t.help.Styles = styles.Help

	return t
}

func (t TUI) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(t.models))
	for _, model := range t.models {
		if mcmd := model.Init(); mcmd != nil {
			cmds = append(cmds, mcmd)
		}
	}

	return tea.Batch(cmds...)
}

func (t TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.Error:
		slog.Error("encountered error", "err", msg)
		return t, tea.Quit
	case tea.KeyPressMsg:
		switch msg.String() {
		case "?":
			t.help.ShowAll = !t.help.ShowAll
			return t, tea.Batch(t.updateViewSize()...)
		case "q":
			if !t.inPrompt() {
				return t, tea.Quit
			}
		case "ctrl+c":
			return t, tea.Quit
		case "esc":
			if t.currentViewKind() == views.Browser && t.models[views.Browser].(browser.Browser).IsOptionsFocused() {
				// special case where options focused, also allow escape to unfocus too
				// allows browser to capture the escape key
				break
			}

			if len(t.views) == 1 {
				return t, tea.Quit
			}

			batched := []tea.Cmd{cmds.PopView()}
			if t.currentViewKind() == views.Browser {
				batched = append(batched, cmds.DeselectFile())
			}
			return t, tea.Batch(batched...)
		}

		// otherwise, send key msgs to the current view
		return t, t.updateCurrent(msg)
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		t.help.SetWidth(msg.Width)
		return t, tea.Batch(t.updateViewSize()...)
	case msgs.FileSelected:
		return t, cmds.LoadFile(t.DB, msg.ID)
	case msgs.FileLoaded:
		t.file = msg.File
		return t, t.broadcast(msg)
	case msgs.FileDeselected, msgs.ReloadFiles:
		t.file = nil
		return t, t.broadcast(msg)
	case msgs.PushView:
		t.pushView(msg.View)
		return t, t.broadcast(msg)
	case msgs.PopView:
		t.popView()
		return t, t.broadcast(msg)
	}

	// fall-through messages (cursor blink, etc.) only need the active view
	return t, t.updateCurrent(msg)
}

func (t TUI) View() tea.View {
	v := t.currentViewModel().View()
	v.Content = lipgloss.JoinVertical(lipgloss.Top, t.titleBar(), v.Content, t.helpBar())
	v.AltScreen = true
	return v
}

func (t TUI) titleBar() string {
	textStyle := lipgloss.NewStyle().Foreground(styles.Colors.Black).Background(styles.Colors.Primary).Padding(0, 1).Bold(true)
	title := textStyle.Render("snips.sh")
	user := textStyle.Render(fmt.Sprintf("u:%s", t.UserID))

	count := t.width - lipgloss.Width(title) - lipgloss.Width(user)
	if count < 0 {
		count = 0
	}

	return title + strings.Repeat(styles.BC(styles.Colors.Primary, "╱"), count) + user
}

func (t TUI) helpBar() string {
	if len(t.models) == 0 {
		return ""
	}

	return t.help.View(t.currentViewModel().Keys())
}

func (t TUI) currentViewKind() views.Kind {
	return t.views[len(t.views)-1]
}

func (t TUI) currentViewModel() views.Model {
	return t.models[t.currentViewKind()]
}

func (t *TUI) pushView(view views.Kind) {
	t.views = append(t.views, view)
}

func (t *TUI) popView() {
	t.views = t.views[:len(t.views)-1]
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

// updateCurrent forwards a message to the active view only.
func (t *TUI) updateCurrent(msg tea.Msg) tea.Cmd {
	kind := t.currentViewKind()
	model, cmd := t.models[kind].Update(msg)
	t.models[kind] = model.(views.Model)
	return cmd
}

// broadcast forwards a message to every view in deterministic order.
func (t *TUI) broadcast(msg tea.Msg) tea.Cmd {
	batched := make([]tea.Cmd, 0, len(t.models))
	for i, model := range t.models {
		newModel, cmd := model.Update(msg)
		t.models[i] = newModel.(views.Model)
		batched = append(batched, cmd)
	}
	return tea.Batch(batched...)
}

func (t *TUI) updateViewSize() []tea.Cmd {
	batched := make([]tea.Cmd, 0, len(t.models))
	for i, model := range t.models {
		newModel, cmd := model.Update(tea.WindowSizeMsg{
			Width:  t.width,
			Height: t.innerViewHeight(),
		})
		t.models[i] = newModel.(views.Model)
		batched = append(batched, cmd)
	}
	return batched
}

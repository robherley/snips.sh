package tui

import (
	"context"
	"fmt"
	"image/color"
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
	"github.com/robherley/snips.sh/internal/tui/views/settings"
)

// topLevelTabs lists the top-level views cycled with shift+tab, in tab-bar order.
var topLevelTabs = []struct {
	kind  views.Kind
	label string
}{
	{views.Browser, "Files"},
	{views.Settings, "Settings"},
}

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
	theme  color.Color
}

func New(ctx context.Context, cfg *config.Config, width, height int, user *snips.User, fingerprint string, database db.DB, files []*snips.File) TUI {
	t := TUI{
		UserID:      user.ID,
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

	theme := styles.Theme(user.ThemeColor)
	t.theme = theme

	t.models = []views.Model{
		views.Browser:  browser.New(cfg, width, t.innerViewHeight(), files, theme),
		views.Code:     code.New(width, t.innerViewHeight()),
		views.Prompt:   prompt.New(ctx, cfg, database, width),
		views.Settings: settings.New(width, t.innerViewHeight(), database, user, fingerprint),
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
		// ctrl+c always quits, even when a view is capturing input
		if msg.String() == "ctrl+c" {
			return t, tea.Quit
		}
		// shift+tab top-level navigation always wins (it isn't typeable)
		if msg.String() == "shift+tab" {
			t.cycleTopLevel()
			return t, nil
		}
		// when a view is consuming raw input (filter, text field), skip our shortcuts
		if t.currentViewModel().IsCapturing() {
			return t, t.updateCurrent(msg)
		}

		switch msg.String() {
		case "?":
			t.help.ShowAll = !t.help.ShowAll
			return t, tea.Batch(t.updateViewSize()...)
		case "q":
			if !t.inPrompt() {
				return t, tea.Quit
			}
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
	case msgs.ThemeChanged:
		t.theme = msg.Color
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
	brand := lipgloss.NewStyle().
		Foreground(styles.Colors.Black).
		Background(t.theme).
		Padding(0, 1).
		Bold(true).
		Render("snips.sh")
	tabs := t.tabs()

	count := t.width - lipgloss.Width(brand) - lipgloss.Width(tabs)
	if count < 0 {
		count = 0
	}

	return brand + tabs + strings.Repeat(styles.BC(t.theme, "╱"), count)
}

func (t TUI) tabs() string {
	active := t.views[0]

	activeStyle := lipgloss.NewStyle().Foreground(t.theme).Bold(true).Underline(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)
	sep := styles.C(t.theme, " ╱ ")

	out := sep // initial separator between brand and tabs
	for i, tab := range topLevelTabs {
		if i > 0 {
			out += sep
		}
		label := tab.label
		if tab.kind == active {
			out += activeStyle.Render(label)
		} else {
			out += inactiveStyle.Render(label)
		}
	}
	return out + " " // breathing room before the slashes
}

// cycleTopLevel resets the view stack so that the next top-level tab becomes
// the active view (and any nested views are popped).
func (t *TUI) cycleTopLevel() {
	active := t.views[0]
	for i, tab := range topLevelTabs {
		if tab.kind == active {
			t.views = []views.Kind{topLevelTabs[(i+1)%len(topLevelTabs)].kind}
			return
		}
	}
}

func (t TUI) helpBar() string {
	if len(t.models) == 0 {
		return ""
	}

	help := t.help.View(t.currentViewModel().Keys())
	user := styles.C(styles.Colors.Muted, fmt.Sprintf("u:%s", t.UserID))

	count := t.width - lipgloss.Width(help) - lipgloss.Width(user) - 2 // gap on each side of the slashes
	if count < 0 {
		return help + " " + user
	}

	return help + " " + strings.Repeat(styles.C(styles.Colors.Muted, "╱"), count) + " " + user
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

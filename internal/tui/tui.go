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
	"github.com/robherley/snips.sh/internal/tui/views/options"
	"github.com/robherley/snips.sh/internal/tui/views/prompt"
	"github.com/robherley/snips.sh/internal/tui/views/settings"
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
		views.Code:     code.New(width, t.innerViewHeight(), theme),
		views.Options:  options.New(cfg, width, t.innerViewHeight(), theme),
		views.Prompt:   prompt.New(ctx, cfg, database, width, t.innerViewHeight(), theme),
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
		// ctrl+p toggles the settings menu from anywhere (it isn't typeable)
		if msg.String() == "ctrl+p" {
			if t.currentViewKind() == views.Settings {
				return t, cmds.PopView()
			}
			return t, cmds.PushView(views.Settings)
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
			if len(t.views) == 1 {
				return t, tea.Quit
			}

			batched := []tea.Cmd{cmds.PopView()}
			if t.currentViewKind() == views.Options {
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
	case msgs.FileDeselected:
		t.file = nil
		return t, t.broadcast(msg)
	case msgs.ReloadFiles:
		t.file = nil
		// the options window's file just went stale; drop it from the stack so
		// popping the prompt lands back on the browser
		t.dropView(views.Options)
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
	content := v.Content
	if t.modalActive() {
		// modal views render bare windows; float them over the view below
		content = styles.Modal(t.width, t.innerViewHeight(), t.underlay(), content)
	}

	title := t.titleBar()
	if t.modalActive() {
		// recede along with the underlay so only the window stays bright
		title = styles.Dim(title)
	}

	v.Content = lipgloss.JoinVertical(lipgloss.Top, title, content, t.helpBar())
	v.AltScreen = true
	// force the web theme's surface color (via OSC 11) so the palette renders
	// consistently regardless of the user's terminal background
	v.BackgroundColor = styles.Colors.Black
	return v
}

// modalActive reports whether the current view is a modal window floating
// above the rest of the TUI.
func (t TUI) modalActive() bool {
	switch t.currentViewKind() {
	case views.Options, views.Prompt, views.Settings:
		return true
	}
	return false
}

// underlay is the rendered content of the topmost non-floating view in the
// stack, which Modal dims to sit beneath modal windows.
func (t TUI) underlay() string {
	for i := len(t.views) - 1; i >= 0; i-- {
		switch t.views[i] {
		case views.Options, views.Prompt, views.Settings:
			continue
		}
		return t.models[t.views[i]].View().Content
	}
	return ""
}

func (t TUI) titleBar() string {
	brand := lipgloss.NewStyle().
		Foreground(styles.Colors.Black).
		Background(t.theme).
		Padding(0, 1).
		Bold(true).
		Render("snips.sh")

	count := t.width - lipgloss.Width(brand) - 1
	if count < 0 {
		count = 0
	}

	return brand + " " + strings.Repeat(styles.BC(t.theme, "╱"), count)
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

func (t *TUI) dropView(kind views.Kind) {
	kept := t.views[:0]
	for _, v := range t.views {
		if v != kind {
			kept = append(kept, v)
		}
	}
	t.views = kept
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

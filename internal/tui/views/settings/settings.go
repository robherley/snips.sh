package settings

import (
	"context"
	"image/color"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/feedback"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/robherley/snips.sh/internal/tui/views"
)

// page is a level of the settings menu; the root lists entries that drill
// into deeper pages.
type page int

const (
	rootPage page = iota
	themePage
	deletePage
)

// entry is a selectable row on the root page that opens a deeper page.
type entry struct {
	label  string
	page   page
	danger bool
}

// entries lists the root menu; add new settings pages here.
var entries = []entry{
	{label: "theme color", page: themePage},
	{label: "delete all my data", page: deletePage, danger: true},
}

// deps are the shared dependencies settings pages need to act. Pages embed
// them by value; every field is a reference, so copies stay in sync.
type deps struct {
	ctx  context.Context
	cfg  *config.Config
	db   db.DB
	user *snips.User
}

// accent is the user's chosen theme color, used to highlight the modal.
func (d deps) accent() color.Color {
	return styles.Theme(d.user.ThemeColor)
}

// result is what a page reports back to the host after handling a key press.
type result struct {
	cmd  tea.Cmd
	quit bool
	back bool
	fb   feedback.Feedback
}

type Settings struct {
	deps
	fingerprint string

	width  int
	height int

	page     page
	cursor   int // selected entry on the root page
	feedback feedback.Feedback

	theme  themeView
	delete deleteView
}

func New(ctx context.Context, cfg *config.Config, width, height int, database db.DB, user *snips.User, fingerprint string) Settings {
	d := deps{
		ctx:  ctx,
		cfg:  cfg,
		db:   database,
		user: user,
	}

	return Settings{
		deps:        d,
		fingerprint: fingerprint,
		width:       width,
		height:      height,
		theme:       newThemeView(d),
		delete:      newDeleteView(d),
	}
}

func (s Settings) Init() tea.Cmd {
	return nil
}

func (s Settings) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		return s, nil
	case msgs.PushView:
		if msg.View == views.Settings {
			// opened fresh: start back at the root with stale feedback cleared
			s.page = rootPage
			s.cursor = 0
			s.feedback = feedback.Feedback{}
		}
		return s, nil
	case tea.KeyPressMsg:
		var res result
		switch s.page {
		case themePage:
			s.theme, res = s.theme.update(msg)
		case deletePage:
			s.delete, res = s.delete.update(msg)
		default:
			return s.updateRootPage(msg)
		}
		return s.apply(res)
	}
	return s, nil
}

// apply carries out a page's reported result.
func (s Settings) apply(res result) (tea.Model, tea.Cmd) {
	if res.quit {
		return s, tea.Quit
	}

	if res.back {
		s.page = rootPage
		s.feedback = res.fb
	}

	return s, res.cmd
}

func (s Settings) updateRootPage(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < len(entries)-1 {
			s.cursor++
		}
	case "enter":
		return s.open(entries[s.cursor].page)
	}
	return s, nil
}

// open drills into a deeper page, giving it a chance to load fresh state; a
// page that fails to load keeps us on the root with feedback.
func (s Settings) open(p page) (tea.Model, tea.Cmd) {
	s.feedback = feedback.Feedback{}

	var (
		cmd tea.Cmd
		err error
	)

	switch p {
	case themePage:
		s.theme = s.theme.enter()
	case deletePage:
		s.delete, cmd, err = s.delete.enter()
	}

	if err != nil {
		s.feedback = feedback.Error(err.Error())
		return s, nil
	}

	s.page = p
	return s, cmd
}

func (s Settings) View() tea.View {
	var title string
	var rows []string
	switch s.page {
	case themePage:
		title = "settings / theme color"
		rows = s.theme.rows()
	case deletePage:
		title = "settings / delete all my data"
		rows = s.delete.rows()
	default:
		title = "settings"
		rows = s.rootRows()
	}

	return tea.NewView(styles.ModalBody(s.accent(), title, rows...))
}

func (s Settings) rootRows() []string {
	rows := []string{
		styles.Table(styles.TableSection{Label: styles.Colors.Muted, Rows: [][2]string{
			{"user id", s.user.ID},
			{"fingerprint", s.fingerprint},
		}}),
		"",
	}

	for i, e := range entries {
		rows = append(rows, s.entryRow(e, i == s.cursor))
	}

	if !s.feedback.Empty() {
		rows = append(rows, "", s.feedback.View())
	}

	return rows
}

// entryRow renders a root menu entry with its current value.
func (s Settings) entryRow(e entry, selected bool) string {
	cursor := "  "
	nameStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)
	if selected {
		cursor = styles.BC(s.accent(), "→ ")
		nameStyle = lipgloss.NewStyle().Foreground(styles.Colors.White).Bold(true)
		if e.danger {
			nameStyle = nameStyle.Foreground(styles.Colors.Red)
		}
	}

	value := ""
	if e.page == themePage {
		opt := styles.ThemeOptions[themeIndexOf(s.user.ThemeColor)]
		swatch := lipgloss.NewStyle().Background(opt.Color).Render("   ")
		value = "  " + swatch + "  " + styles.C(styles.Colors.Muted, opt.Name)
	}

	return cursor + nameStyle.Render(e.label) + value
}

func (s Settings) Keys() help.KeyMap {
	switch s.page {
	case themePage:
		return themeKeys
	case deletePage:
		return deleteKeys
	default:
		return keys
	}
}

func (s Settings) IsCapturing() bool {
	// on deeper pages, keys (esc, q, ...) belong to this view rather than the
	// surrounding TUI; on the root page esc closes the modal at the TUI level
	return s.page != rootPage
}

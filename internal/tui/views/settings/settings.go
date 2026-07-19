package settings

import (
	"context"
	"fmt"
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
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

type Settings struct {
	ctx         context.Context
	db          db.DB
	user        *snips.User
	fingerprint string

	width  int
	height int

	page page

	// TODO(robherley): separate into sub-models for each page when we have more settings
	cursor     int             // selected entry on the root page
	themeIndex int             // selection within the theme page
	confirm    textinput.Model // typed confirmation on the delete page
	fileCount  int64

	feedback feedback.Feedback
}

func New(ctx context.Context, width, height int, database db.DB, user *snips.User, fingerprint string) Settings {
	ti := textinput.New()
	ti.CharLimit = 255
	ti.SetWidth(30)
	ti.Prompt = styles.BC(styles.Colors.Red, "> ")

	return Settings{
		ctx:         ctx,
		db:          database,
		user:        user,
		fingerprint: fingerprint,
		width:       width,
		height:      height,
		confirm:     ti,
	}
}

// themeIndexOf returns the index of the named theme option, defaulting to the
// first option.
func themeIndexOf(name string) int {
	for i, opt := range styles.ThemeOptions {
		if opt.Name == name {
			return i
		}
	}
	return 0
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
		switch s.page {
		case themePage:
			return s.updateThemePage(msg)
		case deletePage:
			return s.updateDeletePage(msg)
		}

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
	}
	return s, nil
}

// open drills into a deeper page.
func (s Settings) open(p page) (tea.Model, tea.Cmd) {
	s.page = p
	s.feedback = feedback.Feedback{}
	switch p {
	case themePage:
		s.themeIndex = themeIndexOf(s.user.ThemeColor)
	case deletePage:
		count, err := s.db.CountFilesByUser(s.ctx, s.user.ID)
		if err != nil {
			s.page = rootPage
			s.feedback = feedback.Error("failed to count files: " + err.Error())
			return s, nil
		}
		s.fileCount = count
		s.confirm.Reset()
		s.confirm.Focus()
		return s, textinput.Blink
	}
	return s, nil
}

func (s Settings) updateDeletePage(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if s.confirm.Value() != s.user.ID {
			s.feedback = feedback.Error("please type your user id to confirm")
			return s, nil
		}
		return s.deleteEverything()
	case "esc":
		s.page = rootPage
		s.feedback = feedback.Feedback{}
		return s, nil
	}

	// everything else is typed into the confirmation input
	var cmd tea.Cmd
	s.confirm, cmd = s.confirm.Update(msg)
	return s, cmd
}

func (s Settings) deleteEverything() (tea.Model, tea.Cmd) {
	count, err := s.db.DeleteFilesByUser(s.ctx, s.user.ID)
	if err != nil {
		s.feedback = feedback.Error("failed to delete: " + err.Error())
		return s, nil
	}

	metrics.IncrCounter([]string{"file", "delete", "all"}, 1)
	logger.From(s.ctx).Info("deleted all user files", "user_id", s.user.ID, "count", count)

	s.page = rootPage
	deleted := fmt.Sprintf("deleted %d files", count)
	if count == 1 {
		deleted = "deleted 1 file"
	}
	s.feedback = feedback.Success(deleted)

	return s, cmds.ReloadFiles(s.db, s.user.ID)
}

func (s Settings) updateThemePage(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if s.themeIndex > 0 {
			s.themeIndex--
		}
	case "down", "j":
		if s.themeIndex < len(styles.ThemeOptions)-1 {
			s.themeIndex++
		}
	case "enter":
		s.page = rootPage
		return s.saveTheme(s.themeIndex)
	case "esc":
		s.page = rootPage
	case "q":
		// the view captures input on deeper pages, so quit needs handling here
		return s, tea.Quit
	}
	return s, nil
}

func (s Settings) saveTheme(index int) (tea.Model, tea.Cmd) {
	name := styles.ThemeOptions[index].Name
	prev := s.user.ThemeColor
	s.user.ThemeColor = name
	if err := s.db.UpdateUser(s.ctx, s.user); err != nil {
		s.user.ThemeColor = prev
		s.feedback = feedback.Error("failed to save: " + err.Error())
		return s, nil
	}

	s.feedback = feedback.Success("theme set to " + name)

	return s, func() tea.Msg { return msgs.ThemeChanged{Color: styles.ThemeOptions[index].Color} }
}

func (s Settings) View() tea.View {
	var title string
	var rows []string
	switch s.page {
	case themePage:
		title = "settings / theme color"
		rows = s.themeRows()
	case deletePage:
		title = "settings / delete all my data"
		rows = s.deleteRows()
	default:
		title = "settings"
		rows = s.rootRows()
	}

	return tea.NewView(styles.ModalBody(s.accent(), title, rows...))
}

// accent is the user's chosen theme color, used to highlight the modal.
func (s Settings) accent() color.Color {
	return styles.Theme(s.user.ThemeColor)
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

func (s Settings) themeRows() []string {
	rows := make([]string, 0, len(styles.ThemeOptions))
	for i, opt := range styles.ThemeOptions {
		swatch := lipgloss.NewStyle().Background(opt.Color).Render("   ")
		cursor := "  "
		nameStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)
		if i == s.themeIndex {
			cursor = lipgloss.NewStyle().Foreground(opt.Color).Bold(true).Render("→ ")
			nameStyle = lipgloss.NewStyle().Foreground(styles.Colors.White).Bold(true)
		}
		rows = append(rows, cursor+swatch+"  "+nameStyle.Render(opt.Name))
	}
	return rows
}

// deleteRows renders the delete confirmation page.
func (s Settings) deleteRows() []string {
	warnStyle := lipgloss.NewStyle().Foreground(styles.Colors.Red)
	mutedStyle := lipgloss.NewStyle().Foreground(styles.Colors.Muted)

	things := fmt.Sprintf("all %d of your files", s.fileCount)
	switch s.fileCount {
	case 0:
		things = "your files (you have none)"
	case 1:
		things = "your only file"
	}

	rows := []string{
		warnStyle.Render(fmt.Sprintf("this permanently deletes %s.", things)),
		mutedStyle.Render("type your user id (" + s.user.ID + ") to confirm"),
		"",
		s.confirm.View(),
	}

	if !s.feedback.Empty() && s.feedback.Err {
		rows = append(rows, "", s.feedback.View())
	}

	return rows
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

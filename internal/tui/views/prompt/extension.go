package prompt

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/armon/go-metrics"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/feedback"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

// extensionDialog changes a file's type via a filterable selector.
type extensionDialog struct {
	selector list.Model
}

func newExtensionDialog(width int) *extensionDialog {
	return &extensionDialog{selector: newExtensionSelector(width)}
}

func (d *extensionDialog) title() string {
	return "edit extension"
}

func (d *extensionDialog) question(*snips.File) string {
	return "What extension do you want to change the file to?"
}

func (d *extensionDialog) init() tea.Cmd {
	return SelectorInitCmd
}

func (d *extensionDialog) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// while its filter input is focused, the selector routes every key to
		// the filter, so move the cursor ourselves to keep the arrow keys
		// working
		switch msg.Code {
		case tea.KeyUp:
			d.selector.CursorUp()
			return nil
		case tea.KeyDown:
			d.selector.CursorDown()
			return nil
		}
	case SelectorInitMsg:
		// bit of a hack to get the selector to filter on init
		d.selector, cmd = d.selector.Update(tea.KeyPressMsg{
			Code: '/',
			Text: "/",
		})
		return cmd
	}

	d.selector, cmd = d.selector.Update(msg)
	return cmd
}

func (d *extensionDialog) view() string {
	return d.selector.View()
}

func (d *extensionDialog) resize(width int) {
	d.selector.SetWidth(width)
}

func (d *extensionDialog) submit(e env) tea.Cmd {
	// SelectedItem is nil when the filter matches nothing
	item, ok := d.selector.SelectedItem().(selectorItem)
	if !ok {
		return SetPromptErrorCmd(errors.New("no matching extension selected"))
	}
	old := e.file.Type
	e.file.Type = item.name

	if err := e.db.UpdateFile(e.ctx, e.file); err != nil {
		return SetPromptErrorCmd(err)
	}

	metrics.IncrCounterWithLabels([]string{"file", "change", "type"}, 1, []metrics.Label{
		{Name: "old", Value: old},
		{Name: "new", Value: e.file.Type},
	})
	logger.From(e.ctx).Info("updated file type", "file", e.file.ID, "old_type", old, "new_type", e.file.Type)

	msg := feedback.Success(fmt.Sprintf("file %q extension set to %q", e.file.ID, item.name))
	return tea.Batch(cmds.ReloadFiles(e.db, e.file.UserID), SetPromptFeedbackCmd(msg, true))
}

var extensions = func() []selectorItem {
	reg := lexers.GlobalLexerRegistry

	items := make([]selectorItem, 0, len(reg.Lexers))

	for _, lexer := range reg.Lexers {
		item := selectorItem{
			name: strings.ToLower(lexer.Config().Name),
		}

		if len(lexer.Config().Filenames) != 0 {
			item.ext = strings.TrimPrefix(lexer.Config().Filenames[0], "*.")
		}

		items = append(items, item)
	}

	return items
}()

func newExtensionSelector(width int) list.Model {
	items := []list.Item{}

	for _, ext := range extensions {
		items = append(items, ext)
	}

	li := list.New(items, selectorItemDelegate{}, width, 12)
	li.SetShowTitle(false)
	li.SetShowPagination(false)
	li.SetShowHelp(false)
	li.SetShowStatusBar(false)

	tiStyles := textinput.DefaultDarkStyles()
	tiStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.Colors.Yellow)
	tiStyles.Blurred.Prompt = lipgloss.NewStyle().Foreground(styles.Colors.Yellow)
	li.FilterInput.SetStyles(tiStyles)

	return li
}

type selectorItem struct {
	name string
	ext  string
}

func (i selectorItem) FilterValue() string {
	return i.name + i.ext
}

type selectorItemDelegate struct{}

func (d selectorItemDelegate) Height() int {
	return 1
}

func (d selectorItemDelegate) Spacing() int {
	return 0
}

func (d selectorItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d selectorItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(selectorItem)
	if !ok {
		return
	}

	str := i.name
	if i.ext != "" {
		str += fmt.Sprintf(" (%s)", i.ext)
	}

	color := styles.Colors.Muted
	prefix := "  "
	if m.Index() == index {
		prefix = styles.C(styles.Colors.Yellow, "→ ")
		color = styles.Colors.White
	}

	fmt.Fprintf(w, "%s%s", prefix, styles.C(color, str))
}

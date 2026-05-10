package prompt

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

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

func NewExtensionSelector(width int) list.Model {
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

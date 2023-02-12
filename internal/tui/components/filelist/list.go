package filelist

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robherley/snips.sh/internal/tui/messages"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type List struct {
	list list.Model
	keys *keyMap
}

func New(width, height int, files []ListItem) *List {
	del := list.NewDefaultDelegate()

	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Border(lipgloss.ThickBorder(), false).
		BorderLeft(true).
		BorderForeground(styles.ColorSecondary).
		Foreground(styles.ColorSecondary)
	del.Styles.SelectedTitle = selectedStyle.Copy().Bold(true).Foreground(styles.Colors.White)
	del.Styles.SelectedDesc = selectedStyle.Copy()

	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = f
	}

	flist := list.New(
		items,
		del,
		width,
		height,
	)

	keys := newKeyMap()
	flist.AdditionalFullHelpKeys = keys.Bindings
	flist.AdditionalShortHelpKeys = keys.Bindings

	flist.SetStatusBarItemName("file", "files")
	flist.SetShowTitle(false)

	flist.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render("•")
	flist.Styles.NoItems = lipgloss.NewStyle().Foreground(styles.Colors.Muted).MarginLeft(1)

	return &List{
		list: flist,
		keys: keys,
	}
}

func (fl *List) Init() tea.Cmd {
	return nil
}

func (fl *List) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fl.list.SetWidth(msg.Width)
		fl.list.SetHeight(msg.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, fl.keys.selectFile):
			if fl.list.SelectedItem() == nil {
				break
			}

			return fl, func() tea.Msg {
				return messages.SelectedFile{
					ID: fl.list.SelectedItem().(ListItem).ID,
				}
			}
		}
	}
	fl.list, cmd = fl.list.Update(msg)
	return fl, cmd
}

func (fl *List) View() string {
	fl.list.SetShowStatusBar(len(fl.list.Items()) != 0)

	return fl.list.View()
}

type keyMap struct {
	selectFile key.Binding
	deleteFile key.Binding
	signFile   key.Binding
}

func (k *keyMap) Bindings() []key.Binding {
	return []key.Binding{
		k.selectFile,
		k.deleteFile,
		k.signFile,
	}
}

func newKeyMap() *keyMap {
	return &keyMap{
		selectFile: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view"),
		),
		deleteFile: key.NewBinding(
			key.WithKeys("delete", "backspace"),
			key.WithHelp("delete", "remove"),
		),
		signFile: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sign"),
		),
	}
}

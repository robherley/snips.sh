package filelist

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/robherley/snips.sh/internal/tui/views"
)

type FileList struct {
	list list.Model
}

func New(width, height int, files []ListItem) FileList {
	del := list.NewDefaultDelegate()

	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Border(lipgloss.ThickBorder(), false).
		BorderLeft(true).
		BorderForeground(styles.Colors.Blue).
		Foreground(styles.Colors.Blue)
	del.Styles.SelectedTitle = selectedStyle.Copy().Bold(true).Foreground(styles.Colors.White)
	del.Styles.SelectedDesc = selectedStyle.Copy()

	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = f
	}

	ls := list.New(
		items,
		del,
		width,
		height,
	)

	ls.SetStatusBarItemName("file", "files")
	ls.SetShowTitle(false)

	ls.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(styles.Colors.Blue).Render("•")
	ls.Paginator.InactiveDot = lipgloss.NewStyle().Foreground(styles.Colors.Muted).Render("◦")
	ls.Styles.NoItems = lipgloss.NewStyle().Foreground(styles.Colors.Muted).MarginLeft(1)

	return FileList{
		list: ls,
	}
}

func (m FileList) Init() tea.Cmd {
	return nil
}

func (m FileList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.list.SelectedItem() == nil {
				break
			}

			return m, tea.Batch(
				cmds.SelectFile(m.list.SelectedItem().(ListItem).ID),
				cmds.PushView(views.FileOptions),
			)
		}
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m FileList) View() string {
	m.list.SetShowStatusBar(len(m.list.Items()) != 0)

	return m.list.View()
}

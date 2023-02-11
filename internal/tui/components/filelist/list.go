package filelist

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

type List struct {
	list list.Model
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

	li := list.New(
		items,
		del,
		width,
		height,
	)

	li.SetStatusBarItemName("file", "files")
	li.SetShowTitle(false)

	li.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render("•")
	li.Styles.NoItems = lipgloss.NewStyle().Foreground(styles.Colors.Muted).MarginLeft(1)

	return &List{
		list: li,
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
	}
	fl.list, cmd = fl.list.Update(msg)
	return fl, cmd
}

func (fl *List) View() string {
	fl.list.SetShowStatusBar(len(fl.list.Items()) != 0)

	return fl.list.View()
}

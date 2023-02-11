package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/db"
)

type fileList struct {
	Width  int
	Height int

	list  list.Model
	files []db.File
}

func NewFileList(width, height int) *fileList {
	del := list.NewDefaultDelegate()

	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Border(lipgloss.ThickBorder(), false).
		BorderLeft(true).
		BorderForeground(ColorSecondary).
		Foreground(ColorSecondary)
	del.Styles.SelectedTitle = selectedStyle.Copy().Bold(true).Foreground(Colors.White)
	del.Styles.SelectedDesc = selectedStyle.Copy()

	li := list.New(
		[]list.Item{},
		del,
		width,
		height,
	)

	li.SetStatusBarItemName("file", "files")
	li.SetShowTitle(false)

	li.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(ColorSecondary).Render("•")
	li.Styles.NoItems = lipgloss.NewStyle().Foreground(Colors.Muted).MarginLeft(1)

	return &fileList{
		Width:  width,
		Height: height,
		list:   li,
	}
}

func (fl *fileList) Init() tea.Cmd {
	return fl.list.StartSpinner()
}

func (fl *fileList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fl.Width = msg.Width
		fl.Height = msg.Height
		fl.list.SetWidth(msg.Width)
		fl.list.SetHeight(msg.Height)
	case FilesMsg:
		fl.handleFiles(msg.Files)
	}
	fl.list, cmd = fl.list.Update(msg)
	return fl, cmd
}

func (fl *fileList) View() string {
	fl.list.SetShowStatusBar(len(fl.files) != 0)

	return fl.list.View()
}

func (fl *fileList) handleFiles(files []db.File) {
	fl.files = files

	items := make([]list.Item, len(fl.files))
	for i, file := range fl.files {
		title := file.ID
		if file.Private {
			title += " 🔒"
		}

		attr := []string{
			strings.ToLower(file.Type),
			humanize.Bytes(file.Size),
			humanize.Time(file.CreatedAt),
		}
		description := strings.Join(attr, " • ")

		items[i] = filelistitem{
			title,
			description,
		}
	}

	fl.list.SetItems(items)
}

type filelistitem struct {
	title, desc string
}

func (i filelistitem) Title() string       { return i.title }
func (i filelistitem) Description() string { return i.desc }
func (i filelistitem) FilterValue() string { return i.title + i.desc }

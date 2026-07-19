package browser

import (
	"fmt"
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/robherley/snips.sh/internal/tui/views"
	"github.com/robherley/snips.sh/internal/tui/views/prompt"
)

type Browser struct {
	cfg    *config.Config
	list   list.Model
	height int
	width  int
	theme  color.Color
}

func New(cfg *config.Config, width, height int, files []*snips.File, theme color.Color) Browser {
	l := list.New(toItems(files), newItemDelegate(theme), width, height)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetStatusBarItemName("file", "files")

	l.Styles.TitleBar = l.Styles.TitleBar.PaddingTop(1)
	l.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(theme).Render("■")
	l.Paginator.InactiveDot = lipgloss.NewStyle().Foreground(styles.Colors.Muted).Render("▪")

	return Browser{
		cfg:    cfg,
		list:   l,
		width:  width,
		height: height,
		theme:  theme,
	}
}

func (bwsr Browser) Init() tea.Cmd {
	return nil
}

func (bwsr Browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// don't intercept anything while the user is typing in the filter
		if bwsr.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "tab":
			file := bwsr.selectedFile()
			if file == nil {
				return bwsr, nil
			}
			return bwsr, tea.Batch(
				cmds.SelectFile(file.ID),
				cmds.PushView(views.Options),
			)
		case "enter":
			file := bwsr.selectedFile()
			if file == nil {
				return bwsr, nil
			}
			return bwsr, tea.Batch(
				cmds.SelectFile(file.ID),
				cmds.PushView(views.Code),
			)
		case "x":
			file := bwsr.selectedFile()
			if file == nil {
				return bwsr, nil
			}
			return bwsr, tea.Batch(
				cmds.SelectFile(file.ID),
				prompt.SetPromptKindCmd(prompt.DeleteFile),
				cmds.PushView(views.Prompt),
			)
		case "s":
			file := bwsr.selectedFile()
			if file == nil || !file.Private {
				// signed urls only make sense for private files
				return bwsr, nil
			}
			return bwsr, tea.Batch(
				cmds.SelectFile(file.ID),
				prompt.SetPromptKindCmd(prompt.GenerateSignedURL),
				cmds.PushView(views.Prompt),
			)
		}
	case tea.WindowSizeMsg:
		bwsr.width, bwsr.height = msg.Width, msg.Height
		// reserve 2 rows at the bottom: our combined status + pagination,
		// and a gap above the help bar
		bwsr.list.SetSize(msg.Width, max(msg.Height-2, 0))
	case msgs.ReloadFiles:
		bwsr.list.SetItems(toItems(msg.Files))
	case msgs.ThemeChanged:
		bwsr.theme = msg.Color
		bwsr.list.SetDelegate(newItemDelegate(msg.Color))
		bwsr.list.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(msg.Color).Render("■")
	}

	var cmd tea.Cmd
	bwsr.list, cmd = bwsr.list.Update(msg)
	return bwsr, cmd
}

func (bwsr Browser) View() tea.View {
	return tea.NewView(bwsr.viewContent())
}

func (bwsr Browser) viewContent() string {
	if len(bwsr.list.Items()) == 0 {
		addr := bwsr.cfg.HTTP.External.String()
		link := lipgloss.NewStyle().Foreground(bwsr.theme).Hyperlink(addr).Render(addr)
		return lipgloss.NewStyle().
			PaddingTop(1).
			PaddingBottom(1).
			Render(styles.C(bwsr.theme, "No files found!\nLearn how to get started at: ") + link)
	}

	return lipgloss.JoinVertical(lipgloss.Top, bwsr.list.View(), bwsr.statusBar(), "")
}

// statusBar renders the file count, optional filter info, and pagination dots
// on a single line below the list.
func (bwsr Browser) statusBar() string {
	visible := bwsr.list.VisibleItems()
	total := len(bwsr.list.Items())

	itemName := "files"
	if len(visible) == 1 {
		itemName = "file"
	}

	var status string
	switch bwsr.list.FilterState() {
	case list.Filtering:
		if len(visible) == 0 {
			status = "Nothing matched"
		} else {
			status = fmt.Sprintf("%d %s", len(visible), itemName)
		}
	case list.FilterApplied:
		f := bwsr.list.FilterInput.Value()
		status = fmt.Sprintf("%q  %d %s", f, len(visible), itemName)
	default:
		if total == 0 {
			status = "No files"
		} else {
			status = fmt.Sprintf("%d %s", len(visible), itemName)
		}
	}

	if filtered := total - len(visible); filtered > 0 {
		status += fmt.Sprintf("  •  %d filtered", filtered)
	}

	pagination := ""
	if bwsr.list.Paginator.TotalPages > 1 {
		pagination = "  " + bwsr.list.Paginator.View()
	}

	return lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(styles.Colors.Muted).
		Render(status + pagination)
}

func (bwsr Browser) Keys() help.KeyMap {
	return keys
}

func (bwsr Browser) IsCapturing() bool {
	return bwsr.list.FilterState() == list.Filtering
}

// selectedFile returns the file currently highlighted in the list, or nil if
// the list is empty.
func (bwsr Browser) selectedFile() *snips.File {
	item, ok := bwsr.list.SelectedItem().(fileItem)
	if !ok {
		return nil
	}
	return item.file
}

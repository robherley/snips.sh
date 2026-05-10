package browser

import (
	"fmt"

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
)

type Browser struct {
	cfg    *config.Config
	list   list.Model
	height int
	width  int

	options struct {
		focused bool
		index   int
	}
}

func New(cfg *config.Config, width, height int, files []*snips.File) Browser {
	l := list.New(toItems(files), newItemDelegate(), width, height)
	l.SetShowTitle(false)      // the F1 tab in the title bar already labels this view
	l.SetShowHelp(false)       // help is rendered at the TUI level
	l.SetShowStatusBar(false)  // we render our own combined status + pagination row
	l.SetShowPagination(false) // ditto
	l.SetStatusBarItemName("file", "files")

	// breathing room above the filter input
	l.Styles.TitleBar = l.Styles.TitleBar.PaddingTop(1)

	// bigger pagination glyphs (• → ● / ○)
	l.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(styles.Colors.Primary).Render("●")
	l.Paginator.InactiveDot = lipgloss.NewStyle().Foreground(styles.Colors.Muted).Render("○")

	return Browser{
		cfg:    cfg,
		list:   l,
		width:  width,
		height: height,
	}
}

func (bwsr Browser) Init() tea.Cmd {
	return nil
}

func (bwsr Browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.PushView, msgs.PopView:
		bwsr.options.index = 0
	case tea.KeyPressMsg:
		// don't intercept anything while the user is typing in the filter
		if bwsr.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "tab":
			if bwsr.options.focused {
				bwsr.options.index = 0
			}
			bwsr.options.focused = !bwsr.options.focused
			return bwsr, nil
		case "esc":
			if bwsr.options.focused {
				bwsr.options.focused = false
				return bwsr, nil
			}
		case "enter":
			if bwsr.options.focused {
				return bwsr.handleOptionsEnter()
			}
			file := bwsr.selectedFile()
			if file == nil {
				return bwsr, nil
			}
			return bwsr, tea.Batch(
				cmds.SelectFile(file.ID),
				cmds.PushView(views.Code),
			)
		}

		if bwsr.options.focused {
			return bwsr.handleOptionsNavigation(msg)
		}
	case tea.WindowSizeMsg:
		bwsr.width, bwsr.height = msg.Width, msg.Height
		// reserve 1 row at the bottom for our combined status + pagination
		bwsr.list.SetSize(msg.Width, msg.Height-1)
	case msgs.ReloadFiles:
		bwsr.list.SetItems(toItems(msg.Files))
		bwsr.options.focused = false
		bwsr.options.index = 0
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
		return lipgloss.NewStyle().
			PaddingTop(1).
			PaddingBottom(1).
			Foreground(styles.Colors.Primary).
			Render(fmt.Sprintf("No files found!\nLearn how to get started at: %s", bwsr.cfg.HTTP.External.String()))
	}

	if bwsr.options.focused {
		return bwsr.renderModal()
	}
	return lipgloss.JoinVertical(lipgloss.Top, bwsr.list.View(), bwsr.statusBar())
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
	return getKeyMap(bwsr.IsOptionsFocused())
}

func (bwsr Browser) IsOptionsFocused() bool {
	return bwsr.options.focused
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

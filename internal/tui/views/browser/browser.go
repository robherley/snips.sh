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
	l.SetShowTitle(false) // the F1 tab in the title bar already labels this view
	l.SetStatusBarItemName("file", "files")
	l.SetShowHelp(false) // help is rendered at the TUI level

	// unify the status bar grays so "Nothing matched • 48 filtered" is one tone
	l.Styles.StatusBar = l.Styles.StatusBar.Foreground(styles.Colors.Muted)
	l.Styles.StatusEmpty = l.Styles.StatusEmpty.Foreground(styles.Colors.Muted)
	l.Styles.StatusBarFilterCount = l.Styles.StatusBarFilterCount.Foreground(styles.Colors.Muted)

	// breathing room above the filter input
	l.Styles.TitleBar = l.Styles.TitleBar.PaddingTop(1)

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
		bwsr.list.SetSize(msg.Width, msg.Height)
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
	return bwsr.list.View()
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

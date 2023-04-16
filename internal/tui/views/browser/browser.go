package browser

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/msgs"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

const (
	BreakPoint = 80
	Selector   = "→ "
)

type Browser struct {
	cfg    *config.Config
	files  []*snips.File
	height int
	width  int

	options struct {
		focused bool
		index   int
	}

	table struct {
		index             int
		offset            int
		preRendered       [][]string
		preRenderedWidths []int
	}
}

func New(cfg *config.Config, width, height int, files []*snips.File) Browser {
	bwsr := Browser{
		cfg:    cfg,
		files:  files,
		height: height,
		width:  width,
	}

	bwsr.preRender()
	return bwsr
}

func (bwsr Browser) Init() tea.Cmd {
	return nil
}

func (bwsr Browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case msgs.PushView, msgs.PopView:
		bwsr.options.index = 0
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if bwsr.options.focused {
				bwsr.options.focused = false
			}
		case "tab":
			if bwsr.options.focused {
				bwsr.options.index = 0
			}
			bwsr.options.focused = !bwsr.options.focused
		}

		if bwsr.options.focused {
			return bwsr.handleOptionsNavigation(msg)
		}

		return bwsr.handleTableNavigation(msg)
	case tea.WindowSizeMsg:
		bwsr.width, bwsr.height = msg.Width, msg.Height
	case msgs.ReloadFiles:
		bwsr.files = msg.Files
		bwsr.preRender()
		bwsr.options.focused = false
		bwsr.options.index = 0
		bwsr.table.index = 0
	}
	return bwsr, cmd
}

func (bwsr Browser) View() string {
	if bwsr.width < BreakPoint {
		if bwsr.options.focused {
			return bwsr.renderOptions()
		}
		return bwsr.renderTable()
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, bwsr.renderTable(), bwsr.renderSeparator(), bwsr.renderOptions())
}

func (bwsr Browser) IsOptionsFocused() bool {
	return bwsr.options.focused
}

func (bwsr Browser) renderSeparator() string {
	separatorStyle := lipgloss.NewStyle().
		Height(bwsr.height).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(styles.Colors.Muted).
		MarginLeft(1)
		// options will supply right padding since it needs it for collapse mode

	if bwsr.options.focused {
		separatorStyle = separatorStyle.BorderForeground(styles.Colors.Primary).BorderStyle(lipgloss.ThickBorder())
	}

	return separatorStyle.Render(" ")
}

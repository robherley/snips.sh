package filetable

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/robherley/snips.sh/internal/tui/views"
)

type column struct {
	width  int
	render func(file *snips.File) string
}

var cols = []column{
	{
		width: 12,
		render: func(file *snips.File) string {
			return file.ID
		},
	},
	{
		width: 10,
		render: func(file *snips.File) string {
			return humanize.Bytes(file.Size)
		},
	},
	{
		width: 13,
		render: func(file *snips.File) string {
			return humanize.Time(file.CreatedAt)
		},
	},
	{
		width: 20,
		render: func(file *snips.File) string {
			ftype := strings.ToLower(file.Type)
			if file.Private {
				ftype += " (private)"
			}

			return ftype
		},
	},
}

type FileTable struct {
	files  []*snips.File
	height int
	width  int
	index  int
	offset int
}

func New(width, height int, files []*snips.File) FileTable {
	// only first 10 files
	// if len(files) > 10 {
	// 	files = files[:10]
	// }

	return FileTable{
		files:  files,
		height: height,
		width:  width,
		index:  0,
		offset: 0,
	}
}

func (ft FileTable) Init() tea.Cmd {
	return nil
}

func (ft FileTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// move down
		case "down", "j":
			if ft.index < len(ft.files)-1 {
				ft.index++
			}

			if ft.index > ft.offset+ft.numRenderedRows()-1 {
				ft.offset++
			}
		// move up
		case "up", "k":
			if ft.index > 0 {
				ft.index--
			}

			if ft.index < ft.offset {
				ft.offset--
			}
		// move to next "page"
		case "right", "l":
			ft.index += ft.numRenderedRows()
			if ft.index > len(ft.files)-1 {
				ft.index = len(ft.files) - 1
			}

			if ft.index > ft.offset+ft.numRenderedRows()-1 {
				ft.offset = ft.index - ft.numRenderedRows() + 1
			}
		// move to previous "page"
		case "left", "h":
			ft.index -= ft.numRenderedRows()
			if ft.index < 0 {
				ft.index = 0
			}

			if ft.index < ft.offset {
				ft.offset = ft.index
			}
		case "enter":
			if ft.index >= len(ft.files) {
				break
			}

			return ft, tea.Batch(
				cmds.SelectFile(ft.files[ft.index].ID),
				cmds.PushView(views.FileOptions),
			)
		}
	case tea.WindowSizeMsg:
		ft.width, ft.height = msg.Width, msg.Height
	}
	return ft, cmd
}

func (ft FileTable) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		ft.renderHeader(),
		ft.renderRows(),
		ft.renderFooter(),
	)
}

func (ft FileTable) renderHeader() string {
	return ""
}

func (ft FileTable) renderFooter() string {
	return lipgloss.
		NewStyle().
		Foreground(styles.Colors.Blue).
		Render(fmt.Sprintf("\n [%d/%d files] (max %d)", ft.index+1, len(ft.files), snips.FileLimit))
}

func (ft FileTable) renderRow(i int) string {
	row := strings.Builder{}
	rowStyle := lipgloss.NewStyle()

	if i == ft.index {
		row.WriteString("→ ")
		rowStyle = rowStyle.Foreground(styles.Colors.Yellow).Bold(true)
	} else {
		rowStyle = rowStyle.Foreground(styles.Colors.Muted)
		row.WriteString("  ")
	}

	file := ft.files[i]
	for _, col := range cols {
		row.WriteString(rowStyle.Render(fmt.Sprintf("%-*s", col.width, col.render(file))))
	}

	return rowStyle.Render(row.String())
}

func (ft FileTable) renderRows() string {
	numRows := ft.numRenderedRows()
	rows := make([]string, 0, numRows)
	for i := ft.offset; i < numRows+ft.offset; i++ {
		if i >= len(ft.files) {
			// render empty row
			rows = append(rows, "")
			continue
		}

		rows = append(rows, ft.renderRow(i))
	}

	return lipgloss.JoinVertical(lipgloss.Top, rows...)
}

func (ft FileTable) numRenderedRows() int {
	padding := lipgloss.Height(ft.renderHeader()) + lipgloss.Height(ft.renderFooter())
	return min(ft.height-padding, len(ft.files))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

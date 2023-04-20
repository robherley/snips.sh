package browser

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
	name   string
	render func(file *snips.File) string
}

var columns = []column{
	{
		name: "ID",
		render: func(file *snips.File) string {
			return file.ID
		},
	},
	{
		name: "Size",
		render: func(file *snips.File) string {
			return humanize.Bytes(file.Size)
		},
	},
	{
		name: "Modified",
		render: func(file *snips.File) string {
			return humanize.Time(file.UpdatedAt)
		},
	},
	{
		name: "Type",
		render: func(file *snips.File) string {
			return strings.ToLower(file.Type)
		},
	},
	{
		name: "", // display an empty column for visibility
		render: func(file *snips.File) string {
			if file.Private {
				return "(p)"
			}

			return ""
		},
	},
}

func (bwsr Browser) handleTableNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	numRows := bwsr.numRowsToRender()

	switch msg.String() {
	// move down
	case "down", "j":
		if bwsr.table.index < len(bwsr.files)-1 {
			bwsr.table.index++
		}

		if bwsr.table.index > bwsr.table.offset+numRows-1 {
			bwsr.table.offset++
		}
	// move up
	case "up", "k":
		if bwsr.table.index > 0 {
			bwsr.table.index--
		}

		if bwsr.table.index < bwsr.table.offset {
			bwsr.table.offset--
		}
	// move to next "page"
	case "right", "l":
		bwsr.table.index += numRows
		if bwsr.table.index > len(bwsr.files)-1 {
			bwsr.table.index = len(bwsr.files) - 1
		}

		if bwsr.table.index > bwsr.table.offset+numRows-1 {
			bwsr.table.offset = bwsr.table.index - numRows + 1
		}
	// move to previous "page"
	case "left", "h":
		bwsr.table.index -= numRows
		if bwsr.table.index < 0 {
			bwsr.table.index = 0
		}

		if bwsr.table.index < bwsr.table.offset {
			bwsr.table.offset = bwsr.table.index
		}
	case "enter":
		if bwsr.table.index >= len(bwsr.files) {
			break
		}

		return bwsr, tea.Batch(
			cmds.SelectFile(bwsr.files[bwsr.table.index].ID),
			cmds.PushView(views.Code),
		)
	}

	return bwsr, nil
}

func (bwsr Browser) renderTable() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		bwsr.renderHeader(),
		bwsr.renderRows(),
		bwsr.renderFooter(),
	)
}

func (bwsr Browser) renderHeader() string {
	if len(bwsr.files) == 0 {
		return lipgloss.NewStyle().Foreground(styles.Colors.Primary).Render("No files found")
	}

	header := make([]string, 0, len(columns))
	for i, col := range columns {
		header = append(header, fmt.Sprintf("%-*s", bwsr.table.preRenderedWidths[i], col.name))
	}

	return lipgloss.NewStyle().PaddingTop(1).Bold(true).Render("  " + strings.Join(header, "  "))
}

func (bwsr Browser) renderFooter() string {
	if len(bwsr.files) == 0 {
		return ""
	}

	return lipgloss.NewStyle().
		Foreground(styles.Colors.Primary).
		Padding(1).
		Render(fmt.Sprintf("%d/%d files", bwsr.table.index+1, len(bwsr.files)))
}

func (bwsr Browser) renderRows() string {
	numRows := bwsr.numRowsToRender()
	rows := make([]string, 0, numRows)
	for i := bwsr.table.offset; i < numRows+bwsr.table.offset; i++ {
		if i >= len(bwsr.files) {
			// render empty row
			rows = append(rows, "")
			continue
		}

		rows = append(rows, bwsr.renderRow(i))
	}

	return lipgloss.JoinVertical(lipgloss.Top, rows...)
}

func (bwsr Browser) renderRow(i int) string {
	preRenderedRow := bwsr.table.preRendered[i]
	row := make([]string, 0, len(preRenderedRow))
	for i, col := range preRenderedRow {
		row = append(row, fmt.Sprintf("%-*s", bwsr.table.preRenderedWidths[i], col))
	}

	rowStyle := lipgloss.NewStyle()

	color := styles.Colors.Muted
	prefix := "  "
	if i == bwsr.table.index && !bwsr.options.focused {
		color = styles.Colors.White
		prefix = styles.BC(styles.Colors.Primary, "→ ")
		rowStyle = rowStyle.Bold(true)
	}

	return prefix + rowStyle.Foreground(color).Render(strings.Join(row, "  "))
}

func (bwsr Browser) numRowsToRender() int {
	padding := lipgloss.Height(bwsr.renderHeader()) + lipgloss.Height(bwsr.renderFooter())
	return min(bwsr.height-padding, len(bwsr.files))
}

func (bwsr *Browser) preRender() {
	bwsr.table.preRendered = make([][]string, len(bwsr.files))
	bwsr.table.preRenderedWidths = make([]int, len(columns))

	for i := range columns {
		bwsr.table.preRenderedWidths[i] = lipgloss.Width(columns[i].name)
	}

	for i, file := range bwsr.files {
		row := make([]string, len(columns))
		for j, col := range columns {
			row[j] = col.render(file)
			bwsr.table.preRenderedWidths[j] = max(bwsr.table.preRenderedWidths[j], lipgloss.Width(row[j]))
		}

		bwsr.table.preRendered[i] = row
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

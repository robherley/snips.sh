package browser

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
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

type Browser struct {
	cfg               *config.Config
	files             []*snips.File
	height            int
	width             int
	index             int
	offset            int
	optionsFocused    bool
	preRendered       [][]string
	preRenderedWidths []int
}

func New(cfg *config.Config, width, height int, files []*snips.File) Browser {
	bwsr := Browser{
		cfg:    cfg,
		files:  files,
		height: height,
		width:  width,
		index:  0,
		offset: 0,
	}

	bwsr.preRender()
	return bwsr
}

func (bwsr Browser) Init() tea.Cmd {
	return nil
}

func (bwsr Browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	numRows := bwsr.numRowsToRender()

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			bwsr.optionsFocused = !bwsr.optionsFocused
		// move down
		case "down", "j":
			if bwsr.index < len(bwsr.files)-1 {
				bwsr.index++
			}

			if bwsr.index > bwsr.offset+numRows-1 {
				bwsr.offset++
			}
		// move up
		case "up", "k":
			if bwsr.index > 0 {
				bwsr.index--
			}

			if bwsr.index < bwsr.offset {
				bwsr.offset--
			}
		// move to next "page"
		case "right", "l":
			bwsr.index += numRows
			if bwsr.index > len(bwsr.files)-1 {
				bwsr.index = len(bwsr.files) - 1
			}

			if bwsr.index > bwsr.offset+numRows-1 {
				bwsr.offset = bwsr.index - numRows + 1
			}
		// move to previous "page"
		case "lebwsr", "h":
			bwsr.index -= numRows
			if bwsr.index < 0 {
				bwsr.index = 0
			}

			if bwsr.index < bwsr.offset {
				bwsr.offset = bwsr.index
			}
		case "enter":
			if bwsr.index >= len(bwsr.files) {
				break
			}

			return bwsr, tea.Batch(
				cmds.SelectFile(bwsr.files[bwsr.index].ID),
				cmds.PushView(views.Code),
			)
		}
	case tea.WindowSizeMsg:
		bwsr.width, bwsr.height = msg.Width, msg.Height
	}
	return bwsr, cmd
}

func (bwsr Browser) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Left, bwsr.renderTable(), bwsr.renderSeparator(), bwsr.renderDetails())
}

func (bwsr Browser) renderSeparator() string {
	return lipgloss.NewStyle().
		Height(bwsr.height).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(styles.Colors.Muted).
		MarginRight(1).
		MarginLeft(1).
		Render("")
}

func (bwsr Browser) renderDetails() string {
	// only render if wide and has files
	if bwsr.width < 80 || len(bwsr.files) == 0 {
		return ""
	}

	details := strings.Builder{}

	file := bwsr.files[bwsr.index]

	httpAddr := bwsr.cfg.HTTPAddressForFile(file.ID)
	visibility := "public"
	if file.Private {
		httpAddr = styles.C(styles.Colors.Muted, "<none> (requires a signed URL)")
		visibility = styles.C(styles.Colors.Red, "private")
	}

	values := [][2]string{
		{"ID", file.ID},
		{"Size", humanize.Bytes(file.Size)},
		{"Created", fmt.Sprintf("%s (%s)", file.CreatedAt.Format(time.RFC3339), humanize.Time(file.CreatedAt))},
		{"Modified", fmt.Sprintf("%s (%s)", file.CreatedAt.Format(time.RFC3339), humanize.Time(file.UpdatedAt))},
		{"Type", strings.ToLower(file.Type)},
		{"Visibility", visibility},
	}

	access := [][2]string{
		{"URL", httpAddr},
		{"SSH", bwsr.cfg.SSHCommandForFile(file.ID)},
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Blue).
		Bold(true)

	for _, v := range values {
		details.WriteString(fmt.Sprintf("%s  %s\n", keyStyle.Width(10).Render(v[0]), v[1]))
	}

	details.WriteRune('\n')

	for _, v := range access {
		details.WriteString(fmt.Sprintf("%s  %s\n", keyStyle.Width(3).Foreground(styles.Colors.Purple).Render(v[0]), v[1]))
	}

	return lipgloss.NewStyle().
		PaddingTop(1).
		Render(details.String())
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
		header = append(header, fmt.Sprintf("%-*s", bwsr.preRenderedWidths[i], col.name))
	}

	return lipgloss.NewStyle().PaddingTop(1).Bold(true).Render("  " + strings.Join(header, "  "))
}

func (bwsr Browser) renderFooter() string {
	if len(bwsr.files) == 0 {
		return ""
	}

	return lipgloss.
		NewStyle().
		Foreground(styles.Colors.Primary).
		Render(fmt.Sprintf("\n %d/%d files", bwsr.index+1, len(bwsr.files)))
}

func (bwsr Browser) renderRows() string {
	numRows := bwsr.numRowsToRender()
	rows := make([]string, 0, numRows)
	for i := bwsr.offset; i < numRows+bwsr.offset; i++ {
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
	preRenderedRow := bwsr.preRendered[i]
	row := make([]string, 0, len(preRenderedRow))
	for i, col := range preRenderedRow {
		row = append(row, fmt.Sprintf("%-*s", bwsr.preRenderedWidths[i], col))
	}

	rowStyle := lipgloss.NewStyle()

	color := styles.Colors.Muted
	prefix := "  "
	if i == bwsr.index {
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
	bwsr.preRendered = make([][]string, len(bwsr.files))
	bwsr.preRenderedWidths = make([]int, len(columns))

	for i := range columns {
		bwsr.preRenderedWidths[i] = lipgloss.Width(columns[i].name)
	}

	for i, file := range bwsr.files {
		row := make([]string, len(columns))
		for j, col := range columns {
			row[j] = col.render(file)
			bwsr.preRenderedWidths[j] = max(bwsr.preRenderedWidths[j], lipgloss.Width(row[j]))
		}

		bwsr.preRendered[i] = row
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

package browser

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/robherley/snips.sh/internal/tui/views"
	"github.com/robherley/snips.sh/internal/tui/views/prompt"
)

const Selector = "→ "

type option struct {
	name   string
	prompt prompt.Kind
	danger bool
}

var options = []option{
	{
		name:   "edit extension",
		prompt: prompt.ChangeExtension,
	},
	{
		name:   "generate signed url",
		prompt: prompt.GenerateSignedURL,
	},
	{
		name:   "toggle visibility",
		prompt: prompt.ChangeVisibility,
	},
	{
		name:   "delete file",
		prompt: prompt.DeleteFile,
		danger: true,
	},
}

func (bwsr Browser) handleOptionsNavigation(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	opts := bwsr.getOptions()
	numOpts := len(opts)

	switch msg.String() {
	case "down", "j":
		if bwsr.options.index < numOpts-1 {
			bwsr.options.index++
		}
	case "up", "k":
		if bwsr.options.index > 0 {
			bwsr.options.index--
		}
	}

	return bwsr, nil
}

func (bwsr Browser) handleOptionsEnter() (tea.Model, tea.Cmd) {
	opts := bwsr.getOptions()
	if len(opts) == 0 {
		return bwsr, nil
	}

	file := bwsr.selectedFile()
	if file == nil {
		return bwsr, nil
	}

	selected := opts[bwsr.options.index]
	return bwsr, tea.Batch(
		cmds.SelectFile(file.ID),
		prompt.SetPromptKindCmd(selected.prompt),
		cmds.PushView(views.Prompt),
	)
}

func (bwsr Browser) getOptions() []option {
	file := bwsr.selectedFile()
	if file == nil {
		return nil
	}

	var opts []option
	for _, o := range options {
		if file.IsBinary() && o.prompt == prompt.ChangeExtension {
			// don't allow changing extension for binary files
			continue
		}

		if !file.Private && o.prompt == prompt.GenerateSignedURL {
			// don't allow generating signed urls for public files
			continue
		}

		opts = append(opts, o)
	}

	return opts
}

func (bwsr Browser) renderModal() string {
	body := styles.ModalBody(bwsr.theme, "options",
		bwsr.renderDetails(),
		bwsr.renderSelector(),
	)

	return styles.Modal(bwsr.width, bwsr.height, body)
}

func (bwsr Browser) renderSelector() string {
	selector := strings.Builder{}

	for i, o := range bwsr.getOptions() {
		color := styles.Colors.Muted
		prefix := "  "
		if i == bwsr.options.index && bwsr.options.focused {
			prefix = Selector
			color = styles.Colors.White
			if o.danger {
				color = styles.Colors.Red
			}
		}

		selector.WriteString(styles.C(bwsr.theme, prefix))
		selector.WriteString(styles.C(color, o.name))
		selector.WriteRune('\n')
	}

	return selector.String()
}

func (bwsr Browser) renderDetails() string {
	file := bwsr.selectedFile()
	if file == nil {
		return ""
	}

	rawHTTPAddr := bwsr.cfg.HTTPAddressForFile(file.ID)
	httpAddr := lipgloss.NewStyle().Hyperlink(rawHTTPAddr).Render(rawHTTPAddr)
	visibility := "public"
	if file.Private {
		httpAddr = styles.C(styles.Colors.Muted, "<none> (requires a signed URL)")
		visibility = styles.C(styles.Colors.Red, "private")
	}

	values := [][2]string{
		{"id", file.ID},
		{"size", humanize.Bytes(file.Size)},
		{"created", fmt.Sprintf("%s (%s)", file.CreatedAt.Format(time.RFC3339), humanize.Time(file.CreatedAt))},
		{"modified", fmt.Sprintf("%s (%s)", file.UpdatedAt.Format(time.RFC3339), humanize.Time(file.UpdatedAt))},
		{"type", strings.ToLower(file.Type)},
		{"visibility", visibility},
	}

	access := [][2]string{
		{"url", httpAddr},
		{"ssh", bwsr.cfg.SSHCommandForFile(file.ID)},
	}

	return renderTable(values, access) + "\n"
}

func renderTable(sections ...[][2]string) string {
	labelWidth, valueWidth := 0, 0
	for _, rows := range sections {
		for _, row := range rows {
			labelWidth = max(labelWidth, lipgloss.Width(row[0]))
			valueWidth = max(valueWidth, lipgloss.Width(row[1]))
		}
	}

	border := lipgloss.NewStyle().Foreground(styles.Colors.Muted)
	valueCell := lipgloss.NewStyle().Width(valueWidth)
	labelStyles := []lipgloss.Style{
		lipgloss.NewStyle().Foreground(styles.Colors.Blue).Bold(true).Width(labelWidth),
		lipgloss.NewStyle().Foreground(styles.Colors.Purple).Bold(true).Width(labelWidth),
	}

	renderBorder := func(left, mid, right string) string {
		return border.Render(left + strings.Repeat("─", labelWidth+2) + mid + strings.Repeat("─", valueWidth+2) + right)
	}

	lines := []string{renderBorder("╭", "┬", "╮")}
	for i, rows := range sections {
		if i > 0 {
			lines = append(lines, renderBorder("├", "┼", "┤"))
		}
		labelStyle := labelStyles[min(i, len(labelStyles)-1)]
		for _, row := range rows {
			lines = append(lines,
				border.Render("│ ")+labelStyle.Render(row[0])+
					border.Render(" │ ")+valueCell.Render(row[1])+border.Render(" │"))
		}
	}
	lines = append(lines, renderBorder("╰", "┴", "╯"))

	return strings.Join(lines, "\n")
}

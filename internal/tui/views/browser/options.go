package browser

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/styles"
	"github.com/robherley/snips.sh/internal/tui/views"
	"github.com/robherley/snips.sh/internal/tui/views/prompt"
)

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

func (bwsr Browser) handleOptionsNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	opts := bwsr.getOptions()
	numOpts := len(opts)

	switch msg.String() {
	// move down
	case "down", "j":
		if bwsr.options.index < numOpts-1 {
			bwsr.options.index++
		}
	// move up
	case "up", "k":
		if bwsr.options.index > 0 {
			bwsr.options.index--
		}
	case "enter":
		if numOpts == 0 {
			return bwsr, nil
		}

		selected := opts[bwsr.options.index]

		return bwsr, tea.Batch(
			cmds.SelectFile(bwsr.files[bwsr.table.index].ID),
			prompt.SetPromptKindCmd(selected.prompt),
			cmds.PushView(views.Prompt),
		)
	}

	return bwsr, nil
}

func (bwser Browser) getOptions() []option {
	if len(bwser.files) == 0 {
		return nil
	}

	file := bwser.files[bwser.table.index]

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

func (bwsr Browser) renderOptions() string {
	return lipgloss.JoinVertical(lipgloss.Top, bwsr.renderDetails(), bwsr.renderSelector())
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

		selector.WriteString(styles.C(styles.Colors.Yellow, prefix))
		selector.WriteString(styles.C(color, o.name))
		selector.WriteRune('\n')
	}

	return lipgloss.JoinVertical(lipgloss.Top, styles.BC(styles.Colors.Yellow, "Options"), selector.String())
}

func (bwsr Browser) renderDetails() string {
	// only render has files
	if len(bwsr.files) == 0 {
		return ""
	}

	details := strings.Builder{}

	file := bwsr.files[bwsr.table.index]

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

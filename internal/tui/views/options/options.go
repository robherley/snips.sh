package options

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/cmds"
	"github.com/robherley/snips.sh/internal/tui/msgs"
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
		name:   "rename file",
		prompt: prompt.Rename,
	},
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

// available filters the option list to the ones that apply to the file.
func available(file *snips.File) []option {
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

// Options is the modal window of file details and actions for the selected
// file, floating above the browser.
type Options struct {
	cfg    *config.Config
	width  int
	height int
	theme  color.Color

	file  *snips.File
	index int
}

func New(cfg *config.Config, width, height int, theme color.Color) Options {
	return Options{
		cfg:    cfg,
		width:  width,
		height: height,
		theme:  theme,
	}
}

func (o Options) Init() tea.Cmd {
	return nil
}

func (o Options) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		opts := available(o.file)

		switch msg.String() {
		case "down", "j":
			if o.index < len(opts)-1 {
				o.index++
			}
		case "up", "k":
			if o.index > 0 {
				o.index--
			}
		case "tab":
			return o, cmds.PopView()
		case "enter":
			if len(opts) == 0 {
				return o, nil
			}
			return o, tea.Batch(
				prompt.SetPromptKindCmd(opts[o.index].prompt, "options"),
				cmds.PushView(views.Prompt),
			)
		}
	case msgs.FileLoaded:
		o.file = msg.File
		o.index = 0
	case msgs.FileDeselected:
		o.file = nil
		o.index = 0
	case tea.WindowSizeMsg:
		o.width, o.height = msg.Width, msg.Height
	case msgs.ThemeChanged:
		o.theme = msg.Color
	}

	return o, nil
}

func (o Options) View() tea.View {
	if o.file == nil {
		return tea.NewView(lipgloss.Place(o.width, o.height, lipgloss.Left, lipgloss.Top, ""))
	}

	// bare modal window: the TUI floats it over the view beneath in the stack
	return tea.NewView(styles.ModalBody(o.theme, "options",
		o.renderDetails(),
		o.renderSelector(),
	))
}

func (o Options) Keys() help.KeyMap {
	return keys
}

func (o Options) IsCapturing() bool {
	return false
}

func (o Options) renderSelector() string {
	selector := strings.Builder{}

	for i, opt := range available(o.file) {
		color := styles.Colors.Muted
		prefix := "  "
		if i == o.index {
			prefix = Selector
			color = styles.Colors.White
			if opt.danger {
				color = styles.Colors.Red
			}
		}

		selector.WriteString(styles.C(o.theme, prefix))
		selector.WriteString(styles.C(color, opt.name))
		selector.WriteRune('\n')
	}

	return strings.TrimSuffix(selector.String(), "\n")
}

func (o Options) renderDetails() string {
	file := o.file

	rawHTTPAddr := o.cfg.HTTPAddressForFile(file.ID)
	if file.Name != "" {
		rawHTTPAddr = o.cfg.HTTPAddressForNamedFile(file.ID, file.Name)
	}
	httpAddr := lipgloss.NewStyle().Hyperlink(rawHTTPAddr).Render(rawHTTPAddr)
	visibility := "public"
	if file.Private {
		httpAddr = styles.C(styles.Colors.Muted, "<none> (requires a signed URL)")
		visibility = styles.C(styles.Colors.Red, "private")
	}

	name := styles.C(styles.Colors.Muted, "<none>")
	if file.Name != "" {
		name = file.Name
	}

	values := [][2]string{
		{"id", file.ID},
		{"name", name},
		{"size", humanize.Bytes(file.Size)},
		{"created", fmt.Sprintf("%s (%s)", file.CreatedAt.Format(time.RFC3339), humanize.Time(file.CreatedAt))},
		{"modified", fmt.Sprintf("%s (%s)", file.UpdatedAt.Format(time.RFC3339), humanize.Time(file.UpdatedAt))},
		{"type", strings.ToLower(file.Type)},
		{"visibility", visibility},
	}

	access := [][2]string{
		{"url", httpAddr},
		{"ssh", o.cfg.SSHCommandForFile(file.ID)},
	}
	if file.Name != "" {
		access = append(access, [2]string{"ssh (name)", o.cfg.SSHCommandForNamedFile(file.Name)})
	}

	return styles.Table(
		styles.TableSection{Label: styles.Colors.Blue, Rows: values},
		styles.TableSection{Label: styles.Colors.Purple, Rows: access},
	) + "\n"
}

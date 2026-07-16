package browser

import (
	"fmt"
	"math"
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
	body := lipgloss.JoinVertical(lipgloss.Top,
		bwsr.renderDetails(),
		bwsr.renderSelector(),
	)

	window := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.Colors.Muted).
		Padding(0, 2).
		Render(body)

	x := max((bwsr.width-lipgloss.Width(window))/2, 0)
	y := max((bwsr.height-lipgloss.Height(window))/2, 0)

	// layers must go through a Compositor: composing a Layer directly onto
	// the Canvas ignores its X/Y offset and blanks the cells around it
	return lipgloss.NewCanvas(bwsr.width, bwsr.height).
		Compose(lipgloss.NewCompositor(
			lipgloss.NewLayer(bwsr.renderBackdrop()).Z(0),
			lipgloss.NewLayer(window).X(x).Y(y).Z(1),
		)).
		Render()
}

// bayer is a 4x4 ordered-dither threshold matrix.
var bayer = [4][4]float64{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

// brailleBits maps a dot position within a braille cell's 2x4 grid to its bit
// in the Unicode braille pattern block (U+2800-U+28FF).
var brailleBits = [4][2]rune{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// renderBackdrop fills the view with a braille-dot texture that thickens
// toward the edges, ordered-dithered at the braille sub-dot resolution (2x4
// dots per cell) so the density ramps smoothly. Each dot is a pure function
// of its coordinates, so the pattern is stable across renders.
func (bwsr Browser) renderBackdrop() string {
	if bwsr.width <= 0 || bwsr.height <= 0 {
		return ""
	}

	// braille sub-dots are roughly square, so no cell aspect correction
	cx, cy := float64(bwsr.width*2)/2, float64(bwsr.height*4)/2
	maxDist := math.Hypot(cx, cy)

	dots := strings.Builder{}
	for y := 0; y < bwsr.height; y++ {
		if y > 0 {
			dots.WriteByte('\n')
		}
		for x := 0; x < bwsr.width; x++ {
			cell := rune(0x2800)
			for dy := range 4 {
				for dx := range 2 {
					px, py := x*2+dx, y*4+dy
					dist := math.Hypot(float64(px)-cx, float64(py)-cy) / maxDist
					// cap density so the edges stay speckled rather than solid
					if dist*0.75 > (bayer[py%4][px%4]+0.5)/16 {
						cell |= brailleBits[dy][dx]
					}
				}
			}
			dots.WriteRune(cell)
		}
	}

	return lipgloss.NewStyle().Foreground(styles.Colors.Muted).Render(dots.String())
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
	file := bwsr.selectedFile()
	if file == nil {
		return ""
	}

	details := strings.Builder{}

	rawHTTPAddr := bwsr.cfg.HTTPAddressForFile(file.ID)
	httpAddr := lipgloss.NewStyle().Hyperlink(rawHTTPAddr).Render(rawHTTPAddr)
	visibility := "public"
	if file.Private {
		httpAddr = styles.C(styles.Colors.Muted, "<none> (requires a signed URL)")
		visibility = styles.C(styles.Colors.Red, "private")
	}

	values := [][2]string{
		{"ID", file.ID},
		{"Size", humanize.Bytes(file.Size)},
		{"Created", fmt.Sprintf("%s (%s)", file.CreatedAt.Format(time.RFC3339), humanize.Time(file.CreatedAt))},
		{"Modified", fmt.Sprintf("%s (%s)", file.UpdatedAt.Format(time.RFC3339), humanize.Time(file.UpdatedAt))},
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
		fmt.Fprintf(&details, "%s  %s\n", keyStyle.Width(10).Render(v[0]), v[1])
	}

	details.WriteRune('\n')

	for _, v := range access {
		fmt.Fprintf(&details, "%s  %s\n", keyStyle.Width(3).Foreground(styles.Colors.Purple).Render(v[0]), v[1])
	}

	return lipgloss.NewStyle().
		PaddingTop(1).
		Render(details.String())
}

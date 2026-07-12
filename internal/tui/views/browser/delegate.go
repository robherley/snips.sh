package browser

import (
	"fmt"
	"image/color"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/tui/styles"
)

const ellipsis = "…"

type fileItem struct {
	file *snips.File
}

func (i fileItem) Title() string {
	return i.file.ID
}

func (i fileItem) Description() string {
	visibility := "public"
	if i.file.Private {
		visibility = "private"
	}
	return strings.Join([]string{
		strings.ToLower(i.file.Type),
		humanize.Bytes(i.file.Size),
		humanize.Time(i.file.UpdatedAt),
		visibility,
	}, " · ")
}

func (i fileItem) FilterValue() string {
	return i.file.ID + " " + strings.ToLower(i.file.Type)
}

func toItems(files []*snips.File) []list.Item {
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = fileItem{file: f}
	}
	return items
}

type fileDelegate struct {
	styles list.DefaultItemStyles
	theme  color.Color
}

func newItemDelegate(theme color.Color) fileDelegate {
	s := list.NewDefaultItemStyles(true)

	s.NormalTitle = s.NormalTitle.Foreground(styles.Colors.Muted)
	s.NormalDesc = s.NormalDesc.Foreground(styles.Colors.Muted)

	s.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(theme).
		Foreground(theme).
		Padding(0, 0, 0, 1).
		Bold(true)
	s.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(theme).
		Foreground(styles.Colors.White).
		Padding(0, 0, 0, 1)

	s.DimmedTitle = s.DimmedTitle.Foreground(styles.Colors.Muted)
	s.DimmedDesc = s.DimmedDesc.Foreground(styles.Colors.Muted)

	s.FilterMatch = lipgloss.NewStyle().Underline(true)

	return fileDelegate{styles: s, theme: theme}
}

func (d fileDelegate) Height() int                             { return 2 }
func (d fileDelegate) Spacing() int                            { return 1 }
func (d fileDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// matchHighlight is the per-rune style applied to filter matches in the title.
// Theme color + bold + underline regardless of the row's selected state.
func (d fileDelegate) matchHighlight() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(d.theme).Bold(true).Underline(true)
}

// titleMatchIdx filters fuzzy-match positions to those that fall inside the
// title (file ID) portion of FilterValue.
func titleMatchIdx(matched []int, id string) []int {
	out := make([]int, 0, len(matched))
	for _, i := range matched {
		if i < len(id) {
			out = append(out, i)
		}
	}
	return out
}

// descTypeMatchIdx returns fuzzy-match positions translated to the type span at
// the start of the description string.
func descTypeMatchIdx(matched []int, id, typ string) []int {
	typeStart := len(id) + 1 // skip the space separator in FilterValue
	typeLen := len(strings.ToLower(typ))
	out := make([]int, 0, len(matched))
	for _, i := range matched {
		if i >= typeStart && i < typeStart+typeLen {
			out = append(out, i-typeStart)
		}
	}
	return out
}

// itemHint is the right-aligned shortcut text shown on the highlighted item.
func (d fileDelegate) itemHint(file *snips.File) string {
	hints := [][2]string{
		{"[tab]", "options"},
		{"[↵]", "view"},
	}
	if file.Private {
		hints = append(hints, [2]string{"[s]", "sign"})
	}
	hints = append(hints, [2]string{"[x]", "delete"})

	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = styles.BC(d.theme, h[0]) + " " + styles.C(styles.Colors.Muted, h[1])
	}
	return strings.Join(parts, styles.C(styles.Colors.Muted, "  "))
}

func (d fileDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	file, ok := item.(fileItem)
	if !ok {
		return
	}

	title := file.Title()
	desc := file.Description()

	width := m.Width()
	if width <= 0 {
		return
	}
	// account for the left border + padding the styles add (1 col each)
	contentWidth := width - 2

	isSelected := index == m.Index()
	isFiltering := m.FilterState() == list.Filtering
	isFiltered := isFiltering || m.FilterState() == list.FilterApplied
	emptyFilter := isFiltering && m.FilterValue() == ""

	var matchedRunes []int
	if isFiltered {
		matchedRunes = m.MatchesForItem(index)
	}

	switch {
	case emptyFilter:
		title = ansi.Truncate(title, contentWidth, ellipsis)
		desc = ansi.Truncate(desc, contentWidth, ellipsis)
		title = d.styles.DimmedTitle.Render(title)
		desc = d.styles.DimmedDesc.Render(desc)

	case isSelected && !isFiltering:
		title = ansi.Truncate(title, contentWidth, ellipsis)
		if isFiltered {
			unmatched := d.styles.SelectedTitle.Inline(true)
			title = lipgloss.StyleRunes(title, titleMatchIdx(matchedRunes, file.file.ID), d.matchHighlight(), unmatched)
		}
		hint := d.itemHint(file.file)
		gap := contentWidth - lipgloss.Width(title) - lipgloss.Width(hint)
		if gap >= 1 {
			title = title + strings.Repeat(" ", gap) + hint
		}
		if isFiltered {
			descIdx := descTypeMatchIdx(matchedRunes, file.file.ID, file.file.Type)
			if len(descIdx) > 0 {
				unmatched := d.styles.SelectedDesc.Inline(true)
				desc = lipgloss.StyleRunes(desc, descIdx, d.matchHighlight(), unmatched)
			}
		}
		// emphasize "private" in red on the highlighted item — safe to embed at
		// the tail of the description because nothing renders after it
		if file.file.Private {
			desc = strings.Replace(desc, "private", styles.C(styles.Colors.Red, "private"), 1)
		}
		desc = ansi.Truncate(desc, contentWidth, ellipsis)
		title = d.styles.SelectedTitle.Render(title)
		desc = d.styles.SelectedDesc.Render(desc)

	default:
		title = ansi.Truncate(title, contentWidth, ellipsis)
		if isFiltered {
			unmatched := d.styles.NormalTitle.Inline(true)
			title = lipgloss.StyleRunes(title, titleMatchIdx(matchedRunes, file.file.ID), d.matchHighlight(), unmatched)
		}
		if isFiltered {
			descIdx := descTypeMatchIdx(matchedRunes, file.file.ID, file.file.Type)
			if len(descIdx) > 0 {
				unmatched := d.styles.NormalDesc.Inline(true)
				desc = lipgloss.StyleRunes(desc, descIdx, d.matchHighlight(), unmatched)
			}
		}
		desc = ansi.Truncate(desc, contentWidth, ellipsis)
		title = d.styles.NormalTitle.Render(title)
		desc = d.styles.NormalDesc.Render(desc)
	}

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

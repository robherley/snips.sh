package styles

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestModalBodyWidths(t *testing.T) {
	cases := map[string]struct {
		title string
		rows  []string
	}{
		"short title": {
			title: "options",
			rows:  []string{"a row"},
		},
		"title wider than rows": {
			title: strings.Repeat("t", ModalMinWidth+20),
			rows:  []string{"a row"},
		},
		"rows wider than title": {
			title: "options",
			rows:  []string{strings.Repeat("r", ModalMinWidth+20)},
		},
		"rows narrower than min width": {
			title: "settings > theme color",
			rows:  []string{"→ blue", "  red"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			body := ModalBody(Colors.Blue, tc.title, tc.rows...)

			lines := strings.Split(body, "\n")
			want := lipgloss.Width(lines[0])
			if want < ModalMinWidth {
				t.Errorf("window is %d cells wide, want at least %d", want, ModalMinWidth)
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got != want {
					t.Errorf("line %d is %d cells wide, want %d", i, got, want)
				}
				if plain := ansi.Strip(line); strings.TrimRight(plain, " ") != plain {
					t.Errorf("line %d ends in padding spaces instead of a border: %q", i, plain)
				}
			}

			if !strings.Contains(body, tc.title) {
				t.Errorf("title %q missing from rendered window", tc.title)
			}
		})
	}
}

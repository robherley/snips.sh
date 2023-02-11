package filelist

import (
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

type ListItem struct {
	ID        string
	Size      uint64
	CreatedAt time.Time
	Private   bool
	Type      string
}

func (li ListItem) Title() string {
	title := li.ID
	if li.Private {
		title += " 🔒"
	}

	return title
}

func (li ListItem) Description() string {
	attr := []string{
		strings.ToLower(li.Type),
		humanize.Bytes(li.Size),
		humanize.Time(li.CreatedAt),
	}

	return strings.Join(attr, " • ")
}

func (li ListItem) FilterValue() string {
	return li.ID + " " + li.Type
}

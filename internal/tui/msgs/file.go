package msgs

import "github.com/robherley/snips.sh/internal/snips"

type FileSelected struct {
	ID string
}

type FileDeselected struct{}

type FileLoaded struct {
	File *snips.File
}

type ReloadFiles struct {
	Files []*snips.File
}

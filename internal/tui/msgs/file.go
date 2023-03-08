package msgs

import "github.com/robherley/snips.sh/internal/db"

type FileSelected struct {
	ID string
}

type FileDeselected struct{}

type FileLoaded struct {
	File *db.File
}

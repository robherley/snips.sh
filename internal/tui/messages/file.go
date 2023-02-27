package messages

import "github.com/robherley/snips.sh/internal/db"

type FileSelected struct {
	ID string
}

type FileLoaded struct {
	File *db.File
}

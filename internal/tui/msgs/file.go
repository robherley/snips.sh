package msgs

import "github.com/robherley/snips.sh/internal/db/models"

type FileSelected struct {
	ID string
}

type FileDeselected struct{}

type FileLoaded struct {
	File *models.File
}

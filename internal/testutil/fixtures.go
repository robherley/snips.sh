package testutil

import (
	"time"

	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

type fixtures struct{}

var Fixtures = &fixtures{}

func (f *fixtures) File() snips.File {
	file := snips.File{
		ID:        id.New(),
		CreatedAt: time.Now().Add(-5 * time.Minute),
		UpdatedAt: time.Now().Add(-5 * time.Minute),
		Size:      100,
		Private:   false,
		UserID:    id.New(),
	}

	file.SetContent([]byte("hello world"), true)

	return file
}

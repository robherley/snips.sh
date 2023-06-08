package testutil

import (
	"time"

	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

type fixtures struct{}

var Fixtures = &fixtures{}

func (f *fixtures) File() snips.File {
	return snips.File{
		ID:        id.New(),
		CreatedAt: time.Now().Add(-5 * time.Minute),
		UpdatedAt: time.Now().Add(-5 * time.Minute),
		Size:      100,
		Content:   []byte("hello world"), // TODO: update with main
		Private:   false,
		UserID:    id.New(),
	}
}

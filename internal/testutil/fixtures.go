package testutil

import (
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

type fixtures struct{}

var Fixtures = &fixtures{}

func (f *fixtures) File(t *testing.T) snips.File {
	file := snips.File{
		ID:        id.New(),
		CreatedAt: time.Now().Add(-5 * time.Minute),
		UpdatedAt: time.Now().Add(-5 * time.Minute),
		Size:      100,
		Private:   false,
		UserID:    id.New(),
	}

	err := file.SetContent([]byte("hello world"), true)
	if err != nil {
		t.Fatal(err)
	}

	return file
}

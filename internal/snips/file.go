package snips

import (
	"errors"
	"time"
)

const (
	FileLimit        = 100
	FileTypeBinary   = "binary"
	FileTypeMarkdown = "markdown"
)

var (
	ErrFileLimit = errors.New("file limit reached")
)

type File struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Size      uint64
	Content   []byte
	Private   bool
	Type      string
	UserID    string
}

func (f *File) IsBinary() bool {
	return f.Type == FileTypeBinary
}

func (f *File) IsMarkdown() bool {
	return f.Type == FileTypeMarkdown
}

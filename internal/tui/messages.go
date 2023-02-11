package tui

import "github.com/robherley/snips.sh/internal/db"

type ErrorMsg struct {
	err error
}

func (em ErrorMsg) Error() string {
	return em.err.Error()
}

type FilesMsg struct {
	Files []db.File
}

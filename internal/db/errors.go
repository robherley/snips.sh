package db

import "errors"

var (
	ErrFileLimit = errors.New("file limit reached")
	ErrNameTaken = errors.New("file already exists with that name")
)

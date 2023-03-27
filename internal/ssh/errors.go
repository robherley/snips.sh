package ssh

import "errors"

var (
	ErrFileNotFound      = errors.New("file not found")
	ErrFileTooLarge      = errors.New("file too large")
	ErrNilProgram        = errors.New("nil program")
	ErrPrivateFileAccess = errors.New("private file access")
	ErrUnknownCommand    = errors.New("unknown command")
	ErrSignPublicFile    = errors.New("unable to sign public file")
	ErrOpOnNonOwnedFile  = errors.New("operation on non-owned file")
)

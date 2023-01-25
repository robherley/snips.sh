package db

import (
	"time"
)

type File struct {
	Model

	Size      uint64
	Content   []byte
	Private   bool `gorm:"index:file_private"`
	Extension *string
	ExpiresAt *time.Time

	UserID string
	User   User
}

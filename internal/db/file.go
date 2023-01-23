package db

import (
	"time"

	"github.com/segmentio/ksuid"
)

type File struct {
	Model

	Size      int64
	Content   []byte
	Private   bool `gorm:"index:file_private"`
	Extension *string
	ExpiresAt *time.Time

	UserID ksuid.KSUID
	User   User
}

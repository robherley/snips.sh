package db

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

const MaxFiles = 100

var (
	ErrUploadLimit = errors.New("upload limit reached")
)

type File struct {
	Model

	Size    uint64
	Content []byte
	Private bool `gorm:"index:file_private"`
	Type    string

	UserID string
	User   User
}

func (f *File) BeforeCreate(tx *gorm.DB) error {
	if err := f.Model.BeforeCreate(tx); err != nil {
		return err
	}

	var count int64
	if err := tx.Model(&File{}).Where("user_id = ?", f.UserID).Count(&count).Error; err != nil {
		return err
	}

	if count >= MaxFiles {
		return fmt.Errorf("%w: %d files allowed per user", ErrUploadLimit, MaxFiles)
	}

	return nil
}

package models

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

const (
	MaxFiles     = 100
	BinaryFile   = "binary"
	MarkdownFile = "markdown"
)

var (
	ErrFileLimit = errors.New("file limit reached")
)

type File struct {
	Base

	Size    uint64
	Content []byte
	Private bool `gorm:"index:file_private"`
	Type    string

	UserID string
	User   User
}

func (f *File) BeforeCreate(tx *gorm.DB) error {
	if err := f.Base.BeforeCreate(tx); err != nil {
		return err
	}

	var count int64
	if err := tx.Model(&File{}).Where("user_id = ?", f.UserID).Count(&count).Error; err != nil {
		return err
	}

	if count >= MaxFiles {
		return fmt.Errorf("%w: %d files allowed per user", ErrFileLimit, MaxFiles)
	}

	return nil
}

func (f *File) IsBinary() bool {
	return f.Type == BinaryFile
}

func (f *File) IsMarkdown() bool {
	return f.Type == MarkdownFile
}

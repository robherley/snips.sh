package models

import (
	"time"

	"github.com/robherley/snips.sh/internal/id"
	"gorm.io/gorm"
)

var All = []interface{}{
	&User{},
	&PublicKey{},
	&File{},
}

type Base struct {
	ID        string    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (b *Base) BeforeCreate(tx *gorm.DB) error {
	tx.Statement.SetColumn("ID", id.New())
	return nil
}

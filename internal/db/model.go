package db

import (
	"time"

	"github.com/robherley/snips.sh/internal/id"
	"gorm.io/gorm"
)

var AllModels = []interface{}{
	&User{},
	&PublicKey{},
	&File{},
}

type Model struct {
	gorm.Model

	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (m *Model) BeforeCreate(tx *gorm.DB) error {
	id, err := id.Generate()
	if err != nil {
		return err
	}

	tx.Statement.SetColumn("ID", id)
	return nil
}

package db

import (
	"github.com/segmentio/ksuid"
	"gorm.io/gorm"
)

var AllModels = []interface{}{
	&User{},
	&PublicKey{},
}

type Model struct {
	gorm.Model

	ID ksuid.KSUID `gorm:"primaryKey"`
}

func (m *Model) BeforeCreate(tx *gorm.DB) error {
	tx.Statement.SetColumn("ID", ksuid.New())
	return nil
}

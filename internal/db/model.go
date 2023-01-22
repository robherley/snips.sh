package db

import (
	"time"

	"github.com/segmentio/ksuid"
	"gorm.io/gorm"
)

var AllModels = []interface{}{
	&User{},
	&PublicKey{},
	&File{},
}

type Model struct {
	ID ksuid.KSUID `gorm:"primaryKey"`
}

func (m *Model) BeforeCreate(tx *gorm.DB) error {
	tx.Statement.SetColumn("ID", ksuid.New())
	return nil
}

func (m *Model) CreatedAt() time.Time {
	return m.ID.Time()
}

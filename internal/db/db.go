package db

import (
	"github.com/robherley/snips.sh/internal/config"
	zl "github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	*gorm.DB
}

func New(cfg *config.Config) (*DB, error) {
	gormdb, err := gorm.Open(sqlite.Open(cfg.DB.FilePath), &gorm.Config{
		Logger: &logger{zl.Logger},
	})
	if err != nil {
		return nil, err
	}

	return &DB{gormdb}, nil
}

package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/robherley/snips.sh/internal/config"
)

func New(cfg *config.Config) (*sql.DB, error) {
	return sql.Open("sqlite3", cfg.DB.FilePath)
}

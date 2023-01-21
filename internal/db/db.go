package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/robherley/snips.sh/internal/config"
)

type DB struct {
	*sql.DB
}

func (db *DB) Version() (string, error) {
	var version string
	if err := db.QueryRow("SELECT SQLITE_VERSION()").Scan(&version); err != nil {
		return "", err
	}

	return version, nil
}

func New(cfg *config.Config) (*DB, error) {
	db, err := sql.Open("sqlite3", cfg.DB.FilePath)
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

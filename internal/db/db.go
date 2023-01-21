package db

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/segmentio/ksuid"
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

func (db *DB) FindUserIDByFingerprint(ctx context.Context, fingerprint string) (string, error) {
	row := db.QueryRowContext(ctx, "SELECT user_id FROM public_keys WHERE fingerprint = ?", fingerprint)
	var uid string
	if err := row.Scan(&uid); err != nil {
		return "", err
	}
	return uid, nil
}

func (db *DB) CreateNewUser(ctx context.Context, fingerprint, ktype string) (string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}

	id := ksuid.New()

	_, err = tx.ExecContext(ctx, "INSERT INTO users (id) VALUES (?)", id)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO public_keys (fingerprint, type, user_id) VALUES (?, ?, ?)", fingerprint, ktype, id)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return id.String(), nil
}

func New(cfg *config.Config) (*DB, error) {
	db, err := sql.Open("sqlite3", cfg.DB.FilePath)
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

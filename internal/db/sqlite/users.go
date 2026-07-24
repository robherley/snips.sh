package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

type users struct{ *sql.DB }

func (s *users) CreateWithPublicKey(ctx context.Context, publicKey *snips.PublicKey) (*snips.User, error) {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	user := &snips.User{
		ID:        id.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	const userQuery = `
		INSERT INTO users (id, created_at, updated_at)
		VALUES (?, ?, ?)
	`
	if _, err := tx.ExecContext(ctx, userQuery, user.ID, user.CreatedAt, user.UpdatedAt); err != nil {
		return nil, err
	}

	publicKey.ID = id.New()
	publicKey.CreatedAt = time.Now().UTC()
	publicKey.UpdatedAt = time.Now().UTC()
	publicKey.UserID = user.ID
	const publicKeyQuery = `
		INSERT INTO public_keys (id, created_at, updated_at, fingerprint, type, user_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	if _, err := tx.ExecContext(ctx, publicKeyQuery,
		publicKey.ID,
		publicKey.CreatedAt,
		publicKey.UpdatedAt,
		publicKey.Fingerprint,
		publicKey.Type,
		publicKey.UserID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *users) Find(ctx context.Context, id string) (*snips.User, error) {
	const query = `
		SELECT id, created_at, updated_at, theme_color
		FROM users
		WHERE id = ?
	`

	user := &snips.User{}
	if err := s.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.ThemeColor,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return user, nil
}

func (s *users) Update(ctx context.Context, user *snips.User) error {
	const query = `
		UPDATE users
		SET updated_at = ?, theme_color = ?
		WHERE id = ?
	`

	updatedAt := time.Now().UTC()
	if _, err := s.ExecContext(ctx, query, updatedAt, user.ThemeColor, user.ID); err != nil {
		return err
	}

	user.UpdatedAt = updatedAt
	return nil
}

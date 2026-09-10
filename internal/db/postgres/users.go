package postgres

import (
	"context"
	"database/sql"
	"errors"

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

	now := nowUTC()
	user := &snips.User{ID: id.New(), CreatedAt: now, UpdatedAt: now}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO users (display_id, created_at, updated_at) VALUES ($1, $2, $3)`,
		user.ID, user.CreatedAt, user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	publicKey.ID = id.New()
	publicKey.CreatedAt = now
	publicKey.UpdatedAt = now
	publicKey.UserID = user.ID
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO public_keys (display_id, created_at, updated_at, fingerprint, type, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		publicKey.ID, publicKey.CreatedAt, publicKey.UpdatedAt,
		publicKey.Fingerprint, publicKey.Type, publicKey.UserID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *users) Find(ctx context.Context, userID string) (*snips.User, error) {
	user := &snips.User{}
	err := s.QueryRowContext(ctx, `
		SELECT display_id, created_at, updated_at, theme_color FROM users WHERE display_id = $1`, userID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.ThemeColor)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()
	return user, nil
}

func (s *users) Update(ctx context.Context, user *snips.User) error {
	updatedAt := nowUTC()
	if _, err := s.ExecContext(ctx,
		`UPDATE users SET updated_at = $1, theme_color = $2 WHERE display_id = $3`,
		updatedAt, user.ThemeColor, user.ID,
	); err != nil {
		return err
	}
	user.UpdatedAt = updatedAt
	return nil
}

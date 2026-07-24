package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

type apiKeys struct{ *sql.DB }

func (s *apiKeys) Create(ctx context.Context, key *snips.APIKey, maxKeys uint64) error {
	keyID := id.New()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	query := `
		INSERT INTO api_keys (
			id, created_at, updated_at, name, token_hash, user_id, last_used_at, expires_at
		) SELECT ?, ?, ?, ?, ?, ?, NULL, ?
	`
	args := []any{
		keyID,
		createdAt,
		updatedAt,
		nullableName(key.Name),
		key.TokenHash,
		key.UserID,
		key.ExpiresAt,
	}
	if maxKeys > 0 {
		query += `WHERE (SELECT COUNT(*) FROM api_keys WHERE user_id = ?) < ?`
		args = append(args, key.UserID, maxKeys)
	}

	result, err := s.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	created, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if created == 0 {
		return db.ErrAPIKeyLimit
	}

	key.ID = keyID
	key.CreatedAt = createdAt
	key.UpdatedAt = updatedAt
	return nil
}

func (s *apiKeys) FindByTokenHash(ctx context.Context, tokenHash string) (*snips.APIKey, error) {
	const query = `
		SELECT id, created_at, updated_at, name, token_hash, user_id, last_used_at, expires_at
		FROM api_keys
		WHERE token_hash = ?
	`

	key, err := scanAPIKey(s.QueryRowContext(ctx, query, tokenHash).Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return key, nil
}

func (s *apiKeys) FindByUser(ctx context.Context, userID string) ([]*snips.APIKey, error) {
	const query = `
		SELECT id, created_at, updated_at, name, token_hash, user_id, last_used_at, expires_at
		FROM api_keys
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC
	`

	rows, err := s.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []*snips.APIKey{}
	for rows.Next() {
		key, err := scanAPIKey(rows.Scan)
		if err != nil {
			return nil, err
		}

		keys = append(keys, key)
	}

	return keys, rows.Err()
}

func scanAPIKey(scan func(dest ...any) error) (*snips.APIKey, error) {
	key := &snips.APIKey{}
	name := sql.NullString{}
	lastUsedAt := sql.NullTime{}
	expiresAt := sql.NullTime{}

	if err := scan(
		&key.ID,
		&key.CreatedAt,
		&key.UpdatedAt,
		&name,
		&key.TokenHash,
		&key.UserID,
		&lastUsedAt,
		&expiresAt,
	); err != nil {
		return nil, err
	}

	key.Name = name.String
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return key, nil
}

func (s *apiKeys) Delete(ctx context.Context, id, userID string) (bool, error) {
	const query = `DELETE FROM api_keys WHERE id = ? AND user_id = ?`

	result, err := s.ExecContext(ctx, query, id, userID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (s *apiKeys) Touch(ctx context.Context, id string) error {
	const query = `UPDATE api_keys SET last_used_at = ? WHERE id = ?`

	_, err := s.ExecContext(ctx, query, time.Now().UTC(), id)
	return err
}

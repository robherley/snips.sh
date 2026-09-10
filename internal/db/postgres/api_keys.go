package postgres

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
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if maxKeys > 0 {
		var count uint64
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys WHERE user_id = $1`, key.UserID).Scan(&count); err != nil {
			return err
		}
		if count >= maxKeys {
			return db.ErrAPIKeyLimit
		}
	}

	now := nowUTC()
	keyID := id.New()
	if key.ExpiresAt != nil {
		expiresAt := key.ExpiresAt.UTC().Truncate(time.Microsecond)
		key.ExpiresAt = &expiresAt
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO api_keys
			(display_id, created_at, updated_at, name, token_hash, user_id, last_used_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULL, $7)`,
		keyID, now, now, nullableName(key.Name), key.TokenHash, key.UserID, key.ExpiresAt,
	)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	key.ID, key.CreatedAt, key.UpdatedAt = keyID, now, now
	return nil
}

func scanAPIKey(scan func(...any) error) (*snips.APIKey, error) {
	key := &snips.APIKey{}
	var name sql.NullString
	var lastUsedAt, expiresAt sql.NullTime
	if err := scan(&key.ID, &key.CreatedAt, &key.UpdatedAt, &name, &key.TokenHash,
		&key.UserID, &lastUsedAt, &expiresAt); err != nil {
		return nil, err
	}
	key.CreatedAt = key.CreatedAt.UTC()
	key.UpdatedAt = key.UpdatedAt.UTC()
	key.Name = name.String
	if lastUsedAt.Valid {
		lastUsed := lastUsedAt.Time.UTC()
		key.LastUsedAt = &lastUsed
	}
	if expiresAt.Valid {
		expires := expiresAt.Time.UTC()
		key.ExpiresAt = &expires
	}
	return key, nil
}

func (s *apiKeys) FindByTokenHash(ctx context.Context, tokenHash string) (*snips.APIKey, error) {
	key, err := scanAPIKey(s.QueryRowContext(ctx, `
		SELECT display_id, created_at, updated_at, name, token_hash, user_id, last_used_at, expires_at
		FROM api_keys WHERE token_hash = $1`, tokenHash).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return key, err
}

func (s *apiKeys) FindByUser(ctx context.Context, userID string) ([]*snips.APIKey, error) {
	rows, err := s.QueryContext(ctx, `
		SELECT display_id, created_at, updated_at, name, token_hash, user_id, last_used_at, expires_at
		FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC, display_id DESC`, userID)
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

func (s *apiKeys) Delete(ctx context.Context, keyID, userID string) (bool, error) {
	result, err := s.ExecContext(ctx, `DELETE FROM api_keys WHERE display_id = $1 AND user_id = $2`, keyID, userID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

func (s *apiKeys) Touch(ctx context.Context, keyID string) error {
	_, err := s.ExecContext(ctx, `UPDATE api_keys SET last_used_at = $1 WHERE display_id = $2`, nowUTC(), keyID)
	return err
}

package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/robherley/snips.sh/internal/snips"
)

type publicKeys struct{ *sql.DB }

func (s *publicKeys) FindByFingerprint(ctx context.Context, fingerprint string) (*snips.PublicKey, error) {
	key := &snips.PublicKey{}
	err := s.QueryRowContext(ctx, `
		SELECT display_id, created_at, updated_at, fingerprint, type, user_id
		FROM public_keys WHERE fingerprint = $1`, fingerprint,
	).Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt, &key.Fingerprint, &key.Type, &key.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	key.CreatedAt = key.CreatedAt.UTC()
	key.UpdatedAt = key.UpdatedAt.UTC()
	return key, nil
}

package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/robherley/snips.sh/internal/snips"
)

type publicKeys struct{ *sql.DB }

func (s *publicKeys) FindByFingerprint(ctx context.Context, fingerprint string) (*snips.PublicKey, error) {
	const query = `
		SELECT id, created_at, updated_at, fingerprint, type, user_id
		FROM public_keys
		WHERE fingerprint = ?
	`

	key := &snips.PublicKey{}
	if err := s.QueryRowContext(ctx, query, fingerprint).Scan(
		&key.ID,
		&key.CreatedAt,
		&key.UpdatedAt,
		&key.Fingerprint,
		&key.Type,
		&key.UserID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return key, nil
}

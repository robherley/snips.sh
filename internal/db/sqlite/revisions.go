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

type revisions struct {
	*sql.DB
	compress bool
}

func (s *revisions) Create(ctx context.Context, revision *snips.Revision, diff []byte, maxRevisions uint64) error {
	storedDiff, err := snips.EncodeContent(diff, s.compress)
	if err != nil {
		return err
	}

	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	const nextSequenceQuery = `
		SELECT COALESCE(MAX(sequence), 0) + 1
		FROM revisions
		WHERE file_id = ?
	`
	if err := tx.QueryRowContext(ctx, nextSequenceQuery, revision.FileID).Scan(&revision.Sequence); err != nil {
		return err
	}

	revision.ID = id.New()
	revision.CreatedAt = time.Now().UTC()
	const insertQuery = `
		INSERT INTO revisions (id, sequence, file_id, created_at, diff, size, type)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	if _, err := tx.ExecContext(ctx, insertQuery,
		revision.ID,
		revision.Sequence,
		revision.FileID,
		revision.CreatedAt,
		storedDiff,
		revision.Size,
		revision.Type,
	); err != nil {
		return err
	}

	if maxRevisions > 0 {
		const pruneQuery = `
			DELETE FROM revisions
			WHERE file_id = ? AND id NOT IN (
				SELECT id FROM revisions
				WHERE file_id = ?
				ORDER BY sequence DESC
				LIMIT ?
			)
		`
		if _, err := tx.ExecContext(ctx, pruneQuery, revision.FileID, revision.FileID, maxRevisions); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *revisions) FindByFileID(ctx context.Context, fileID string, opts ...db.PageOption) ([]*snips.Revision, error) {
	query := `
		SELECT id, sequence, file_id, created_at, size, type
		FROM revisions
		WHERE file_id = ?
		ORDER BY sequence DESC`
	args := applyPage(&query, []any{fileID}, opts)

	return s.query(ctx, query, args...)
}

func (s *revisions) query(ctx context.Context, query string, args ...any) ([]*snips.Revision, error) {
	rows, err := s.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	revisions := []*snips.Revision{}
	for rows.Next() {
		revision := &snips.Revision{}
		if err := rows.Scan(
			&revision.ID,
			&revision.Sequence,
			&revision.FileID,
			&revision.CreatedAt,
			&revision.Size,
			&revision.Type,
		); err != nil {
			return nil, err
		}

		revisions = append(revisions, revision)
	}

	return revisions, rows.Err()
}

func (s *revisions) FindByFileIDAndSequence(ctx context.Context, fileID string, sequence int64) (*snips.Revision, error) {
	const query = `
		SELECT id, sequence, file_id, created_at, size, type
		FROM revisions
		WHERE file_id = ? AND sequence = ?
	`

	revision := &snips.Revision{}
	if err := s.QueryRowContext(ctx, query, fileID, sequence).Scan(
		&revision.ID,
		&revision.Sequence,
		&revision.FileID,
		&revision.CreatedAt,
		&revision.Size,
		&revision.Type,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return revision, nil
}

func (s *revisions) FindDiff(ctx context.Context, id string) ([]byte, error) {
	const query = `SELECT diff FROM revisions WHERE id = ?`

	var diff []byte
	if err := s.QueryRowContext(ctx, query, id).Scan(&diff); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return snips.DecodeContent(diff)
}

func (s *revisions) CountByFileID(ctx context.Context, fileID string) (int64, error) {
	const query = `SELECT COUNT(*) FROM revisions WHERE file_id = ?`

	var count int64
	if err := s.QueryRowContext(ctx, query, fileID).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

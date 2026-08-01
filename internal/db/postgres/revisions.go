package postgres

import (
	"context"
	"database/sql"
	"errors"

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

	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(sequence), 0) + 1 FROM revisions WHERE file_id = $1`,
		revision.FileID,
	).Scan(&revision.Sequence); err != nil {
		return err
	}

	revision.ID = id.New()
	revision.CreatedAt = nowUTC()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO revisions (display_id, sequence, file_id, created_at, diff, size, type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, revision.ID, revision.Sequence,
		revision.FileID, revision.CreatedAt, storedDiff, revision.Size, revision.Type,
	); err != nil {
		return err
	}

	if maxRevisions > 0 {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM revisions WHERE file_id = $1 AND id NOT IN (
				SELECT id FROM revisions WHERE file_id = $1
				ORDER BY sequence DESC LIMIT $2
			)`, revision.FileID, maxRevisions); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scanRevision(row scanner) (*snips.Revision, error) {
	revision := &snips.Revision{}
	if err := row.Scan(&revision.ID, &revision.Sequence, &revision.FileID,
		&revision.CreatedAt, &revision.Size, &revision.Type); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	revision.CreatedAt = revision.CreatedAt.UTC()
	return revision, nil
}

func (s *revisions) FindByFileID(ctx context.Context, fileID string, opts ...db.PageOption) ([]*snips.Revision, error) {
	page := db.ResolvePage(opts...)
	query := `
		SELECT r.display_id, r.sequence, r.file_id, r.created_at, r.size, r.type
		FROM revisions AS r WHERE r.file_id = $1`
	args := []any{fileID}
	if page.Cursor.ID != "" {
		query += ` AND r.id < (
			SELECT cursor.id FROM revisions AS cursor
			WHERE cursor.display_id = $2 AND cursor.file_id = $1
		)`
		args = append(args, page.Cursor.ID)
	}
	query += ` ORDER BY r.id DESC`
	args = applyLimit(&query, args, page)
	rows, err := s.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*snips.Revision{}
	for rows.Next() {
		revision, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, revision)
	}
	return result, rows.Err()
}

func (s *revisions) FindByFileIDAndSequence(ctx context.Context, fileID string, sequence int64) (*snips.Revision, error) {
	return scanRevision(s.QueryRowContext(ctx, `
		SELECT display_id, sequence, file_id, created_at, size, type
		FROM revisions WHERE file_id = $1 AND sequence = $2`, fileID, sequence))
}

func (s *revisions) FindDiff(ctx context.Context, revisionID string) ([]byte, error) {
	var diff []byte
	err := s.QueryRowContext(ctx, `SELECT diff FROM revisions WHERE display_id = $1`, revisionID).Scan(&diff)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return snips.DecodeContent(diff)
}

func (s *revisions) CountByFileID(ctx context.Context, fileID string) (int64, error) {
	var count int64
	err := s.QueryRowContext(ctx, `SELECT COUNT(*) FROM revisions WHERE file_id = $1`, fileID).Scan(&count)
	return count, err
}

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

type files struct {
	*sql.DB
	compress bool
}

func (s *files) Find(ctx context.Context, id string) (*snips.File, error) {
	const query = `
		SELECT id, created_at, updated_at, size, private, type, user_id, name
		FROM files
		WHERE id = ?
	`

	return scanFile(s.QueryRowContext(ctx, query, id))
}

func (s *files) FindWithContent(ctx context.Context, id string) (*snips.File, []byte, error) {
	const query = `
		SELECT id, created_at, updated_at, size, content, private, type, user_id, name
		FROM files
		WHERE id = ?
	`

	file := &snips.File{}
	name := sql.NullString{}
	var content []byte

	if err := s.QueryRowContext(ctx, query, id).Scan(
		&file.ID,
		&file.CreatedAt,
		&file.UpdatedAt,
		&file.Size,
		&content,
		&file.Private,
		&file.Type,
		&file.UserID,
		&name,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}

		return nil, nil, err
	}

	file.Name = name.String
	decoded, err := snips.DecodeContent(content)
	if err != nil {
		return nil, nil, err
	}

	return file, decoded, nil
}

func scanFile(row *sql.Row) (*snips.File, error) {
	file := &snips.File{}
	name := sql.NullString{}

	if err := row.Scan(
		&file.ID,
		&file.CreatedAt,
		&file.UpdatedAt,
		&file.Size,
		&file.Private,
		&file.Type,
		&file.UserID,
		&name,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	file.Name = name.String
	return file, nil
}

func nameConstraintErr(err error) error {
	sqliteErr := sqlite3.Error{}
	if errors.As(err, &sqliteErr) &&
		sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique &&
		strings.Contains(sqliteErr.Error(), "files.name") {
		return db.ErrNameTaken
	}

	return err
}

func (s *files) Create(ctx context.Context, file *snips.File, content []byte, maxFileCount uint64) error {
	const countQuery = `SELECT COUNT(*) FROM files WHERE user_id = ?`

	var count uint64
	if err := s.QueryRowContext(ctx, countQuery, file.UserID).Scan(&count); err != nil {
		return err
	}
	if maxFileCount > 0 && count >= maxFileCount {
		return db.ErrFileLimit
	}
	storedContent, err := snips.EncodeContent(content, s.compress)
	if err != nil {
		return err
	}

	file.ID = id.New()
	file.CreatedAt = time.Now().UTC()
	file.UpdatedAt = time.Now().UTC()
	file.Size = uint64(len(content))

	const insertQuery = `
		INSERT INTO files (
			id, created_at, updated_at, size, content, private, type, user_id, name
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	if _, err := s.ExecContext(ctx, insertQuery,
		file.ID,
		file.CreatedAt,
		file.UpdatedAt,
		file.Size,
		storedContent,
		file.Private,
		file.Type,
		file.UserID,
		nullableName(file.Name),
	); err != nil {
		return nameConstraintErr(err)
	}

	return nil
}

func (s *files) Update(ctx context.Context, file *snips.File) error {
	file.UpdatedAt = time.Now().UTC()

	const query = `
		UPDATE files
		SET updated_at = ?, size = ?, private = ?, type = ?, name = ?
		WHERE id = ?
	`

	if _, err := s.ExecContext(ctx, query,
		file.UpdatedAt,
		file.Size,
		file.Private,
		file.Type,
		nullableName(file.Name),
		file.ID,
	); err != nil {
		return nameConstraintErr(err)
	}

	return nil
}

func (s *files) FindContent(ctx context.Context, id string) ([]byte, error) {
	const query = `SELECT content FROM files WHERE id = ?`

	var content []byte
	if err := s.QueryRowContext(ctx, query, id).Scan(&content); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return snips.DecodeContent(content)
}

func (s *files) UpdateContent(ctx context.Context, file *snips.File, content []byte) error {
	storedContent, err := snips.EncodeContent(content, s.compress)
	if err != nil {
		return err
	}

	file.UpdatedAt = time.Now().UTC()
	file.Size = uint64(len(content))
	const query = `
		UPDATE files
		SET updated_at = ?, size = ?, content = ?, private = ?, type = ?, name = ?
		WHERE id = ?
	`
	if _, err := s.ExecContext(ctx, query,
		file.UpdatedAt,
		file.Size,
		storedContent,
		file.Private,
		file.Type,
		nullableName(file.Name),
		file.ID,
	); err != nil {
		return nameConstraintErr(err)
	}

	return nil
}

func (s *files) FindByUser(ctx context.Context, userID string, opts ...db.PageOption) ([]*snips.File, error) {
	query := `
		SELECT id, created_at, updated_at, size, private, type, user_id, name
		FROM files
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC`
	args := applyPage(&query, []any{userID}, opts)

	return s.query(ctx, query, args...)
}

func (s *files) query(ctx context.Context, query string, args ...any) ([]*snips.File, error) {
	rows, err := s.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []*snips.File{}
	for rows.Next() {
		file := &snips.File{}
		name := sql.NullString{}
		if err := rows.Scan(
			&file.ID,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.Size,
			&file.Private,
			&file.Type,
			&file.UserID,
			&name,
		); err != nil {
			return nil, err
		}

		file.Name = name.String
		files = append(files, file)
	}

	return files, rows.Err()
}

func (s *files) FindByName(ctx context.Context, userID, name string) (*snips.File, error) {
	const query = `
		SELECT id, created_at, updated_at, size, private, type, user_id, name
		FROM files
		WHERE user_id = ? AND name = ? COLLATE NOCASE
	`

	return scanFile(s.QueryRowContext(ctx, query, userID, name))
}

func (s *files) CountByUser(ctx context.Context, userID string) (int64, error) {
	const query = `SELECT COUNT(*) FROM files WHERE user_id = ?`

	var count int64
	if err := s.QueryRowContext(ctx, query, userID).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *files) Delete(ctx context.Context, id string) error {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM revisions WHERE file_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM files WHERE id = ?`, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *files) DeleteByUser(ctx context.Context, userID string) (int64, error) {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	const deleteRevisionsQuery = `
		DELETE FROM revisions
		WHERE file_id IN (SELECT id FROM files WHERE user_id = ?)
	`
	if _, err := tx.ExecContext(ctx, deleteRevisionsQuery, userID); err != nil {
		return 0, err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM files WHERE user_id = ?`, userID)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return count, tx.Commit()
}

package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

type files struct {
	*sql.DB
	compress bool
}

type scanner interface{ Scan(...any) error }

func scanFile(row scanner) (*snips.File, error) {
	file := &snips.File{}
	var name sql.NullString
	if err := row.Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt, &file.Size,
		&file.Private, &file.Type, &file.UserID, &name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	file.Name = name.String
	file.CreatedAt = file.CreatedAt.UTC()
	file.UpdatedAt = file.UpdatedAt.UTC()
	return file, nil
}

func (s *files) Find(ctx context.Context, fileID string) (*snips.File, error) {
	return scanFile(s.QueryRowContext(ctx, `
		SELECT display_id, created_at, updated_at, size, private, type, user_id, name
		FROM files WHERE display_id = $1`, fileID))
}

func (s *files) FindWithContent(ctx context.Context, fileID string) (*snips.File, []byte, error) {
	file := &snips.File{}
	var name sql.NullString
	var content []byte
	err := s.QueryRowContext(ctx, `
		SELECT display_id, created_at, updated_at, size, content, private, type, user_id, name
		FROM files WHERE display_id = $1`, fileID,
	).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt, &file.Size, &content,
		&file.Private, &file.Type, &file.UserID, &name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	file.Name = name.String
	file.CreatedAt = file.CreatedAt.UTC()
	file.UpdatedAt = file.UpdatedAt.UTC()
	decoded, err := snips.DecodeContent(content)
	if err != nil {
		return nil, nil, err
	}
	return file, decoded, nil
}

func nameConstraintErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_files_user_id_name" {
		return db.ErrNameTaken
	}
	return err
}

func (s *files) Create(ctx context.Context, file *snips.File, content []byte, maxFiles uint64) error {
	storedContent, err := snips.EncodeContent(content, s.compress)
	if err != nil {
		return err
	}
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if maxFiles > 0 {
		var count uint64
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM files WHERE user_id = $1`, file.UserID).Scan(&count); err != nil {
			return err
		}
		if count >= maxFiles {
			return db.ErrFileLimit
		}
	}

	now := nowUTC()
	fileID := id.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO files
			(display_id, created_at, updated_at, size, content, private, type, user_id, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		fileID, now, now, len(content), storedContent, file.Private, file.Type,
		file.UserID, nullableName(file.Name),
	)
	if err != nil {
		return nameConstraintErr(err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	file.ID, file.CreatedAt, file.UpdatedAt, file.Size = fileID, now, now, uint64(len(content))
	return nil
}

func (s *files) Update(ctx context.Context, file *snips.File) error {
	updatedAt := nowUTC()
	_, err := s.ExecContext(ctx, `
		UPDATE files SET updated_at = $1, size = $2, private = $3, type = $4, name = $5
		WHERE display_id = $6`, updatedAt, file.Size, file.Private, file.Type,
		nullableName(file.Name), file.ID)
	if err != nil {
		return nameConstraintErr(err)
	}
	file.UpdatedAt = updatedAt
	return nil
}

func (s *files) FindContent(ctx context.Context, fileID string) ([]byte, error) {
	var content []byte
	err := s.QueryRowContext(ctx, `SELECT content FROM files WHERE display_id = $1`, fileID).Scan(&content)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return snips.DecodeContent(content)
}

func (s *files) UpdateContent(ctx context.Context, file *snips.File, content []byte) error {
	storedContent, err := snips.EncodeContent(content, s.compress)
	if err != nil {
		return err
	}
	updatedAt := nowUTC()
	_, err = s.ExecContext(ctx, `
		UPDATE files
		SET updated_at = $1, size = $2, content = $3, private = $4, type = $5, name = $6
		WHERE display_id = $7`, updatedAt, len(content), storedContent, file.Private, file.Type,
		nullableName(file.Name), file.ID)
	if err != nil {
		return nameConstraintErr(err)
	}
	file.UpdatedAt, file.Size = updatedAt, uint64(len(content))
	return nil
}

func (s *files) FindByUser(ctx context.Context, userID string, opts ...db.PageOption) ([]*snips.File, error) {
	page := db.ResolvePage(opts...)
	query := `
		SELECT f.display_id, f.created_at, f.updated_at, f.size, f.private, f.type, f.user_id, f.name
		FROM files AS f WHERE f.user_id = $1`
	args := []any{userID}
	if page.Cursor.ID != "" {
		query += ` AND f.id < (
			SELECT cursor.id FROM files AS cursor
			WHERE cursor.display_id = $2 AND cursor.user_id = $1
		)`
		args = append(args, page.Cursor.ID)
	}
	query += ` ORDER BY f.id DESC`
	args = applyLimit(&query, args, page)
	return s.query(ctx, query, args...)
}

func (s *files) query(ctx context.Context, query string, args ...any) ([]*snips.File, error) {
	rows, err := s.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*snips.File{}
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, file)
	}
	return result, rows.Err()
}

func (s *files) FindByName(ctx context.Context, userID, name string) (*snips.File, error) {
	return scanFile(s.QueryRowContext(ctx, `
		SELECT display_id, created_at, updated_at, size, private, type, user_id, name
		FROM files WHERE user_id = $1 AND lower(name) = lower($2)`, userID, name))
}

func (s *files) CountByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := s.QueryRowContext(ctx, `SELECT COUNT(*) FROM files WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (s *files) Delete(ctx context.Context, fileID string) error {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM revisions WHERE file_id = $1`, fileID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM files WHERE display_id = $1`, fileID); err != nil {
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
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM revisions WHERE file_id IN (SELECT display_id FROM files WHERE user_id = $1)`, userID); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM files WHERE user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return count, tx.Commit()
}

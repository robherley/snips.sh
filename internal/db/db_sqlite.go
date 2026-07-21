package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Sqlite struct {
	*sql.DB
}

func NewSqlite(dsn string) (DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return &Sqlite{db}, nil
}

func (s *Sqlite) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	return goose.UpContext(ctx, s.DB, "migrations")
}

func (s *Sqlite) FindFile(ctx context.Context, id string) (*snips.File, error) {
	const query = `
		SELECT
			id,
			created_at,
			updated_at,
			size,
			content,
			private,
			type,
			user_id,
			name
		FROM files
		WHERE id = ?
	`

	return scanFile(s.QueryRowContext(ctx, query, id))
}

func scanFile(row *sql.Row) (*snips.File, error) {
	file := &snips.File{}
	name := sql.NullString{}

	if err := row.Scan(
		&file.ID,
		&file.CreatedAt,
		&file.UpdatedAt,
		&file.Size,
		&file.RawContent,
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

// applyPage appends the limit/offset pagination to a listing query; sqlite
// requires a LIMIT clause (-1 = unbounded) to use OFFSET.
func applyPage(query *string, args []any, opts []PageOption) []any {
	p := buildPage(opts)
	if p.limit == 0 && p.offset == 0 {
		return args
	}

	limit := int64(-1)
	if p.limit > 0 {
		limit = int64(p.limit)
	}

	*query += ` LIMIT ?`
	args = append(args, limit)

	if p.offset > 0 {
		*query += ` OFFSET ?`
		args = append(args, p.offset)
	}

	return args
}

// nullableName stores unnamed files as NULL rather than empty string.
func nullableName(name string) sql.NullString {
	return sql.NullString{String: name, Valid: name != ""}
}

// nameConstraintErr maps a violation of the per-user unique name index to
// ErrNameTaken so callers can show a friendly message.
func nameConstraintErr(err error) error {
	sqliteErr := sqlite3.Error{}
	if errors.As(err, &sqliteErr) &&
		sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique &&
		strings.Contains(sqliteErr.Error(), "files.name") {
		return ErrNameTaken
	}

	return err
}

func (s *Sqlite) CreateFile(ctx context.Context, file *snips.File, maxFileCount uint64) error {
	const countQuery = `
		SELECT COUNT(*)
		FROM files
		WHERE user_id = ?
	`

	var count uint64
	row := s.QueryRowContext(ctx, countQuery, file.UserID)
	if err := row.Scan(&count); err != nil {
		return err
	}

	if maxFileCount > 0 && count >= maxFileCount {
		return ErrFileLimit
	}

	file.ID = id.New()
	file.CreatedAt = time.Now().UTC()
	file.UpdatedAt = time.Now().UTC()

	const insertQuery = `
		INSERT INTO files (
			id,
			created_at,
			updated_at,
			size,
			content,
			private,
			type,
			user_id,
			name
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	if _, err := s.ExecContext(ctx, insertQuery,
		file.ID,
		file.CreatedAt,
		file.UpdatedAt,
		file.Size,
		file.RawContent,
		file.Private,
		file.Type,
		file.UserID,
		nullableName(file.Name),
	); err != nil {
		return nameConstraintErr(err)
	}

	return nil
}

func (s *Sqlite) UpdateFile(ctx context.Context, file *snips.File) error {
	file.UpdatedAt = time.Now().UTC()

	const query = `
		UPDATE files
		SET
			updated_at = ?,
			size = ?,
			content = ?,
			private = ?,
			type = ?,
			name = ?
		WHERE id = ?
	`

	if _, err := s.ExecContext(ctx, query,
		file.UpdatedAt,
		file.Size,
		file.RawContent,
		file.Private,
		file.Type,
		nullableName(file.Name),
		file.ID,
	); err != nil {
		return nameConstraintErr(err)
	}

	return nil
}

func (s *Sqlite) FindFilesByUser(ctx context.Context, userID string, opts ...PageOption) ([]*snips.File, error) {
	// note that content is _not_ included
	query := `
		SELECT
			id,
			created_at,
			updated_at,
			size,
			private,
			type,
			user_id,
			name
		FROM files
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC`
	args := applyPage(&query, []any{userID}, opts)

	return s.queryFilesWithoutContent(ctx, query, args...)
}

func (s *Sqlite) queryFilesWithoutContent(ctx context.Context, query string, args ...any) ([]*snips.File, error) {
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

func (s *Sqlite) FindFileByName(ctx context.Context, userID, name string) (*snips.File, error) {
	// names are unique per user, so at most one row can match
	const query = `
		SELECT
			id,
			created_at,
			updated_at,
			size,
			content,
			private,
			type,
			user_id,
			name
		FROM files
		WHERE user_id = ? AND name = ? COLLATE NOCASE
	`

	return scanFile(s.QueryRowContext(ctx, query, userID, name))
}

func (s *Sqlite) CountFilesByUser(ctx context.Context, userID string) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM files
		WHERE user_id = ?
	`

	var count int64
	if err := s.QueryRowContext(ctx, query, userID).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Sqlite) FindPublicKeyByFingerprint(ctx context.Context, fingerprint string) (*snips.PublicKey, error) {
	const query = `
		SELECT
			id,
			created_at,
			updated_at,
			fingerprint,
			type,
			user_id
		FROM public_keys
		WHERE fingerprint = ?
	`

	pk := &snips.PublicKey{}
	row := s.QueryRowContext(ctx, query, fingerprint)

	if err := row.Scan(
		&pk.ID,
		&pk.CreatedAt,
		&pk.UpdatedAt,
		&pk.Fingerprint,
		&pk.Type,
		&pk.UserID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return pk, nil
}

func (s *Sqlite) CreateUserWithPublicKey(ctx context.Context, publickey *snips.PublicKey) (*snips.User, error) {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	user := &snips.User{
		ID:        id.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	const userQuery = `
		INSERT INTO users (
			id,
			created_at,
			updated_at
		) VALUES (?, ?, ?)
	`

	if _, err := tx.ExecContext(ctx, userQuery,
		user.ID,
		user.CreatedAt,
		user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	publickey.ID = id.New()
	publickey.CreatedAt = time.Now().UTC()
	publickey.UpdatedAt = time.Now().UTC()
	publickey.UserID = user.ID

	const pkQuery = `
		INSERT INTO public_keys (
			id,
			created_at,
			updated_at,
			fingerprint,
			type,
			user_id
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	if _, err := tx.ExecContext(ctx, pkQuery,
		publickey.ID,
		publickey.CreatedAt,
		publickey.UpdatedAt,
		publickey.Fingerprint,
		publickey.Type,
		publickey.UserID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Sqlite) DeleteFile(ctx context.Context, id string) error {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const deleteRevisionsQuery = `
		DELETE FROM revisions
		WHERE file_id = ?
	`

	if _, err := tx.ExecContext(ctx, deleteRevisionsQuery, id); err != nil {
		return err
	}

	const deleteFileQuery = `
		DELETE FROM files
		WHERE id = ?
	`

	if _, err := tx.ExecContext(ctx, deleteFileQuery, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Sqlite) DeleteFilesByUser(ctx context.Context, userID string) (int64, error) {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const deleteRevisionsQuery = `
		DELETE FROM revisions
		WHERE file_id IN (SELECT id FROM files WHERE user_id = ?)
	`

	if _, err := tx.ExecContext(ctx, deleteRevisionsQuery, userID); err != nil {
		return 0, err
	}

	const deleteFilesQuery = `
		DELETE FROM files
		WHERE user_id = ?
	`

	result, err := tx.ExecContext(ctx, deleteFilesQuery, userID)
	if err != nil {
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return count, tx.Commit()
}

func (s *Sqlite) CreateRevision(ctx context.Context, revision *snips.Revision, maxRevisions uint64) error {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const nextSeqQuery = `
		SELECT COALESCE(MAX(sequence), 0) + 1
		FROM revisions
		WHERE file_id = ?
	`

	var nextSeq int64
	row := tx.QueryRowContext(ctx, nextSeqQuery, revision.FileID)
	if err := row.Scan(&nextSeq); err != nil {
		return err
	}

	revision.ID = id.New()
	revision.Sequence = nextSeq
	revision.CreatedAt = time.Now().UTC()

	const insertQuery = `
		INSERT INTO revisions (
			id,
			sequence,
			file_id,
			created_at,
			diff,
			size,
			type
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	if _, err := tx.ExecContext(ctx, insertQuery,
		revision.ID,
		revision.Sequence,
		revision.FileID,
		revision.CreatedAt,
		revision.RawDiff,
		revision.Size,
		revision.Type,
	); err != nil {
		return err
	}

	// Prune oldest revisions if over the limit
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

func (s *Sqlite) FindRevisionsByFileID(ctx context.Context, fileID string, opts ...PageOption) ([]*snips.Revision, error) {
	query := `
		SELECT
			id,
			sequence,
			file_id,
			created_at,
			size,
			type
		FROM revisions
		WHERE file_id = ?
		ORDER BY sequence DESC`
	args := applyPage(&query, []any{fileID}, opts)

	return s.queryRevisionsWithoutDiff(ctx, query, args...)
}

func (s *Sqlite) queryRevisionsWithoutDiff(ctx context.Context, query string, args ...any) ([]*snips.Revision, error) {
	rows, err := s.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	revisions := []*snips.Revision{}
	for rows.Next() {
		rev := &snips.Revision{}
		if err := rows.Scan(
			&rev.ID,
			&rev.Sequence,
			&rev.FileID,
			&rev.CreatedAt,
			&rev.Size,
			&rev.Type,
		); err != nil {
			return nil, err
		}

		revisions = append(revisions, rev)
	}

	return revisions, rows.Err()
}

func (s *Sqlite) FindRevisionByFileIDAndSequence(ctx context.Context, fileID string, sequence int64) (*snips.Revision, error) {
	const query = `
		SELECT
			id,
			sequence,
			file_id,
			created_at,
			diff,
			size,
			type
		FROM revisions
		WHERE file_id = ? AND sequence = ?
	`

	rev := &snips.Revision{}
	row := s.QueryRowContext(ctx, query, fileID, sequence)

	if err := row.Scan(
		&rev.ID,
		&rev.Sequence,
		&rev.FileID,
		&rev.CreatedAt,
		&rev.RawDiff,
		&rev.Size,
		&rev.Type,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return rev, nil
}

func (s *Sqlite) CountRevisionsByFileID(ctx context.Context, fileID string) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM revisions
		WHERE file_id = ?
	`

	var count int64
	row := s.QueryRowContext(ctx, query, fileID)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Sqlite) FindUser(ctx context.Context, id string) (*snips.User, error) {
	const query = `
		SELECT
			id,
			created_at,
			updated_at,
			theme_color
		FROM users
		WHERE id = ?
	`

	user := &snips.User{}
	row := s.QueryRowContext(ctx, query, id)

	if err := row.Scan(
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

func (s *Sqlite) CreateAPIKey(ctx context.Context, key *snips.APIKey, maxKeys uint64) error {
	const countQuery = `
		SELECT COUNT(*)
		FROM api_keys
		WHERE user_id = ?
	`

	var count uint64
	row := s.QueryRowContext(ctx, countQuery, key.UserID)
	if err := row.Scan(&count); err != nil {
		return err
	}

	if maxKeys > 0 && count >= maxKeys {
		return ErrAPIKeyLimit
	}

	key.ID = id.New()
	key.CreatedAt = time.Now().UTC()
	key.UpdatedAt = time.Now().UTC()

	const insertQuery = `
		INSERT INTO api_keys (
			id,
			created_at,
			updated_at,
			name,
			token_hash,
			user_id,
			last_used_at,
			expires_at
		) VALUES (?, ?, ?, ?, ?, ?, NULL, ?)
	`

	_, err := s.ExecContext(ctx, insertQuery,
		key.ID,
		key.CreatedAt,
		key.UpdatedAt,
		nullableName(key.Name),
		key.TokenHash,
		key.UserID,
		key.ExpiresAt,
	)

	return err
}

func (s *Sqlite) FindAPIKeyByTokenHash(ctx context.Context, tokenHash string) (*snips.APIKey, error) {
	const query = `
		SELECT
			id,
			created_at,
			updated_at,
			name,
			token_hash,
			user_id,
			last_used_at,
			expires_at
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

func (s *Sqlite) FindAPIKeysByUser(ctx context.Context, userID string) ([]*snips.APIKey, error) {
	const query = `
		SELECT
			id,
			created_at,
			updated_at,
			name,
			token_hash,
			user_id,
			last_used_at,
			expires_at
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

func (s *Sqlite) DeleteAPIKey(ctx context.Context, id, userID string) (bool, error) {
	const query = `
		DELETE FROM api_keys
		WHERE id = ? AND user_id = ?
	`

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

func (s *Sqlite) TouchAPIKey(ctx context.Context, id string) error {
	const query = `
		UPDATE api_keys
		SET last_used_at = ?
		WHERE id = ?
	`

	_, err := s.ExecContext(ctx, query, time.Now().UTC(), id)
	return err
}

func (s *Sqlite) UpdateUser(ctx context.Context, user *snips.User) error {
	const query = `
		UPDATE users
		SET
			updated_at = ?,
			theme_color = ?
		WHERE id = ?
	`

	updatedAt := time.Now().UTC()
	if _, err := s.ExecContext(ctx, query, updatedAt, user.ThemeColor, user.ID); err != nil {
		return err
	}

	user.UpdatedAt = updatedAt
	return nil
}

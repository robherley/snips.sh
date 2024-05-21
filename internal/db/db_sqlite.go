package db

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"time"

	"ariga.io/atlas/sql/migrate"
	aschema "ariga.io/atlas/sql/schema"
	asqlite "ariga.io/atlas/sql/sqlite"
	_ "github.com/mattn/go-sqlite3" // sqlite driver

	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/snips"
)

var (
	//go:embed schema.hcl
	schema []byte
)

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
	driver, err := asqlite.Open(s.DB)
	if err != nil {
		return err
	}

	want := &aschema.Schema{}
	if err := asqlite.EvalHCLBytes(schema, want, nil); err != nil {
		return err
	}

	got, err := driver.InspectSchema(ctx, "", nil)
	if err != nil {
		return err
	}

	changes, err := driver.SchemaDiff(got, want)
	if err != nil {
		return err
	}

	return driver.ApplyChanges(ctx, changes, []migrate.PlanOption{}...)
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
			user_id
		FROM files
		WHERE id = ?
	`

	file := &snips.File{}
	row := s.QueryRowContext(ctx, query, id)

	if err := row.Scan(
		&file.ID,
		&file.CreatedAt,
		&file.UpdatedAt,
		&file.Size,
		&file.RawContent,
		&file.Private,
		&file.Type,
		&file.UserID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return file, nil
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
			user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
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
	); err != nil {
		return err
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
			type = ?
		WHERE id = ?
	`

	if _, err := s.ExecContext(ctx, query,
		file.UpdatedAt,
		file.Size,
		file.RawContent,
		file.Private,
		file.Type,
		file.ID,
	); err != nil {
		return err
	}

	return nil
}

func (s *Sqlite) DeleteFile(ctx context.Context, id string) error {
	const query = `
		DELETE FROM files
		WHERE id = ?
	`

	if _, err := s.ExecContext(ctx, query, id); err != nil {
		return err
	}

	return nil
}

func (s *Sqlite) FindFilesByUser(ctx context.Context, userID string) ([]*snips.File, error) {
	// note that content is _not_ included
	const query = `
		SELECT
			id,
			created_at,
			updated_at,
			size,
			private,
			type,
			user_id
		FROM files
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`

	rows, err := s.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []*snips.File{}
	for rows.Next() {
		file := &snips.File{}
		if err := rows.Scan(
			&file.ID,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.Size,
			&file.Private,
			&file.Type,
			&file.UserID,
		); err != nil {
			return nil, err
		}

		files = append(files, file)
	}

	return files, nil
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

func (s *Sqlite) FindUser(ctx context.Context, id string) (*snips.User, error) {
	const query = `
		SELECT
			id,
			created_at,
			updated_at
		FROM users
		WHERE id = ?
	`

	user := &snips.User{}
	row := s.QueryRowContext(ctx, query, id)

	if err := row.Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return user, nil
}

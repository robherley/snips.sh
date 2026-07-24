package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"

	"github.com/pressly/goose/v3"
	"github.com/robherley/snips.sh/internal/db"
)

//go:embed migrations/*.sql
var sqliteMigrations embed.FS

type migrator struct{ *sql.DB }

func New(dsn string) (*db.DB, error) {
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return NewWithDB(database), nil
}

// NewWithDB builds a SQLite backend around an existing connection pool.
func NewWithDB(database *sql.DB) *db.DB {
	return &db.DB{
		Migrator:   &migrator{DB: database},
		Closer:     database,
		Files:      &files{DB: database},
		PublicKeys: &publicKeys{DB: database},
		Users:      &users{DB: database},
		Revisions:  &revisions{DB: database},
		APIKeys:    &apiKeys{DB: database},
	}
}

func (s *migrator) Migrate(ctx context.Context) error {
	migrations, err := fs.Sub(sqliteMigrations, "migrations")
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, s.DB, migrations)
	if err != nil {
		return err
	}

	_, err = provider.Up(ctx)
	return err
}

// applyPage appends SQLite limit/offset pagination to a listing query. SQLite
// requires a LIMIT clause (-1 means unbounded) when OFFSET is present.
func applyPage(query *string, args []any, opts []db.PageOption) []any {
	page := db.ResolvePage(opts...)
	if page.Limit == 0 && page.Offset == 0 {
		return args
	}

	limit := int64(-1)
	if page.Limit > 0 {
		limit = int64(page.Limit)
	}

	*query += ` LIMIT ?`
	args = append(args, limit)
	if page.Offset > 0 {
		*query += ` OFFSET ?`
		args = append(args, page.Offset)
	}

	return args
}

func nullableName(name string) sql.NullString {
	return sql.NullString{String: name, Valid: name != ""}
}

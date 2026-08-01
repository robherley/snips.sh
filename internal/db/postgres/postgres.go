package postgres

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/robherley/snips.sh/internal/db"
)

//go:embed migrations/*.sql
var migrations embed.FS

type migrator struct{ *sql.DB }

// New builds a PostgreSQL backend from a postgres:// or postgresql:// URL. The
// returned sql.DB is a connection pool.
func New(value string, compress bool) (*db.DB, error) {
	database, err := sql.Open("pgx", value)
	if err != nil {
		return nil, err
	}

	return NewWithDB(database, compress), nil
}

// NewWithDB builds a PostgreSQL backend around an existing connection pool.
func NewWithDB(database *sql.DB, compress bool) *db.DB {
	return &db.DB{
		Migrator:   &migrator{DB: database},
		Closer:     database,
		Files:      &files{DB: database, compress: compress},
		PublicKeys: &publicKeys{DB: database},
		Users:      &users{DB: database},
		Revisions:  &revisions{DB: database, compress: compress},
		APIKeys:    &apiKeys{DB: database},
	}
}

func (s *migrator) Migrate(ctx context.Context) error {
	migrations, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, s.DB, migrations)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}

func applyLimit(query *string, args []any, page db.Page) []any {
	if page.Limit > 0 {
		args = append(args, page.Limit)
		*query += " LIMIT $" + strconv.Itoa(len(args))
	}
	return args
}

func nullableName(name string) any {
	if name == "" {
		return nil
	}
	return name
}

func nowUTC() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

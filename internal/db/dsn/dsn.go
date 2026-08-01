package dsn

import (
	"fmt"
	"strings"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/db/postgres"
	"github.com/robherley/snips.sh/internal/db/sqlite"
)

type Driver string

const (
	SQLite   Driver = "sqlite"
	Postgres Driver = "postgres"
)

type DSN struct {
	Driver Driver
	Value  string
}

// Parse infers the database driver for value. A postgres:// or postgresql://
// prefix selects Postgres. Everything else, including an empty value, defaults
// to SQLite.
func Parse(value string) *DSN {
	parsed := &DSN{Driver: SQLite, Value: value}
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		parsed.Driver = Postgres
		return parsed
	}
	return parsed
}

// NewDB returns a new database connection for the given DSN and config.
func (d *DSN) NewDB(cfg *config.Config) (*db.DB, error) {
	switch d.Driver {
	case SQLite:
		return sqlite.New(d.Value, cfg.FileCompression)
	case Postgres:
		return postgres.New(d.Value, cfg.FileCompression)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", d.Driver)
	}
}

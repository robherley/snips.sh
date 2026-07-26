// Package dsn parses database connection strings and infers their driver.
package dsn

import (
	"errors"
	"strings"
)

var ErrPostgresProtocolRequired = errors.New("PostgreSQL DSN must begin with postgres:// or postgresql://")

// Driver identifies a supported database backend.
type Driver string

const (
	SQLite   Driver = "sqlite"
	Postgres Driver = "postgres"
)

// DSN is a parsed database connection string.
type DSN struct {
	Driver Driver
	Value  string
}

// Parse infers the database driver for value. A postgres:// or postgresql://
// prefix selects Postgres. Everything else, including an empty value, defaults
// to SQLite.
func Parse(value string) DSN {
	parsed := DSN{Driver: SQLite, Value: value}
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		parsed.Driver = Postgres
		return parsed
	}
	return parsed
}

package dsn_test

import (
	"testing"

	"github.com/robherley/snips.sh/internal/db/dsn"
)

func TestParse(t *testing.T) {
	tests := map[string]dsn.Driver{
		"":                                      dsn.SQLite,
		"data/snips.db":                         dsn.SQLite,
		"file:data/snips.db?_busy_timeout=5000": dsn.SQLite,
		":memory:":                              dsn.SQLite,
		"host=localhost port=5432 user=snips dbname=snips sslmode=disable": dsn.SQLite,
		"postgres://user:password@localhost:5432/snips":                    dsn.Postgres,
		"POSTGRESQL://user:password@localhost:5432/snips":                  dsn.Postgres,
	}

	for value, expected := range tests {
		t.Run(value, func(t *testing.T) {
			parsed := dsn.Parse(value)
			if parsed.Driver != expected {
				t.Fatalf("Parse(%q).Driver = %q, want %q", value, parsed.Driver, expected)
			}
			if parsed.Value != value {
				t.Fatalf("Parse(%q).Value = %q, want original value", value, parsed.Value)
			}
		})
	}
}

package testutil

import (
	"context"
	"database/sql"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/db/dsn"
	"github.com/robherley/snips.sh/internal/db/postgres"
	"github.com/robherley/snips.sh/internal/db/sqlite"
	"github.com/stretchr/testify/require"
)

const postgresTestURL = "postgres://snips:snips@127.0.0.1:55432/snips_test?sslmode=disable"

var postgresSchemaCounter atomic.Uint64

type Database struct {
	*db.DB
	SQL    *sql.DB
	Schema string
}

func NewDatabase(t *testing.T, driver dsn.Driver, compress bool) *Database {
	t.Helper()

	var database *Database
	switch driver {
	case dsn.SQLite:
		raw, err := sql.Open("sqlite3", t.TempDir()+"/snips.db")
		require.NoError(t, err)
		database = &Database{DB: sqlite.NewWithDB(raw, compress), SQL: raw}
	case dsn.Postgres:
		database = newPostgresDatabase(t, compress)
	default:
		t.Fatalf("unsupported database driver: %s", driver)
	}

	t.Cleanup(func() { require.NoError(t, database.Close()) })
	require.NoError(t, database.Migrate(t.Context()))
	return database
}

func newPostgresDatabase(t *testing.T, compress bool) *Database {
	t.Helper()

	parsed, err := url.Parse(postgresTestURL)
	require.NoError(t, err)
	admin, err := sql.Open("pgx", postgresTestURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	schema := "snips_test_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" +
		strconv.FormatUint(postgresSchemaCounter.Add(1), 10)
	_, err = admin.ExecContext(t.Context(), `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		require.NoError(t, err)
	})

	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	backend, err := postgres.New(parsed.String(), compress)
	require.NoError(t, err)
	return &Database{DB: backend, SQL: admin, Schema: schema}
}

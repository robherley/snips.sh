package postgres

import (
	"context"
	"database/sql"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/stretchr/testify/require"
)

const postgresTestURL = "postgres://snips:snips@127.0.0.1:55432/snips_test?sslmode=disable"

var postgresTestCounter atomic.Uint64

type testDB struct {
	*db.DB
	admin  *sql.DB
	schema string
}

func newTestDB(t *testing.T) *testDB {
	t.Helper()

	parsed, err := url.Parse(postgresTestURL)
	require.NoError(t, err)

	admin, err := sql.Open("pgx", postgresTestURL)
	require.NoError(t, err)
	schema := "snips_test_" + strconv.FormatUint(postgresTestCounter.Add(1), 10)
	_, err = admin.ExecContext(t.Context(), `CREATE SCHEMA `+schema)
	require.NoError(t, err)

	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	database, err := New(parsed.String(), true)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		_ = admin.Close()
	})

	require.NoError(t, database.Migrate(t.Context()))
	return &testDB{DB: database, admin: admin, schema: schema}
}

func (database *testDB) createTestUser(t *testing.T) *snips.User {
	t.Helper()

	publicKey := testutil.Fixtures.PublicKey(t)
	user, err := database.Users.CreateWithPublicKey(t.Context(), &publicKey)
	require.NoError(t, err)
	return user
}

func (database *testDB) createTestFile(t *testing.T, userID, name, content string) *snips.File {
	t.Helper()

	file := testutil.Fixtures.File(t)
	file.UserID = userID
	file.Type = "plaintext"
	file.Name = name
	require.NoError(t, database.Files.Create(t.Context(), &file, []byte(content), 2))
	return &file
}

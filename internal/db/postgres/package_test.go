package postgres_test

import (
	"testing"

	"github.com/robherley/snips.sh/internal/db/dsn"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/stretchr/testify/require"
)

type testDB struct {
	*testutil.Database
}

func newTestDB(t *testing.T) *testDB {
	t.Helper()
	return &testDB{Database: testutil.NewDatabase(t, dsn.Postgres, true)}
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

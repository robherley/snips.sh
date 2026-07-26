package postgres

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/db/dsn"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/stretchr/testify/require"
)

func TestApplyLimit(t *testing.T) {
	query := "SELECT * FROM files WHERE user_id = $1"
	page := db.ResolvePage(db.WithLimit(10))
	args := applyLimit(&query, []any{"user"}, page)
	require.Equal(t, "SELECT * FROM files WHERE user_id = $1 LIMIT $2", query)
	require.Equal(t, []any{"user", uint64(10)}, args)
}

func TestNewRequiresPostgresProtocol(t *testing.T) {
	for _, value := range []string{
		"host=localhost user=snips dbname=snips",
		"data/snips.db",
	} {
		t.Run(value, func(t *testing.T) {
			_, err := New(value, false)
			require.ErrorIs(t, err, dsn.ErrPostgresProtocolRequired)
		})
	}
}

func TestPostgres(t *testing.T) {
	dsn := os.Getenv("SNIPS_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("SNIPS_TEST_POSTGRES_URL is not set")
	}

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.True(t, parsed.Scheme == "postgres" || parsed.Scheme == "postgresql")

	admin, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	schema := "snips_test_" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_")) +
		"_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err = admin.ExecContext(t.Context(), `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		_ = admin.Close()
	})

	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	database, err := New(parsed.String(), true)
	require.NoError(t, err)
	defer database.Close()
	require.NoError(t, database.Migrate(t.Context()))

	for _, table := range []string{"users", "public_keys", "files", "revisions", "api_keys"} {
		var dataType, isIdentity, primaryKey string
		err := admin.QueryRowContext(t.Context(), `
			SELECT data_type, is_identity
			FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2 AND column_name = 'id'`,
			schema, table,
		).Scan(&dataType, &isIdentity)
		require.NoError(t, err)
		require.Equal(t, "bigint", dataType)
		require.Equal(t, "YES", isIdentity)

		err = admin.QueryRowContext(t.Context(), `
			SELECT key_column_usage.column_name
			FROM information_schema.table_constraints
			JOIN information_schema.key_column_usage
				USING (constraint_catalog, constraint_schema, constraint_name)
			WHERE table_constraints.table_schema = $1
				AND table_constraints.table_name = $2
				AND table_constraints.constraint_type = 'PRIMARY KEY'`,
			schema, table,
		).Scan(&primaryKey)
		require.NoError(t, err)
		require.Equal(t, "id", primaryKey)
	}

	publicKey := &snips.PublicKey{Fingerprint: "SHA256:test", Type: "ssh-ed25519"}
	user, err := database.Users.CreateWithPublicKey(t.Context(), publicKey)
	require.NoError(t, err)
	foundKey, err := database.PublicKeys.FindByFingerprint(t.Context(), publicKey.Fingerprint)
	require.NoError(t, err)
	require.Equal(t, publicKey, foundKey)

	file := &snips.File{UserID: user.ID, Type: "plaintext", Name: "Greeting"}
	require.NoError(t, database.Files.Create(t.Context(), file, []byte("hello world"), 1))
	var internalID int64
	var displayID string
	err = admin.QueryRowContext(t.Context(), `SELECT id, display_id FROM `+schema+`.files WHERE display_id = $1`, file.ID).
		Scan(&internalID, &displayID)
	require.NoError(t, err)
	require.Positive(t, internalID)
	require.Equal(t, file.ID, displayID)
	err = database.Files.Create(t.Context(), &snips.File{UserID: user.ID}, nil, 1)
	require.ErrorIs(t, err, db.ErrFileLimit)
	foundFile, content, err := database.Files.FindWithContent(t.Context(), file.ID)
	require.NoError(t, err)
	require.Equal(t, file, foundFile)
	require.Equal(t, []byte("hello world"), content)
	foundFile, err = database.Files.FindByName(t.Context(), user.ID, "gREETING")
	require.NoError(t, err)
	require.Equal(t, file, foundFile)
	err = database.Files.Create(t.Context(), &snips.File{
		UserID: user.ID, Type: "plaintext", Name: "gREETING",
	}, nil, 0)
	require.ErrorIs(t, err, db.ErrNameTaken)
	for _, name := range []string{"Second", "Third"} {
		require.NoError(t, database.Files.Create(t.Context(), &snips.File{
			UserID: user.ID, Type: "plaintext", Name: name,
		}, []byte(name), 0))
	}
	filePage, err := database.Files.FindByUser(t.Context(), user.ID, db.WithLimit(2))
	require.NoError(t, err)
	require.Len(t, filePage, 2)
	fileCursor := filePage[len(filePage)-1]
	nextFilePage, err := database.Files.FindByUser(t.Context(), user.ID,
		db.WithLimit(2),
		db.WithCursor(db.Cursor{ID: fileCursor.ID}),
	)
	require.NoError(t, err)
	require.Len(t, nextFilePage, 1)
	require.NotEqual(t, filePage[0].ID, nextFilePage[0].ID)
	require.NotEqual(t, filePage[1].ID, nextFilePage[0].ID)

	revision := &snips.Revision{FileID: file.ID, Size: 11, Type: "plaintext"}
	require.NoError(t, database.Revisions.Create(t.Context(), revision, []byte("a diff"), 1))
	diff, err := database.Revisions.FindDiff(t.Context(), revision.ID)
	require.NoError(t, err)
	require.Equal(t, []byte("a diff"), diff)
	for range 2 {
		require.NoError(t, database.Revisions.Create(t.Context(),
			&snips.Revision{FileID: file.ID, Size: 11, Type: "plaintext"},
			[]byte("another diff"), 0,
		))
	}
	revisionPage, err := database.Revisions.FindByFileID(t.Context(), file.ID, db.WithLimit(2))
	require.NoError(t, err)
	require.Len(t, revisionPage, 2)
	nextRevisionPage, err := database.Revisions.FindByFileID(t.Context(), file.ID,
		db.WithLimit(2),
		db.WithCursor(db.Cursor{ID: revisionPage[len(revisionPage)-1].ID}),
	)
	require.NoError(t, err)
	require.Len(t, nextRevisionPage, 1)
	require.Less(t, nextRevisionPage[0].Sequence, revisionPage[1].Sequence)

	apiKey := &snips.APIKey{UserID: user.ID, Name: "test", TokenHash: "hash"}
	require.NoError(t, database.APIKeys.Create(t.Context(), apiKey, 1))
	err = database.APIKeys.Create(t.Context(), &snips.APIKey{UserID: user.ID, TokenHash: "other"}, 1)
	require.ErrorIs(t, err, db.ErrAPIKeyLimit)
	foundAPIKey, err := database.APIKeys.FindByTokenHash(t.Context(), "hash")
	require.NoError(t, err)
	require.Equal(t, apiKey, foundAPIKey)
}

package postgres

import (
	"testing"

	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestUsers(t *testing.T) {
	t.Run("CreateWithPublicKey", func(t *testing.T) {
		database := newTestDB(t)
		publicKey := testutil.Fixtures.PublicKey(t)

		user, err := database.Users.CreateWithPublicKey(t.Context(), &publicKey)
		require.NoError(t, err)
		require.NotEmpty(t, user.ID)
		require.False(t, user.CreatedAt.IsZero())
		require.Equal(t, user.CreatedAt, user.UpdatedAt)
		require.NotEmpty(t, publicKey.ID)
		require.Equal(t, user.ID, publicKey.UserID)

		duplicateKey := testutil.Fixtures.PublicKey(t)
		duplicateKey.Fingerprint = publicKey.Fingerprint
		_, err = database.Users.CreateWithPublicKey(t.Context(), &duplicateKey)
		require.Error(t, err)
		var count int
		require.NoError(t, database.admin.QueryRowContext(t.Context(),
			`SELECT count(*) FROM `+database.schema+`.users`,
		).Scan(&count))
		require.Equal(t, 1, count, "the failed transaction must not leave a user behind")
	})

	t.Run("Find", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)

		foundUser, err := database.Users.Find(t.Context(), user.ID)
		require.NoError(t, err)
		require.Equal(t, user, foundUser)

		missingUser, err := database.Users.Find(t.Context(), "missing")
		require.NoError(t, err)
		require.Nil(t, missingUser)
	})

	t.Run("Update", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		originalUpdatedAt := user.UpdatedAt
		user.ThemeColor = "#abcdef"

		require.NoError(t, database.Users.Update(t.Context(), user))
		require.GreaterOrEqual(t, user.UpdatedAt, originalUpdatedAt)
		foundUser, err := database.Users.Find(t.Context(), user.ID)
		require.NoError(t, err)
		require.Equal(t, "#abcdef", foundUser.ThemeColor)
		require.Equal(t, user.UpdatedAt, foundUser.UpdatedAt)
	})
}

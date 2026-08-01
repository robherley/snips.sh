package postgres_test

import (
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestAPIKeys(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		expiresAt := time.Now().UTC().Add(time.Hour)
		key := testutil.Fixtures.APIKey(t)
		key.UserID = user.ID
		key.Name = "first"
		key.ExpiresAt = &expiresAt

		require.NoError(t, database.APIKeys.Create(t.Context(), &key, 1))
		require.NotEmpty(t, key.ID)
		require.False(t, key.CreatedAt.IsZero())
		require.Equal(t, key.CreatedAt, key.UpdatedAt)
		require.Equal(t, expiresAt.Truncate(time.Microsecond), *key.ExpiresAt)

		overLimit := testutil.Fixtures.APIKey(t)
		overLimit.UserID = user.ID
		err := database.APIKeys.Create(t.Context(), &overLimit, 1)
		require.ErrorIs(t, err, db.ErrAPIKeyLimit)
	})

	t.Run("FindByTokenHash", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		key := testutil.Fixtures.APIKey(t)
		key.UserID = user.ID
		require.NoError(t, database.APIKeys.Create(t.Context(), &key, 0))

		foundKey, err := database.APIKeys.FindByTokenHash(t.Context(), key.TokenHash)
		require.NoError(t, err)
		require.Equal(t, &key, foundKey)
		missingKey, err := database.APIKeys.FindByTokenHash(t.Context(), "missing")
		require.NoError(t, err)
		require.Nil(t, missingKey)
	})

	t.Run("FindByUser", func(t *testing.T) {
		database := newTestDB(t)
		firstUser := database.createTestUser(t)
		secondPublicKey := testutil.Fixtures.PublicKey(t)
		secondUser, err := database.Users.CreateWithPublicKey(t.Context(), &secondPublicKey)
		require.NoError(t, err)
		firstKey := testutil.Fixtures.APIKey(t)
		firstKey.UserID = firstUser.ID
		secondKey := testutil.Fixtures.APIKey(t)
		secondKey.UserID = firstUser.ID
		otherKey := testutil.Fixtures.APIKey(t)
		otherKey.UserID = secondUser.ID
		require.NoError(t, database.APIKeys.Create(t.Context(), &firstKey, 0))
		require.NoError(t, database.APIKeys.Create(t.Context(), &secondKey, 0))
		require.NoError(t, database.APIKeys.Create(t.Context(), &otherKey, 0))

		keys, err := database.APIKeys.FindByUser(t.Context(), firstUser.ID)
		require.NoError(t, err)
		require.Equal(t, []*snips.APIKey{&secondKey, &firstKey}, keys)
		keys, err = database.APIKeys.FindByUser(t.Context(), "missing")
		require.NoError(t, err)
		require.Empty(t, keys)
	})

	t.Run("Touch", func(t *testing.T) {
		database := newTestDB(t)
		user := database.createTestUser(t)
		key := testutil.Fixtures.APIKey(t)
		key.UserID = user.ID
		require.NoError(t, database.APIKeys.Create(t.Context(), &key, 0))

		require.NoError(t, database.APIKeys.Touch(t.Context(), key.ID))
		foundKey, err := database.APIKeys.FindByTokenHash(t.Context(), key.TokenHash)
		require.NoError(t, err)
		require.NotNil(t, foundKey.LastUsedAt)
		require.NoError(t, database.APIKeys.Touch(t.Context(), "missing"))
	})

	t.Run("Delete", func(t *testing.T) {
		database := newTestDB(t)
		owner := database.createTestUser(t)
		otherPublicKey := testutil.Fixtures.PublicKey(t)
		otherUser, err := database.Users.CreateWithPublicKey(t.Context(), &otherPublicKey)
		require.NoError(t, err)
		key := testutil.Fixtures.APIKey(t)
		key.UserID = owner.ID
		require.NoError(t, database.APIKeys.Create(t.Context(), &key, 0))

		deleted, err := database.APIKeys.Delete(t.Context(), key.ID, otherUser.ID)
		require.NoError(t, err)
		require.False(t, deleted)
		deleted, err = database.APIKeys.Delete(t.Context(), key.ID, owner.ID)
		require.NoError(t, err)
		require.True(t, deleted)
		deleted, err = database.APIKeys.Delete(t.Context(), key.ID, owner.ID)
		require.NoError(t, err)
		require.False(t, deleted)
	})
}

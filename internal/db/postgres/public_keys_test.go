package postgres

import (
	"testing"

	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestPublicKeys(t *testing.T) {
	t.Run("FindByFingerprint", func(t *testing.T) {
		database := newTestDB(t)
		publicKey := testutil.Fixtures.PublicKey(t)
		_, err := database.Users.CreateWithPublicKey(t.Context(), &publicKey)
		require.NoError(t, err)

		foundPublicKey, err := database.PublicKeys.FindByFingerprint(t.Context(), publicKey.Fingerprint)
		require.NoError(t, err)
		require.Equal(t, &publicKey, foundPublicKey)

		missingPublicKey, err := database.PublicKeys.FindByFingerprint(t.Context(), "SHA256:missing")
		require.NoError(t, err)
		require.Nil(t, missingPublicKey)
	})
}

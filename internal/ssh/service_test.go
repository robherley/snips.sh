package ssh_test

import (
	"net/url"
	"path/filepath"
	"testing"

	dbmock "github.com/robherley/snips.sh/internal/db/mock"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh/testdata"
)

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()

	internal, err := url.Parse("ssh://localhost:0")
	require.NoError(t, err)

	cfg := &config.Config{}
	cfg.SSH.Internal = *internal

	return cfg
}

func TestNew_HostKey(t *testing.T) {
	t.Run("uses inline PEM host key when set", func(t *testing.T) {
		cfg := newTestConfig(t)
		cfg.SSH.HostKey = string(testdata.PEMBytes["ed25519"])
		cfg.SSH.HostKeyPath = filepath.Join(t.TempDir(), "does-not-exist")

		database := dbmock.NewDB(t)

		service, err := ssh.New(cfg, database.DB)
		require.NoError(t, err)

		require.Len(t, service.HostSigners, 1)
		assert.Equal(t, "ssh-ed25519", service.HostSigners[0].PublicKey().Type())
	})

	t.Run("errors on invalid inline PEM host key", func(t *testing.T) {
		cfg := newTestConfig(t)
		cfg.SSH.HostKey = "not-a-valid-pem-key"

		database := dbmock.NewDB(t)

		_, err := ssh.New(cfg, database.DB)
		assert.Error(t, err)
	})

	t.Run("falls back to host key path when inline key is unset", func(t *testing.T) {
		cfg := newTestConfig(t)
		cfg.SSH.HostKeyPath = filepath.Join(t.TempDir(), "snips")

		database := dbmock.NewDB(t)

		service, err := ssh.New(cfg, database.DB)
		require.NoError(t, err)

		require.Len(t, service.HostSigners, 1)
	})
}

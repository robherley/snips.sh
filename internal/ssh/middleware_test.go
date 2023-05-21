package ssh_test

import (
	"net/url"
	"testing"
	"time"

	cssh "github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/id"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/testdata"
)

var (
	testTimeout   = 5 * time.Second
	testHost, _   = url.Parse("http://localhost:8080")
	privateKey    = testdata.PEMBytes["ed25519"]
	publicKey     = []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAID7d/uFLuDlRbBc4ZVOsx+GbHKuOrPtLHFvHsjWPwO+/")
	fingerprint   = "SHA256:mV1mPX4S6TE+odyfWDXGrC5fvQbLh+w8o2NK3q2MmYw"
	authorizedKey = func() cssh.PublicKey {
		authorizedKey, _, _, _, _ := cssh.ParseAuthorizedKey(publicKey)
		return authorizedKey
	}()
)

func testPrivateKeyAuth(key []byte) gossh.AuthMethod {
	signer, err := gossh.ParsePrivateKey(key)
	if err != nil {
		panic(err)
	}

	return gossh.PublicKeys(signer)
}

func TestAssignUser(t *testing.T) {
	t.Run("creates new user", func(t *testing.T) {
		database := db.NewMockDB(t)

		userID := id.New()
		database.EXPECT().
			FindPublicKeyByFingerprint(mock.Anything, fingerprint).Return(nil, nil)
		database.EXPECT().
			CreateUserWithPublicKey(mock.Anything, mock.Anything).Return(&snips.User{
			ID: userID,
		}, nil)

		nextFunc := func(sesh cssh.Session) {
			assert.Equal(t, userID, sesh.Context().Value(ssh.UserIDContextKey))
			assert.Equal(t, fingerprint, sesh.Context().Value(ssh.FingerprintContextKey))
		}

		session := testsession.New(t, &cssh.Server{
			Handler: ssh.AssignUser(database, *testHost)(nextFunc),
			PublicKeyHandler: func(ctx cssh.Context, key cssh.PublicKey) bool {
				return true
			},
		}, &gossh.ClientConfig{
			Auth: []gossh.AuthMethod{
				testPrivateKeyAuth(privateKey),
			},
			Timeout: testTimeout,
		})

		_ = session.Run("")
	})

	t.Run("matches existing user by fingerprint", func(t *testing.T) {
		database := db.NewMockDB(t)

		userID := id.New()
		database.EXPECT().
			FindPublicKeyByFingerprint(mock.Anything, fingerprint).Return(&snips.PublicKey{
			UserID: userID,
		}, nil)
		database.EXPECT().
			FindUser(mock.Anything, userID).Return(&snips.User{
			ID: userID,
		}, nil)

		nextFunc := func(sesh cssh.Session) {
			assert.Equal(t, userID, sesh.Context().Value(ssh.UserIDContextKey))
			assert.Equal(t, fingerprint, sesh.Context().Value(ssh.FingerprintContextKey))
		}

		session := testsession.New(t, &cssh.Server{
			Handler: ssh.AssignUser(database, *testHost)(nextFunc),
			PublicKeyHandler: func(ctx cssh.Context, key cssh.PublicKey) bool {
				return true
			},
		}, &gossh.ClientConfig{
			Auth: []gossh.AuthMethod{
				testPrivateKeyAuth(privateKey),
			},
			Timeout: testTimeout,
		})

		_ = session.Run("")
	})
}

func TestBlockIfNoPublicKey(t *testing.T) {
	t.Run("password", func(t *testing.T) {
		nextFunc := func(sesh cssh.Session) {
			panic("this should not be called")
		}

		session := testsession.New(t, &cssh.Server{
			Handler: ssh.BlockIfNoPublicKey(nextFunc),
			PublicKeyHandler: func(ctx cssh.Context, key cssh.PublicKey) bool {
				return true
			},
			PasswordHandler: func(ctx cssh.Context, password string) bool {
				return true
			},
		}, &gossh.ClientConfig{
			Auth: []gossh.AuthMethod{
				gossh.Password("password"),
			},
			Timeout: testTimeout,
		})

		err := session.Run("")
		assert.Error(t, err)
	})

	t.Run("publickey", func(t *testing.T) {
		session := testsession.New(t, &cssh.Server{
			Handler: ssh.BlockIfNoPublicKey(func(sesh cssh.Session) {}),
			PublicKeyHandler: func(ctx cssh.Context, key cssh.PublicKey) bool {
				return true
			},
			PasswordHandler: func(ctx cssh.Context, password string) bool {
				return true
			},
		}, &gossh.ClientConfig{
			Auth: []gossh.AuthMethod{
				testPrivateKeyAuth(privateKey),
			},
			Timeout: testTimeout,
		})

		err := session.Run("")
		assert.NoError(t, err)
	})
}

func TestWithRequestID(t *testing.T) {
	nextFunc := func(sesh cssh.Session) {
		val := sesh.Context().Value(ssh.RequestIDContextKey)
		assert.NotNil(t, val)
	}

	session := testsession.New(t, &cssh.Server{
		Handler: ssh.WithRequestID(nextFunc),
	}, nil)

	err := session.Run("")
	assert.NoError(t, err)
}

func TestWithLogger(t *testing.T) {
	nextFunc := func(sesh cssh.Session) {
		val := sesh.Context().Value(logger.ContextKey)
		assert.NotNil(t, val)
	}

	session := testsession.New(t, &cssh.Server{
		Handler: ssh.WithRequestID(ssh.WithLogger(nextFunc)),
	}, nil)

	err := session.Run("")
	assert.NoError(t, err)
}

func TestWithPublicKeyAllowList(t *testing.T) {
	t.Run("no allowlist", func(t *testing.T) {
		session := testsession.New(t, &cssh.Server{
			Handler: ssh.WithPublicKeyAllowList(nil)(func(sesh cssh.Session) {}),
			PublicKeyHandler: func(ctx cssh.Context, key cssh.PublicKey) bool {
				return true
			},
		}, &gossh.ClientConfig{
			Auth: []gossh.AuthMethod{
				testPrivateKeyAuth(privateKey),
			},
			Timeout: testTimeout,
		})

		err := session.Run("")
		assert.NoError(t, err)
	})

	t.Run("allowlist blocks user", func(t *testing.T) {
		nextFunc := func(sesh cssh.Session) {
			panic("this should not be called")
		}

		session := testsession.New(t, &cssh.Server{
			Handler: ssh.WithPublicKeyAllowList(
				[]cssh.PublicKey{authorizedKey},
			)(nextFunc),
			PublicKeyHandler: func(ctx cssh.Context, key cssh.PublicKey) bool {
				return true
			},
		}, &gossh.ClientConfig{
			Auth: []gossh.AuthMethod{
				testPrivateKeyAuth(testdata.PEMBytes["rsa"]),
			},
			Timeout: testTimeout,
		})

		err := session.Run("")
		assert.Error(t, err)
	})

	t.Run("allowlist allows user", func(t *testing.T) {
		session := testsession.New(t, &cssh.Server{
			Handler: ssh.WithPublicKeyAllowList(
				[]cssh.PublicKey{authorizedKey},
			)(func(sesh cssh.Session) {}),
			PublicKeyHandler: func(ctx cssh.Context, key cssh.PublicKey) bool {
				return true
			},
		}, &gossh.ClientConfig{
			Auth: []gossh.AuthMethod{
				testPrivateKeyAuth(privateKey),
			},
			Timeout: testTimeout,
		})

		err := session.Run("")
		assert.NoError(t, err)
	})
}

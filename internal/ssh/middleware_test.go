package ssh_test

import (
	"testing"

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
	privateKey  = testdata.PEMBytes["ed25519"]
	fingerprint = "SHA256:mV1mPX4S6TE+odyfWDXGrC5fvQbLh+w8o2NK3q2MmYw"
)

func testPrivateKeyAuth() gossh.AuthMethod {
	signer, err := gossh.ParsePrivateKey(privateKey)
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
			Handler: ssh.AssignUser(database)(nextFunc),
			PublicKeyHandler: func(ctx cssh.Context, key cssh.PublicKey) bool {
				return true
			},
		}, &gossh.ClientConfig{
			Auth: []gossh.AuthMethod{
				testPrivateKeyAuth(),
			},
		})

		session.Run("")
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
			Handler: ssh.AssignUser(database)(nextFunc),
			PublicKeyHandler: func(ctx cssh.Context, key cssh.PublicKey) bool {
				return true
			},
		}, &gossh.ClientConfig{
			Auth: []gossh.AuthMethod{
				testPrivateKeyAuth(),
			},
		})

		session.Run("")
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
				testPrivateKeyAuth(),
			},
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

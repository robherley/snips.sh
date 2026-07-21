package web_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWithAuthentication(t *testing.T) {
	newToken := func(t *testing.T) (string, string) {
		token, hash, err := snips.NewAPIKeyToken()
		require.NoError(t, err)
		return token, hash
	}

	t.Run("rejects missing or malformed credentials", func(t *testing.T) {
		cases := []struct {
			name   string
			header string
		}{
			{name: "no authorization header", header: ""},
			{name: "not a bearer token", header: "Basic dXNlcjpwYXNz"},
			{name: "bearer token without snips prefix", header: "Bearer some-other-token"},
			{name: "bearer prefix only", header: "Bearer "},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// no DB expectations: malformed credentials must never hit the database
				database := db.NewMockDB(t)

				nextCalled := false
				handler := web.WithAuthentication(database, func(w http.ResponseWriter, r *http.Request) {
					nextCalled = true
				})

				req := httptest.NewRequest("GET", "/api/v1/user", nil)
				if tc.header != "" {
					req.Header.Set("Authorization", tc.header)
				}

				rec := httptest.NewRecorder()
				handler(rec, req)

				assert.False(t, nextCalled)
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
				assert.Equal(t, `Bearer realm="snips.sh api"`, rec.Header().Get("WWW-Authenticate"))

				assert.Equal(t, "missing or malformed api key\n", rec.Body.String())
			})
		}
	})

	t.Run("rejects unknown tokens", func(t *testing.T) {
		token, hash := newToken(t)

		database := db.NewMockDB(t)
		database.EXPECT().FindAPIKeyByTokenHash(mock.Anything, hash).Return(nil, nil).Once()

		nextCalled := false
		handler := web.WithAuthentication(database, func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})

		req := httptest.NewRequest("GET", "/api/v1/user", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, `Bearer realm="snips.sh api"`, rec.Header().Get("WWW-Authenticate"))

		assert.Equal(t, "unknown api key\n", rec.Body.String())
	})

	t.Run("fails closed on database errors", func(t *testing.T) {
		token, hash := newToken(t)

		database := db.NewMockDB(t)
		database.EXPECT().FindAPIKeyByTokenHash(mock.Anything, hash).Return(nil, errors.New("boom")).Once()

		nextCalled := false
		handler := web.WithAuthentication(database, func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})

		req := httptest.NewRequest("GET", "/api/v1/user", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Empty(t, rec.Header().Get("WWW-Authenticate"))
	})

	t.Run("accepts a valid token and stores the user id", func(t *testing.T) {
		token, hash := newToken(t)
		key := &snips.APIKey{ID: "key123", TokenHash: hash, UserID: "user123"}

		database := db.NewMockDB(t)
		database.EXPECT().FindAPIKeyByTokenHash(mock.Anything, hash).Return(key, nil).Once()
		database.EXPECT().TouchAPIKey(mock.Anything, "key123").Return(nil).Once()

		var (
			gotUserID string
			gotOK     bool
		)
		handler := web.WithAuthentication(database, func(w http.ResponseWriter, r *http.Request) {
			gotUserID, gotOK = web.UserID(r.Context())
		})

		req := httptest.NewRequest("GET", "/api/v1/user", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, gotOK)
		assert.Equal(t, "user123", gotUserID)
	})

	t.Run("proceeds when touching last_used_at fails", func(t *testing.T) {
		token, hash := newToken(t)
		key := &snips.APIKey{ID: "key123", TokenHash: hash, UserID: "user123"}

		database := db.NewMockDB(t)
		database.EXPECT().FindAPIKeyByTokenHash(mock.Anything, hash).Return(key, nil).Once()
		database.EXPECT().TouchAPIKey(mock.Anything, "key123").Return(errors.New("boom")).Once()

		nextCalled := false
		handler := web.WithAuthentication(database, func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})

		req := httptest.NewRequest("GET", "/api/v1/user", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.True(t, nextCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestUserID(t *testing.T) {
	// unauthenticated contexts report !ok rather than panicking
	req := httptest.NewRequest("GET", "/", nil)
	userID, ok := web.UserID(req.Context())
	assert.False(t, ok)
	assert.Empty(t, userID)
}

func TestWithAuthentication_ExpiredKey(t *testing.T) {
	token, hash, err := snips.NewAPIKeyToken()
	require.NoError(t, err)

	past := time.Now().UTC().Add(-time.Minute)
	key := &snips.APIKey{ID: "key123", TokenHash: hash, UserID: "user123", ExpiresAt: &past}

	database := db.NewMockDB(t)
	database.EXPECT().FindAPIKeyByTokenHash(mock.Anything, hash).Return(key, nil).Once()

	nextCalled := false
	handler := web.WithAuthentication(database, func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest("GET", "/api/v1/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "expired api key\n", rec.Body.String())
}

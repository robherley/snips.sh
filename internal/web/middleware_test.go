package web_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/signer"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/robherley/snips.sh/internal/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWithRequestID(t *testing.T) {
	t.Run("sets request ID header and context", func(t *testing.T) {
		var gotHeader string
		handler := web.WithRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotHeader = r.Header.Get(web.RequestIDHeader)
			w.WriteHeader(http.StatusOK)
		}))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)

		assert.NotEmpty(t, gotHeader)
	})

	t.Run("generates unique IDs per request", func(t *testing.T) {
		var ids []string
		handler := web.WithRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ids = append(ids, r.Header.Get(web.RequestIDHeader))
		}))

		for range 3 {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			handler.ServeHTTP(rec, req)
		}

		require.Len(t, ids, 3)
		assert.NotEqual(t, ids[0], ids[1])
		assert.NotEqual(t, ids[1], ids[2])
	})
}

func TestWithLogger(t *testing.T) {
	t.Run("calls next handler", func(t *testing.T) {
		called := false
		handler := web.WithLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		handler.ServeHTTP(rec, req)

		assert.True(t, called)
	})
}

func TestWithRecover(t *testing.T) {
	t.Run("recovers from panic and returns 500", func(t *testing.T) {
		handler := web.WithRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something went wrong")
		}))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("passes through when no panic", func(t *testing.T) {
		handler := web.WithRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestWithMetrics(t *testing.T) {
	t.Run("calls next handler", func(t *testing.T) {
		called := false
		handler := web.WithMetrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)

		assert.True(t, called)
	})
}

func TestWithMiddleware(t *testing.T) {
	t.Run("applies default middleware", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// WithRequestID should have set the header
			assert.NotEmpty(t, r.Header.Get(web.RequestIDHeader))
			w.WriteHeader(http.StatusOK)
		})

		handler := web.WithMiddleware(inner)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("applies extra middleware after defaults", func(t *testing.T) {
		var order []string

		extra := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "extra")
				next.ServeHTTP(w, r)
			})
		}

		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "inner")
			w.WriteHeader(http.StatusOK)
		})

		handler := web.WithMiddleware(inner, extra)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)

		// extra wraps inner, so extra runs first, then inner
		assert.Equal(t, []string{"extra", "inner"}, order)
	})
}

func TestPattern(t *testing.T) {
	t.Run("strips method prefix", func(t *testing.T) {
		r := &http.Request{Pattern: "GET /foo"}
		r.Method = "GET"
		assert.Equal(t, "/foo", web.Pattern(r))
	})

	t.Run("returns wildcard for empty pattern", func(t *testing.T) {
		r := &http.Request{}
		assert.Equal(t, "*", web.Pattern(r))
	})
}

func TestWithFile(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	hmacSigner := signer.New(cfg.HMACKey)

	newMux := func(t *testing.T, mockDB *db.MockDB, handler http.HandlerFunc) *http.ServeMux {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /f/{fileID}", web.WithFile(cfg, mockDB, handler))
		return mux
	}

	t.Run("sets file in context for public file", func(t *testing.T) {
		mockDB := db.NewMockDB(t)
		file := testutil.Fixtures.File(t)
		file.ID = "abc123"
		mockDB.EXPECT().FindFile(mock.Anything, "abc123").Return(&file, nil)

		var gotFile *snips.File
		mux := newMux(t, mockDB, func(w http.ResponseWriter, r *http.Request) {
			gotFile = web.FileFrom(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/f/abc123", nil)
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.NotNil(t, gotFile)
		assert.Equal(t, "abc123", gotFile.ID)
	})

	t.Run("sets signed=false for unsigned public file", func(t *testing.T) {
		mockDB := db.NewMockDB(t)
		file := testutil.Fixtures.File(t)
		file.ID = "pub1"
		mockDB.EXPECT().FindFile(mock.Anything, "pub1").Return(&file, nil)

		var gotSigned bool
		mux := newMux(t, mockDB, func(w http.ResponseWriter, r *http.Request) {
			gotSigned = web.IsSignedRequest(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/f/pub1", nil)
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.False(t, gotSigned)
	})

	t.Run("returns 404 when file not found", func(t *testing.T) {
		mockDB := db.NewMockDB(t)
		mockDB.EXPECT().FindFile(mock.Anything, "missing").Return(nil, nil)

		handlerCalled := false
		mux := newMux(t, mockDB, func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/f/missing", nil)
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.False(t, handlerCalled)
	})

	t.Run("returns 404 on database error", func(t *testing.T) {
		mockDB := db.NewMockDB(t)
		mockDB.EXPECT().FindFile(mock.Anything, "dberr").Return(nil, errors.New("db down"))

		handlerCalled := false
		mux := newMux(t, mockDB, func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/f/dberr", nil)
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.False(t, handlerCalled)
	})

	t.Run("returns 404 for unsigned private file", func(t *testing.T) {
		mockDB := db.NewMockDB(t)
		file := testutil.Fixtures.File(t)
		file.ID = "priv1"
		file.Private = true
		mockDB.EXPECT().FindFile(mock.Anything, "priv1").Return(&file, nil)

		handlerCalled := false
		mux := newMux(t, mockDB, func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/f/priv1", nil)
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.False(t, handlerCalled)
	})

	t.Run("allows access to signed private file", func(t *testing.T) {
		mockDB := db.NewMockDB(t)
		file := testutil.Fixtures.File(t)
		file.ID = "priv2"
		file.Private = true
		mockDB.EXPECT().FindFile(mock.Anything, "priv2").Return(&file, nil)

		signedURL, _ := hmacSigner.SignURLWithTTL(url.URL{Path: "/f/priv2"}, 1*time.Hour)

		var gotFile *snips.File
		var gotSigned bool
		mux := newMux(t, mockDB, func(w http.ResponseWriter, r *http.Request) {
			gotFile = web.FileFrom(r.Context())
			gotSigned = web.IsSignedRequest(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", fmt.Sprintf("%s?%s", signedURL.Path, signedURL.RawQuery), nil)
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.NotNil(t, gotFile)
		assert.Equal(t, "priv2", gotFile.ID)
		assert.True(t, gotSigned)
	})

	t.Run("returns 404 for expired signed private file", func(t *testing.T) {
		mockDB := db.NewMockDB(t)
		file := testutil.Fixtures.File(t)
		file.ID = "priv3"
		file.Private = true
		mockDB.EXPECT().FindFile(mock.Anything, "priv3").Return(&file, nil)

		expiredURL, _ := hmacSigner.SignURLWithTTL(url.URL{Path: "/f/priv3"}, -1*time.Hour)

		handlerCalled := false
		mux := newMux(t, mockDB, func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", fmt.Sprintf("%s?%s", expiredURL.Path, expiredURL.RawQuery), nil)
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.False(t, handlerCalled)
	})
}

func TestFileFrom(t *testing.T) {
	t.Run("returns nil for empty context", func(t *testing.T) {
		assert.Nil(t, web.FileFrom(context.Background()))
	})
}

func TestIsSignedRequest(t *testing.T) {
	t.Run("returns false for empty context", func(t *testing.T) {
		assert.False(t, web.IsSignedRequest(context.Background()))
	})
}

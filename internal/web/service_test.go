//nolint:goconst
package web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	dbmock "github.com/robherley/snips.sh/internal/db/mock"
	"github.com/robherley/snips.sh/internal/db/sqlite"
	"github.com/robherley/snips.sh/internal/signer"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/robherley/snips.sh/internal/web"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HTTPServiceSuite struct {
	suite.Suite

	config  *config.Config
	assets  web.Assets
	mockDB  *dbmock.Database
	service *web.Service
}

func TestHTTPServiceSuite(t *testing.T) {
	suite.Run(t, new(HTTPServiceSuite))
}

func (suite *HTTPServiceSuite) SetupSuite() {
	var err error
	suite.config, err = config.Load()
	suite.Require().NoError(err)

	suite.assets = testutil.Assets(suite.T())
}

func (suite *HTTPServiceSuite) SetupTest() {
	suite.mockDB = dbmock.NewDB(suite.T())

	var err error
	suite.service, err = web.New(suite.config, suite.mockDB.DB, suite.assets)
	suite.Require().NoError(err)
}

func (suite *HTTPServiceSuite) TestHTTPServer() {
	ts := httptest.NewServer(suite.service.Handler)
	defer ts.Close()

	signedFileID := "wdHzc62hsn"

	hmacSigner := signer.New(suite.config.HMACKey)
	validSigned, _ := hmacSigner.SignURLWithTTL(url.URL{
		Path: "/f/" + signedFileID,
	}, 1*time.Hour)
	invalidSigned, _ := hmacSigner.SignURLWithTTL(url.URL{
		Path: "/f/" + signedFileID,
	}, -1*time.Hour)
	burnSigned, _ := hmacSigner.SignURLWithOptions(url.URL{
		Path: "/f/burnfile01",
	}, 1*time.Hour, true)
	burnRawSigned, _ := hmacSigner.SignURLWithOptions(url.URL{
		Path:     "/f/burnraw001",
		RawQuery: "r=1",
	}, 1*time.Hour, true)
	burnOGSigned, _ := hmacSigner.SignURLWithOptions(url.URL{
		Path: "/f/burnog001/og.png",
	}, 1*time.Hour, true)
	burnRevSigned, _ := hmacSigner.SignURLWithOptions(url.URL{
		Path: "/f/burnrev01/rev",
	}, 1*time.Hour, true)

	cases := []struct {
		name     string
		method   string
		path     string
		expected int
		setup    func()
	}{
		{
			name:     "landing page",
			method:   "GET",
			path:     "/",
			expected: 200,
			setup:    func() {},
		},
		{
			name:     "health check",
			method:   "GET",
			path:     "/health",
			expected: 200,
			setup:    func() {},
		},
		{
			name:     "meta",
			method:   "GET",
			path:     "/meta.json",
			expected: 200,
			setup:    func() {},
		},
		{
			name:     "docs",
			method:   "GET",
			path:     "/docs/self-hosting.md",
			expected: 200,
			setup:    func() {},
		},
		{
			name:     "js assets",
			method:   "GET",
			path:     "/assets/index.js",
			expected: 200,
			setup:    func() {},
		},
		{
			name:     "css assets",
			method:   "GET",
			path:     "/assets/index.css",
			expected: 200,
			setup:    func() {},
		},
		{
			name:     "file that does not exist",
			method:   "GET",
			path:     "/f/foobar",
			expected: 404,
			setup: func() {
				suite.mockDB.Files.EXPECT().Find(mock.Anything, "foobar").Return(nil, nil)
			},
		},
		{
			name:     "public file via named path",
			method:   "GET",
			path:     "/f/eLcyRMrrgP/n/My-Notes",
			expected: 200,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = "eLcyRMrrgP"
				file.Name = "my-notes"

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
				suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("hello world"), nil)
				suite.mockDB.Revisions.EXPECT().CountByFileID(mock.Anything, file.ID).Return(int64(0), nil)
			},
		},
		{
			name:     "named path with wrong name",
			method:   "GET",
			path:     "/f/wrongname1/n/other-name",
			expected: 404,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = "wrongname1"
				file.Name = "my-notes"

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
			},
		},
		{
			name:     "named path for unnamed file",
			method:   "GET",
			path:     "/f/unnamed123/n/my-notes",
			expected: 404,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = "unnamed123"
				file.Name = ""

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
			},
		},
		{
			name:     "revisions via named path",
			method:   "GET",
			path:     "/f/namedrev12/n/my-notes/rev",
			expected: 200,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = "namedrev12"
				file.Name = "my-notes"

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
				suite.mockDB.Revisions.EXPECT().FindByFileID(mock.Anything, file.ID).Return(nil, nil)
			},
		},
		{
			name:     "public file",
			method:   "GET",
			path:     "/f/eLcyRMrrgP",
			expected: 200,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = "eLcyRMrrgP"

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
				suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("hello world"), nil)
				suite.mockDB.Revisions.EXPECT().CountByFileID(mock.Anything, file.ID).Return(int64(0), nil)
			},
		},
		{
			name:     "unsigned private file",
			method:   "GET",
			path:     "/f/" + signedFileID,
			expected: 404,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = signedFileID
				file.Private = true

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
			},
		},
		{
			name:     "signed private file",
			method:   "GET",
			path:     validSigned.Path + "?" + validSigned.RawQuery,
			expected: 200,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = signedFileID
				file.Private = true

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
				suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("hello world"), nil)
				suite.mockDB.Revisions.EXPECT().CountByFileID(mock.Anything, file.ID).Return(int64(0), nil)
			},
		},
		{
			name:     "expired signed private file",
			method:   "GET",
			path:     invalidSigned.Path + "?" + invalidSigned.RawQuery,
			expected: 404,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = signedFileID
				file.Private = true

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
			},
		},
		{
			name:     "burn after read signed private file",
			method:   "GET",
			path:     burnSigned.Path + "?" + burnSigned.RawQuery,
			expected: 200,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = "burnfile01"
				file.Private = true

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil).Once()
				suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("hello world"), nil).Once()
				suite.mockDB.Files.EXPECT().Delete(mock.Anything, file.ID).Return(nil).Once()
				suite.mockDB.Revisions.EXPECT().CountByFileID(mock.Anything, file.ID).Return(int64(0), nil).Once()
			},
		},
		{
			name:     "burn after read raw signed private file",
			method:   "GET",
			path:     burnRawSigned.Path + "?" + burnRawSigned.RawQuery,
			expected: 200,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = "burnraw001"
				file.Private = true

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil).Once()
				suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("hello world"), nil).Once()
				suite.mockDB.Files.EXPECT().Delete(mock.Anything, file.ID).Return(nil).Once()
			},
		},
		{
			name:     "burn signed og image does not consume",
			method:   "GET",
			path:     "/f/burnog001/og.png?" + burnOGSigned.RawQuery,
			expected: 200,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = "burnog001"
				file.Private = true

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil).Once()
			},
		},
		{
			name:     "burn signed revisions do not consume",
			method:   "GET",
			path:     "/f/burnrev01/rev?" + burnRevSigned.RawQuery,
			expected: 200,
			setup: func() {
				file := testutil.Fixtures.File(suite.T())
				file.ID = "burnrev01"
				file.Private = true

				suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil).Once()
				suite.mockDB.Revisions.EXPECT().FindByFileID(mock.Anything, file.ID).Return(nil, nil).Once()
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			tc.setup()

			req, err := http.NewRequest(tc.method, ts.URL+tc.path, nil)
			suite.Require().NoError(err)

			resp, err := ts.Client().Do(req)
			suite.Require().NoError(err)
			suite.Require().Equal(tc.expected, resp.StatusCode)
		})
	}

	suite.Run("burn after read signed private file only works once", func() {
		db, fileID := newBurnAfterReadTestDB(suite.T(), suite.config, []byte("hello world"))
		service, err := web.New(suite.config, db, suite.assets)
		require.NoError(suite.T(), err)

		server := httptest.NewServer(service.Handler)
		defer server.Close()

		externalURL, err := url.Parse(server.URL)
		require.NoError(suite.T(), err)
		cfg := *suite.config
		cfg.HTTP.External = *externalURL
		hmacSigner := signer.New(cfg.HMACKey)
		burnOnceSigned, _ := hmacSigner.SignURLWithOptions(url.URL{Path: "/f/" + fileID}, time.Hour, true)

		resp, err := server.Client().Get(server.URL + burnOnceSigned.Path + "?" + burnOnceSigned.RawQuery)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp, err = server.Client().Get(server.URL + burnOnceSigned.Path + "?" + burnOnceSigned.RawQuery)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})
}

func newBurnAfterReadTestDB(t *testing.T, cfg *config.Config, content []byte) (*db.DB, string) {
	t.Helper()

	database, err := sqlite.New(t.TempDir()+"/snips.db", cfg.FileCompression)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	require.NoError(t, database.Migrate(t.Context()))

	user, err := database.Users.CreateWithPublicKey(t.Context(), &snips.PublicKey{
		Fingerprint: "SHA256:web-burn-after-read-test",
		Type:        "ssh-ed25519",
	})
	require.NoError(t, err)

	file := &snips.File{
		Private: true,
		UserID:  user.ID,
		Type:    "plaintext",
	}
	require.NoError(t, database.Files.Create(t.Context(), file, content, 0))

	return database, file.ID
}

func (suite *HTTPServiceSuite) TestDocMarkdownAccept() {
	ts := httptest.NewServer(suite.service.Handler)
	defer ts.Close()

	suite.Run("landing page returns markdown with frontmatter", func() {
		req, err := http.NewRequest("GET", ts.URL+"/", nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept", "text/markdown")

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))
		suite.Require().Equal("Accept", resp.Header.Get("Vary"))

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		content := string(body)
		suite.Require().True(strings.HasPrefix(content, "---\n"))
		suite.Require().Contains(content, "type: markdown")
		suite.Require().Contains(content, "source: ")
	})

	suite.Run("doc page returns markdown with frontmatter", func() {
		req, err := http.NewRequest("GET", ts.URL+"/docs/self-hosting.md", nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept", "text/markdown")

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		content := string(body)
		suite.Require().Contains(content, "name: self-hosting.md")
		suite.Require().Contains(content, "source: ")
	})

	suite.Run("no markdown accept returns html", func() {
		req, err := http.NewRequest("GET", ts.URL+"/", nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().NotEqual("text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))
	})
}

func (suite *HTTPServiceSuite) TestFileMarkdownAccept() {
	ts := httptest.NewServer(suite.service.Handler)
	defer ts.Close()

	suite.Run("code file returns frontmatter and fenced code block", func() {
		file := testutil.Fixtures.File(suite.T())
		file.ID = "mdtest1"
		file.Type = "go"
		suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
		suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("hello world"), nil)

		req, err := http.NewRequest("GET", ts.URL+"/f/"+file.ID, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept", "text/markdown")

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))
		suite.Require().Equal("Accept", resp.Header.Get("Vary"))

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		content := string(body)
		suite.Require().True(strings.HasPrefix(content, "---\n"))
		suite.Require().Contains(content, "id: mdtest1")
		suite.Require().Contains(content, "type: go")
		suite.Require().Contains(content, "source: ")
		suite.Require().Contains(content, "```go\n")
		suite.Require().Contains(content, "```\n")
	})

	suite.Run("markdown file returns frontmatter and raw markdown", func() {
		file := testutil.Fixtures.File(suite.T())
		file.ID = "mdtest2"
		file.Type = "markdown"
		suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
		suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("# Hello\n\nworld\n"), nil)

		req, err := http.NewRequest("GET", ts.URL+"/f/"+file.ID, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept", "text/markdown")

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		content := string(body)
		suite.Require().Contains(content, "type: markdown")
		suite.Require().Contains(content, "# Hello\n\nworld\n")
	})

	suite.Run("binary file returns frontmatter and placeholder", func() {
		file := testutil.Fixtures.File(suite.T())
		file.ID = "mdtest3"
		file.Type = "binary"
		suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
		suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("hello world"), nil)

		req, err := http.NewRequest("GET", ts.URL+"/f/"+file.ID, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept", "text/markdown")

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		content := string(body)
		suite.Require().Contains(content, "type: binary")
		suite.Require().Contains(content, "_Binary file._\n")
	})

	suite.Run("accept with quality params", func() {
		file := testutil.Fixtures.File(suite.T())
		file.ID = "mdtest4"
		file.Type = "go"
		suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
		suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("hello world"), nil)

		req, err := http.NewRequest("GET", ts.URL+"/f/"+file.ID, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept", "text/html, text/markdown;q=0.9")

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		// Should still return markdown since text/markdown is in the Accept header
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))
	})

	suite.Run("no markdown accept returns html", func() {
		file := testutil.Fixtures.File(suite.T())
		file.ID = "mdtest5"
		file.Type = "go"
		suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
		suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("hello world"), nil)
		suite.mockDB.Revisions.EXPECT().CountByFileID(mock.Anything, file.ID).Return(int64(0), nil)

		req, err := http.NewRequest("GET", ts.URL+"/f/"+file.ID, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept", "text/html")

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().NotEqual("text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))
	})
}

func (suite *HTTPServiceSuite) TestFilePreviewMetadata() {
	ts := httptest.NewServer(suite.service.Handler)
	defer ts.Close()

	tests := []struct {
		name        string
		fileID      string
		fileName    string
		previewName string
	}{
		{name: "named file", fileID: "namedpreview", fileName: "my-notes", previewName: "my-notes"},
		{name: "unnamed file", fileID: "previewtest", previewName: "previewtest"},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			file := testutil.Fixtures.File(suite.T())
			file.ID = tt.fileID
			file.Name = tt.fileName
			file.Type = "go"

			suite.mockDB.Files.EXPECT().Find(mock.Anything, file.ID).Return(&file, nil)
			suite.mockDB.Files.EXPECT().FindContent(mock.Anything, file.ID).Return([]byte("package preview"), nil)
			suite.mockDB.Revisions.EXPECT().CountByFileID(mock.Anything, file.ID).Return(int64(0), nil)

			resp, err := ts.Client().Get(ts.URL + "/f/" + file.ID)
			suite.Require().NoError(err)
			defer resp.Body.Close()
			suite.Require().Equal(http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			suite.Require().NoError(err)
			html := string(body)
			suite.Contains(html, "<title>"+tt.previewName+" - snips.sh</title>")
			suite.Contains(html, `property="og:title" content="`+tt.previewName+` - snips.sh"`)
			suite.Contains(html, `name="twitter:title" content="`+tt.previewName+` - snips.sh"`)
			suite.Contains(html, `property="og:description" content="`+tt.previewName+` · go · 100 B ·`)
			previewPath := "/f/" + file.ID
			if file.Name != "" {
				previewPath += "/n/" + file.Name
			}
			suite.Contains(html, `property="og:url" content="http://localhost:8080`+previewPath+`"`)
			suite.Contains(html, `property="og:image" content="http://localhost:8080`+previewPath+`/og.png"`)
		})
	}
}

func (suite *HTTPServiceSuite) TestPprofEndpoints() {
	suite.Run("pprof unavailable when debug is off", func() {
		// Default config has Debug=false, so the route is not registered.
		ts := httptest.NewServer(suite.service.Handler)
		defer ts.Close()

		req, err := http.NewRequest("GET", ts.URL+"/_debug/pprof/goroutine", nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	suite.Run("pprof accessible from localhost when debug is on", func() {
		debugCfg := *suite.config
		debugCfg.Debug = true

		svc, err := web.New(&debugCfg, suite.mockDB.DB, suite.assets)
		suite.Require().NoError(err)

		ts := httptest.NewServer(svc.Handler)
		defer ts.Close()

		// httptest connects from 127.0.0.1, so WithLocalhostOnly should pass.
		req, err := http.NewRequest("GET", ts.URL+"/_debug/pprof/goroutine", nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(http.StatusOK, resp.StatusCode)
	})
}

func (suite *HTTPServiceSuite) TestAssetCaching() {
	ts := httptest.NewServer(suite.service.Handler)
	defer ts.Close()

	staticAssets := suite.assets.(*web.StaticAssets)
	hashedCSSPath := staticAssets.AssetPath("index.css")
	hashedJSPath := staticAssets.AssetPath("index.js")

	suite.Run("hashed css returns immutable cache", func() {
		req, err := http.NewRequest("GET", ts.URL+hashedCSSPath, nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("public, max-age=31536000, immutable", resp.Header.Get("Cache-Control"))
		suite.Require().Equal("Accept-Encoding", resp.Header.Get("Vary"))
		suite.Require().Equal("text/css", resp.Header.Get("Content-Type"))
	})

	suite.Run("hashed js returns immutable cache", func() {
		req, err := http.NewRequest("GET", ts.URL+hashedJSPath, nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("public, max-age=31536000, immutable", resp.Header.Get("Cache-Control"))
	})

	suite.Run("unhashed css returns short cache", func() {
		req, err := http.NewRequest("GET", ts.URL+"/assets/index.css", nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("public, max-age=60, must-revalidate", resp.Header.Get("Cache-Control"))
		suite.Require().NotEmpty(resp.Header.Get("ETag"))
	})

	suite.Run("gzip content encoding", func() {
		req, err := http.NewRequest("GET", ts.URL+hashedCSSPath, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := (&http.Client{Transport: &http.Transport{DisableCompression: true}}).Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("gzip", resp.Header.Get("Content-Encoding"))
	})

	suite.Run("zstd content encoding", func() {
		req, err := http.NewRequest("GET", ts.URL+hashedCSSPath, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept-Encoding", "zstd")

		resp, err := (&http.Client{Transport: &http.Transport{DisableCompression: true}}).Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("zstd", resp.Header.Get("Content-Encoding"))
	})

	suite.Run("no encoding returns raw", func() {
		req, err := http.NewRequest("GET", ts.URL+hashedCSSPath, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept-Encoding", "identity")

		resp, err := (&http.Client{Transport: &http.Transport{DisableCompression: true}}).Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Empty(resp.Header.Get("Content-Encoding"))

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		suite.Require().Equal(staticAssets.CSS(), body)
	})

	suite.Run("static file returns etag and cache headers", func() {
		req, err := http.NewRequest("GET", ts.URL+"/assets/img/favicon.png", nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("public, max-age=604800", resp.Header.Get("Cache-Control"))
		suite.Require().NotEmpty(resp.Header.Get("ETag"))
	})

	suite.Run("static file returns 304 for matching etag", func() {
		// First request to get the ETag
		req, err := http.NewRequest("GET", ts.URL+"/assets/img/favicon.png", nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		etag := resp.Header.Get("ETag")
		suite.Require().NotEmpty(etag)

		// Second request with If-None-Match
		req2, err := http.NewRequest("GET", ts.URL+"/assets/img/favicon.png", nil)
		suite.Require().NoError(err)
		req2.Header.Set("If-None-Match", etag)

		resp2, err := ts.Client().Do(req2)
		suite.Require().NoError(err)
		suite.Require().Equal(304, resp2.StatusCode)
	})
}

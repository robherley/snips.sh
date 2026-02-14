//nolint:goconst
package web_test

import (
	"io"
	gohttp "net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/signer"
	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/robherley/snips.sh/internal/web"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HTTPServiceSuite struct {
	suite.Suite

	config  *config.Config
	assets  web.Assets
	mockDB  *db.MockDB
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
	suite.mockDB = db.NewMockDB(suite.T())

	var err error
	suite.service, err = web.New(suite.config, suite.mockDB, suite.assets)
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
				suite.mockDB.EXPECT().FindFile(mock.Anything, "foobar").Return(nil, nil)
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

				suite.mockDB.EXPECT().FindFile(mock.Anything, file.ID).Return(&file, nil)
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

				suite.mockDB.EXPECT().FindFile(mock.Anything, file.ID).Return(&file, nil)
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

				suite.mockDB.EXPECT().FindFile(mock.Anything, file.ID).Return(&file, nil)
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

				suite.mockDB.EXPECT().FindFile(mock.Anything, file.ID).Return(&file, nil)
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			tc.setup()

			req, err := gohttp.NewRequest(tc.method, ts.URL+tc.path, nil)
			suite.Require().NoError(err)

			resp, err := ts.Client().Do(req)
			suite.Require().NoError(err)
			suite.Require().Equal(tc.expected, resp.StatusCode)
		})
	}
}

func (suite *HTTPServiceSuite) TestAssetCaching() {
	ts := httptest.NewServer(suite.service.Handler)
	defer ts.Close()

	staticAssets := suite.assets.(*web.StaticAssets)
	hashedCSSPath := staticAssets.AssetPath("index.css")
	hashedJSPath := staticAssets.AssetPath("index.js")

	suite.Run("hashed css returns immutable cache", func() {
		req, err := gohttp.NewRequest("GET", ts.URL+hashedCSSPath, nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("public, max-age=31536000, immutable", resp.Header.Get("Cache-Control"))
		suite.Require().Equal("Accept-Encoding", resp.Header.Get("Vary"))
		suite.Require().Equal("text/css", resp.Header.Get("Content-Type"))
	})

	suite.Run("hashed js returns immutable cache", func() {
		req, err := gohttp.NewRequest("GET", ts.URL+hashedJSPath, nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("public, max-age=31536000, immutable", resp.Header.Get("Cache-Control"))
	})

	suite.Run("unhashed css returns short cache", func() {
		req, err := gohttp.NewRequest("GET", ts.URL+"/assets/index.css", nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("public, max-age=60, must-revalidate", resp.Header.Get("Cache-Control"))
		suite.Require().NotEmpty(resp.Header.Get("ETag"))
	})

	suite.Run("gzip content encoding", func() {
		req, err := gohttp.NewRequest("GET", ts.URL+hashedCSSPath, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := (&gohttp.Client{Transport: &gohttp.Transport{DisableCompression: true}}).Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("gzip", resp.Header.Get("Content-Encoding"))
	})

	suite.Run("zstd content encoding", func() {
		req, err := gohttp.NewRequest("GET", ts.URL+hashedCSSPath, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept-Encoding", "zstd")

		resp, err := (&gohttp.Client{Transport: &gohttp.Transport{DisableCompression: true}}).Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("zstd", resp.Header.Get("Content-Encoding"))
	})

	suite.Run("no encoding returns raw", func() {
		req, err := gohttp.NewRequest("GET", ts.URL+hashedCSSPath, nil)
		suite.Require().NoError(err)
		req.Header.Set("Accept-Encoding", "identity")

		resp, err := (&gohttp.Client{Transport: &gohttp.Transport{DisableCompression: true}}).Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Empty(resp.Header.Get("Content-Encoding"))

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		suite.Require().Equal(staticAssets.CSS(), body)
	})

	suite.Run("static file returns etag and cache headers", func() {
		req, err := gohttp.NewRequest("GET", ts.URL+"/assets/img/favicon.png", nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		suite.Require().Equal("public, max-age=604800", resp.Header.Get("Cache-Control"))
		suite.Require().NotEmpty(resp.Header.Get("ETag"))
	})

	suite.Run("static file returns 304 for matching etag", func() {
		// First request to get the ETag
		req, err := gohttp.NewRequest("GET", ts.URL+"/assets/img/favicon.png", nil)
		suite.Require().NoError(err)

		resp, err := ts.Client().Do(req)
		suite.Require().NoError(err)
		suite.Require().Equal(200, resp.StatusCode)
		etag := resp.Header.Get("ETag")
		suite.Require().NotEmpty(etag)

		// Second request with If-None-Match
		req2, err := gohttp.NewRequest("GET", ts.URL+"/assets/img/favicon.png", nil)
		suite.Require().NoError(err)
		req2.Header.Set("If-None-Match", etag)

		resp2, err := ts.Client().Do(req2)
		suite.Require().NoError(err)
		suite.Require().Equal(304, resp2.StatusCode)
	})
}

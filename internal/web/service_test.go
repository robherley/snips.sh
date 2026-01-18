//nolint:goconst
package web_test

import (
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

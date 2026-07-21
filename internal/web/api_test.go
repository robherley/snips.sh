package web_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/testutil"
	"github.com/robherley/snips.sh/internal/web"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type APISuite struct {
	suite.Suite

	config *config.Config
	assets web.Assets
	mockDB *db.MockDB
	server *httptest.Server

	token  string
	apiKey *snips.APIKey
	userID string
}

func TestAPISuite(t *testing.T) {
	suite.Run(t, new(APISuite))
}

func (suite *APISuite) SetupSuite() {
	var err error
	suite.config, err = config.Load()
	suite.Require().NoError(err)

	suite.assets = testutil.Assets(suite.T())
}

func (suite *APISuite) SetupTest() {
	suite.mockDB = db.NewMockDB(suite.T())

	service, err := web.New(suite.config, suite.mockDB, suite.assets)
	suite.Require().NoError(err)

	suite.server = httptest.NewServer(service.Handler)

	token, hash, err := snips.NewAPIKeyToken()
	suite.Require().NoError(err)

	suite.token = token
	suite.userID = "user123"
	suite.apiKey = &snips.APIKey{
		ID:        "key123",
		TokenHash: hash,
		UserID:    suite.userID,
	}
}

func (suite *APISuite) TearDownTest() {
	suite.server.Close()
}

// expectAuth wires the mock calls every authenticated request makes.
func (suite *APISuite) expectAuth() {
	suite.mockDB.EXPECT().FindAPIKeyByTokenHash(mock.Anything, snips.HashAPIKeyToken(suite.token)).Return(suite.apiKey, nil).Once()
	suite.mockDB.EXPECT().TouchAPIKey(mock.Anything, suite.apiKey.ID).Return(nil).Once()
}

func (suite *APISuite) request(method, path string, body io.Reader, authed bool) *http.Response {
	req, err := http.NewRequest(method, suite.server.URL+path, body)
	suite.Require().NoError(err)

	if authed {
		req.Header.Set("Authorization", "Bearer "+suite.token)
	}

	res, err := suite.server.Client().Do(req)
	suite.Require().NoError(err)

	return res
}

func (suite *APISuite) decode(res *http.Response, v any) {
	defer res.Body.Close()
	suite.Require().NoError(json.NewDecoder(res.Body).Decode(v))
}

func (suite *APISuite) file(id string, private bool) *snips.File {
	file := &snips.File{
		ID:        id,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Private:   private,
		Type:      "plaintext",
		UserID:    suite.userID,
	}
	suite.Require().NoError(file.SetContent([]byte("hello world"), false))
	file.Size = 11

	return file
}

func (suite *APISuite) TestMetaMovedAndServed() {
	client := suite.server.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	res, err := client.Get(suite.server.URL + "/meta.json")
	suite.Require().NoError(err)
	defer res.Body.Close()
	suite.Equal(http.StatusMovedPermanently, res.StatusCode)
	suite.Equal("/api/v1/meta", res.Header.Get("Location"))

	res, err = client.Get(suite.server.URL + "/api/v1/meta")
	suite.Require().NoError(err)

	meta := map[string]any{}
	suite.decode(res, &meta)
	suite.Contains(meta, "limits")
	suite.Contains(meta["limits"], "api_keys_per_user")
}

func (suite *APISuite) TestUnauthorized() {
	for _, header := range []string{"", "Bearer nope", "Bearer " + snips.APIKeyTokenPrefix + "unknown", "Basic foo"} {
		req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/user", nil)
		suite.Require().NoError(err)
		if header != "" {
			req.Header.Set("Authorization", header)
		}

		if strings.HasPrefix(header, "Bearer "+snips.APIKeyTokenPrefix) {
			suite.mockDB.EXPECT().FindAPIKeyByTokenHash(mock.Anything, mock.Anything).Return(nil, nil).Once()
		}

		res, err := suite.server.Client().Do(req)
		suite.Require().NoError(err)
		res.Body.Close()
		suite.Equal(http.StatusUnauthorized, res.StatusCode, "header: %q", header)
	}
}

func (suite *APISuite) TestGetUser() {
	suite.expectAuth()
	suite.mockDB.EXPECT().FindUser(mock.Anything, suite.userID).Return(&snips.User{ID: suite.userID, CreatedAt: time.Now().UTC()}, nil).Once()

	res := suite.request("GET", "/api/v1/user", nil, true)
	suite.Equal(http.StatusOK, res.StatusCode)

	user := map[string]any{}
	suite.decode(res, &user)
	suite.Equal(suite.userID, user["id"])
}

func (suite *APISuite) TestListFiles_Paginated() {
	files := []*snips.File{}
	for _, id := range []string{"file3", "file2", "file1"} {
		files = append(files, suite.file(id, false))
	}

	// page size 2 → the handler asks for 3 and returns a next_cursor
	suite.expectAuth()
	suite.mockDB.EXPECT().FindFilesByUser(mock.Anything, suite.userID, mock.Anything, mock.Anything).Return(files, nil).Once()

	res := suite.request("GET", "/api/v1/files?limit=2", nil, true)
	suite.Equal(http.StatusOK, res.StatusCode)

	page := struct {
		Files      []map[string]any `json:"files"`
		NextCursor string           `json:"next_cursor"`
	}{}
	suite.decode(res, &page)
	suite.Len(page.Files, 2)
	suite.Equal("file3", page.Files[0]["id"])
	suite.NotEmpty(page.NextCursor)

	// following the cursor resumes at the encoded offset
	suite.expectAuth()
	suite.mockDB.EXPECT().FindFilesByUser(mock.Anything, suite.userID, mock.Anything, mock.Anything).Return(files[2:], nil).Once()

	res = suite.request("GET", "/api/v1/files?limit=2&cursor="+page.NextCursor, nil, true)
	suite.Equal(http.StatusOK, res.StatusCode)

	page.NextCursor = ""
	suite.decode(res, &page)
	suite.Len(page.Files, 1)
	suite.Equal("file1", page.Files[0]["id"])
	suite.Empty(page.NextCursor)
}

func (suite *APISuite) TestListFiles_BadCursorAndLimit() {
	suite.expectAuth()
	res := suite.request("GET", "/api/v1/files?cursor=!!!", nil, true)
	res.Body.Close()
	suite.Equal(http.StatusBadRequest, res.StatusCode)

	suite.expectAuth()
	res = suite.request("GET", "/api/v1/files?limit=9999", nil, true)
	res.Body.Close()
	suite.Equal(http.StatusBadRequest, res.StatusCode)
}

func (suite *APISuite) TestListFiles_ByName() {
	file := suite.file("file1", false)
	file.Name = "notes"

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFileByName(mock.Anything, suite.userID, "notes").Return(file, nil).Once()

	res := suite.request("GET", "/api/v1/files?name=notes", nil, true)
	suite.Equal(http.StatusOK, res.StatusCode)

	page := struct {
		Files []map[string]any `json:"files"`
	}{}
	suite.decode(res, &page)
	suite.Len(page.Files, 1)
	suite.Equal("notes", page.Files[0]["name"])
}

func (suite *APISuite) TestCreateFile() {
	suite.expectAuth()
	suite.mockDB.EXPECT().CreateFile(mock.Anything, mock.Anything, suite.config.Limits.FilesPerUser).RunAndReturn(
		func(_ context.Context, file *snips.File, _ uint64) error {
			file.ID = "newfile"
			return nil
		}).Once()

	res := suite.request("POST", "/api/v1/files?name=hello&private=true", strings.NewReader("hello world"), true)
	suite.Equal(http.StatusCreated, res.StatusCode)

	file := map[string]any{}
	suite.decode(res, &file)
	suite.Equal("newfile", file["id"])
	suite.Equal("hello", file["name"])
	suite.Equal(true, file["private"])
}

func (suite *APISuite) TestCreateFile_Errors() {
	// empty body
	suite.expectAuth()
	res := suite.request("POST", "/api/v1/files", strings.NewReader(""), true)
	res.Body.Close()
	suite.Equal(http.StatusBadRequest, res.StatusCode)

	// over the size limit
	suite.expectAuth()
	res = suite.request("POST", "/api/v1/files", strings.NewReader(strings.Repeat("a", int(suite.config.Limits.FileSize)+1)), true)
	res.Body.Close()
	suite.Equal(http.StatusRequestEntityTooLarge, res.StatusCode)

	// invalid name
	suite.expectAuth()
	res = suite.request("POST", "/api/v1/files?name=no/slashes", strings.NewReader("hi"), true)
	res.Body.Close()
	suite.Equal(http.StatusBadRequest, res.StatusCode)

	// name taken
	suite.expectAuth()
	suite.mockDB.EXPECT().CreateFile(mock.Anything, mock.Anything, mock.Anything).Return(db.ErrNameTaken).Once()
	res = suite.request("POST", "/api/v1/files?name=taken", strings.NewReader("hi"), true)
	res.Body.Close()
	suite.Equal(http.StatusConflict, res.StatusCode)

	// file limit
	suite.expectAuth()
	suite.mockDB.EXPECT().CreateFile(mock.Anything, mock.Anything, mock.Anything).Return(db.ErrFileLimit).Once()
	res = suite.request("POST", "/api/v1/files", strings.NewReader("hi"), true)
	res.Body.Close()
	suite.Equal(http.StatusUnprocessableEntity, res.StatusCode)
}

func (suite *APISuite) TestGetFile_Visibility() {
	// own private file is visible
	private := suite.file("mine", true)
	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "mine").Return(private, nil).Once()
	res := suite.request("GET", "/api/v1/files/mine", nil, true)
	res.Body.Close()
	suite.Equal(http.StatusOK, res.StatusCode)

	// someone else's public file is visible
	public := suite.file("theirs-public", false)
	public.UserID = "someone-else"
	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "theirs-public").Return(public, nil).Once()
	res = suite.request("GET", "/api/v1/files/theirs-public", nil, true)
	res.Body.Close()
	suite.Equal(http.StatusOK, res.StatusCode)

	// someone else's private file is a 404
	hidden := suite.file("theirs-private", true)
	hidden.UserID = "someone-else"
	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "theirs-private").Return(hidden, nil).Once()
	res = suite.request("GET", "/api/v1/files/theirs-private", nil, true)
	res.Body.Close()
	suite.Equal(http.StatusNotFound, res.StatusCode)

	// nonexistent file is a 404
	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "nope").Return(nil, nil).Once()
	res = suite.request("GET", "/api/v1/files/nope", nil, true)
	res.Body.Close()
	suite.Equal(http.StatusNotFound, res.StatusCode)
}

func (suite *APISuite) TestUpdateFile() {
	file := suite.file("file1", false)

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "file1").Return(file, nil).Once()
	suite.mockDB.EXPECT().UpdateFile(mock.Anything, mock.Anything).Return(nil).Once()

	res := suite.request("PATCH", "/api/v1/files/file1", strings.NewReader(`{"name":"renamed","private":true}`), true)
	suite.Equal(http.StatusOK, res.StatusCode)

	updated := map[string]any{}
	suite.decode(res, &updated)
	suite.Equal("renamed", updated["name"])
	suite.Equal(true, updated["private"])
}

func (suite *APISuite) TestUpdateFile_MutationsAreOwnerOnly() {
	public := suite.file("theirs", false)
	public.UserID = "someone-else"

	// even a public file 404s for non-owners on mutation
	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "theirs").Return(public, nil).Once()
	res := suite.request("PATCH", "/api/v1/files/theirs", strings.NewReader(`{"private":true}`), true)
	res.Body.Close()
	suite.Equal(http.StatusNotFound, res.StatusCode)
}

func (suite *APISuite) TestUpdateFile_EmptyPatch() {
	file := suite.file("file1", false)

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "file1").Return(file, nil).Once()

	res := suite.request("PATCH", "/api/v1/files/file1", strings.NewReader(`{}`), true)
	res.Body.Close()
	suite.Equal(http.StatusBadRequest, res.StatusCode)
}

func (suite *APISuite) TestDeleteFile() {
	file := suite.file("file1", false)

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "file1").Return(file, nil).Once()
	suite.mockDB.EXPECT().DeleteFile(mock.Anything, "file1").Return(nil).Once()

	res := suite.request("DELETE", "/api/v1/files/file1", nil, true)
	res.Body.Close()
	suite.Equal(http.StatusNoContent, res.StatusCode)
}

func (suite *APISuite) TestGetFileContent() {
	file := suite.file("file1", false)

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "file1").Return(file, nil).Once()

	res := suite.request("GET", "/api/v1/files/file1/content", nil, true)
	suite.Equal(http.StatusOK, res.StatusCode)
	suite.Contains(res.Header.Get("Content-Type"), "text/plain")

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	suite.Require().NoError(err)
	suite.Equal("hello world", string(body))
}

func (suite *APISuite) TestUpdateFileContent() {
	file := suite.file("file1", false)

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "file1").Return(file, nil).Once()
	suite.mockDB.EXPECT().CountRevisionsByFileID(mock.Anything, "file1").Return(0, nil).Once()
	suite.mockDB.EXPECT().CreateRevision(mock.Anything, mock.Anything, suite.config.Limits.RevisionsPerFile).Return(nil).Once()
	suite.mockDB.EXPECT().UpdateFile(mock.Anything, mock.Anything).Return(nil).Once()

	res := suite.request("PUT", "/api/v1/files/file1/content", strings.NewReader("hello new world"), true)
	suite.Equal(http.StatusOK, res.StatusCode)

	updated := map[string]any{}
	suite.decode(res, &updated)
	suite.Equal(float64(15), updated["size"])
}

func (suite *APISuite) TestListRevisions() {
	file := suite.file("file1", false)
	revisions := []*snips.Revision{
		{ID: "rev3", Sequence: 3, FileID: "file1", CreatedAt: time.Now().UTC()},
		{ID: "rev2", Sequence: 2, FileID: "file1", CreatedAt: time.Now().UTC()},
	}

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "file1").Return(file, nil).Once()
	suite.mockDB.EXPECT().FindRevisionsByFileID(mock.Anything, "file1", mock.Anything, mock.Anything).Return(revisions, nil).Once()

	res := suite.request("GET", "/api/v1/files/file1/revisions?limit=1", nil, true)
	suite.Equal(http.StatusOK, res.StatusCode)

	page := struct {
		Revisions  []map[string]any `json:"revisions"`
		NextCursor string           `json:"next_cursor"`
	}{}
	suite.decode(res, &page)
	suite.Len(page.Revisions, 1)
	suite.Equal(float64(3), page.Revisions[0]["sequence"])
	suite.NotEmpty(page.NextCursor)
}

func (suite *APISuite) TestGetRevision() {
	file := suite.file("file1", false)
	rev := &snips.Revision{ID: "rev1", Sequence: 1, FileID: "file1", CreatedAt: time.Now().UTC()}
	suite.Require().NoError(rev.SetDiff([]byte("+hello"), false))

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "file1").Return(file, nil).Once()
	suite.mockDB.EXPECT().FindRevisionByFileIDAndSequence(mock.Anything, "file1", int64(1)).Return(rev, nil).Once()

	res := suite.request("GET", "/api/v1/files/file1/revisions/1", nil, true)
	suite.Equal(http.StatusOK, res.StatusCode)

	revision := map[string]any{}
	suite.decode(res, &revision)
	suite.Equal("+hello", revision["diff"])
}

func (suite *APISuite) TestSignFile() {
	private := suite.file("file1", true)

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "file1").Return(private, nil).Once()

	res := suite.request("POST", "/api/v1/files/file1/sign", strings.NewReader(`{"ttl_seconds":3600}`), true)
	suite.Equal(http.StatusCreated, res.StatusCode)

	signed := map[string]any{}
	suite.decode(res, &signed)
	suite.Contains(signed["url"], "/f/file1")
	suite.Contains(signed["url"], "sig=")
	suite.NotEmpty(signed["expires_at"])
}

func (suite *APISuite) TestSignFile_PublicRejected() {
	public := suite.file("file1", false)

	suite.expectAuth()
	suite.mockDB.EXPECT().FindFile(mock.Anything, "file1").Return(public, nil).Once()

	res := suite.request("POST", "/api/v1/files/file1/sign", strings.NewReader(`{"ttl_seconds":3600}`), true)
	res.Body.Close()
	suite.Equal(http.StatusBadRequest, res.StatusCode)
}

func (suite *APISuite) TestOpenAPISpec() {
	for path, contentType := range map[string]string{
		"/openapi.yaml": "text/plain; charset=utf-8",
		"/openapi.yml":  "text/plain; charset=utf-8",
		"/openapi.json": "application/json",
	} {
		res, err := suite.server.Client().Get(suite.server.URL + path)
		suite.Require().NoError(err)
		suite.Equal(http.StatusOK, res.StatusCode, path)
		suite.Equal(contentType, res.Header.Get("Content-Type"), path)

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		suite.Require().NoError(err)
		suite.Contains(string(body), "openapi", path)

		if contentType == "application/json" {
			doc := map[string]any{}
			suite.Require().NoError(json.Unmarshal(body, &doc), path)
			suite.Contains(doc, "paths")
		}
	}
}

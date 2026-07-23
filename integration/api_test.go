package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/robherley/snips.sh/internal/web"
	"github.com/stretchr/testify/suite"
)

type apiClient struct {
	suite      *APIIntegrationSuite
	baseURL    string
	apiKey     string
	httpClient *http.Client
	createdIDs []string
}

type file struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Size    uint64 `json:"size"`
	Private bool   `json:"private"`
	Type    string `json:"type"`
}

type filesPage struct {
	Files      []file `json:"files"`
	NextCursor string `json:"next_cursor"`
}

type revision struct {
	Sequence int64  `json:"sequence"`
	Diff     string `json:"diff"`
}

type revisionsPage struct {
	Revisions  []revision `json:"revisions"`
	NextCursor string     `json:"next_cursor"`
}

type APIIntegrationSuite struct {
	suite.Suite
	client *apiClient
}

func TestAPIIntegrationSuite(t *testing.T) {
	suite.Run(t, new(APIIntegrationSuite))
}

func (s *APIIntegrationSuite) SetupTest() {
	s.client = newAPIClient(s)
	s.T().Cleanup(s.client.cleanup)
}

//nolint:gocyclo // Subtest assertions are intentionally kept beside each flow.
func (s *APIIntegrationSuite) TestAPIEndToEnd() {
	c := s.client

	s.T().Logf("running against %s", c.baseURL)
	runID := fmt.Sprintf("%d", time.Now().UnixNano())
	publicName := "e2e-" + runID + "-public"
	privateName := "e2e-" + runID + "-private"
	renamedName := "e2e-" + runID + "-renamed"
	publicContent := []byte("snips e2e public " + runID + "\n")
	privateContent := []byte("snips e2e private " + runID + "\n")
	updatedContent := []byte("# Updated by snips e2e\n\nRun: " + runID + "\n")
	var publicFile, privateFile file

	s.Run("discovery and authentication", func() {
		status, body := c.request(http.MethodGet, c.baseURL+"/openapi.json", nil, "", false)
		requireStatus(s, http.StatusOK, status, body)
		var spec struct {
			OpenAPI string                     `json:"openapi"`
			Paths   map[string]json.RawMessage `json:"paths"`
		}
		decodeJSON(s, body, &spec)
		s.Require().NotEmpty(spec.OpenAPI)
		s.Require().Contains(spec.Paths, "/files")
		s.Require().Contains(spec.Paths, "/files/{id}/sign")

		status, body = c.apiRequest(http.MethodGet, "/meta", nil, "", false)
		requireStatus(s, http.StatusOK, status, body)
		var metadata struct {
			Limits struct {
				FileSize struct {
					Bytes uint64 `json:"bytes"`
				} `json:"file_size"`
				FilesPerUser uint64 `json:"files_per_user"`
			} `json:"limits"`
			Endpoints struct {
				HTTP string `json:"http"`
			} `json:"endpoints"`
		}
		decodeJSON(s, body, &metadata)
		s.Require().NotZero(metadata.Limits.FileSize.Bytes)
		s.Require().NotZero(metadata.Limits.FilesPerUser)
		s.Require().NotEmpty(metadata.Endpoints.HTTP)

		status, body = c.apiRequest(http.MethodGet, "/user", nil, "", false)
		requireStatus(s, http.StatusUnauthorized, status, body)
		status, body = c.apiRequest(http.MethodGet, "/user", nil, "", true)
		requireStatus(s, http.StatusOK, status, body)
		var user struct {
			ID        string `json:"id"`
			CreatedAt string `json:"created_at"`
		}
		decodeJSON(s, body, &user)
		s.Require().NotEmpty(user.ID)
		s.Require().NotEmpty(user.CreatedAt)
	})

	s.Run("create public and private files", func() {
		publicFile = c.createFile(publicName, false, "txt", publicContent)
		s.Require().Equal(publicName, publicFile.Name)
		s.Require().False(publicFile.Private)
		s.Require().Equal(uint64(len(publicContent)), publicFile.Size)

		privateFile = c.createFile(privateName, true, "txt", privateContent)
		s.Require().Equal(privateName, privateFile.Name)
		s.Require().True(privateFile.Private)
	})

	s.Run("filter and paginate files", func() {
		query := url.Values{"name": {publicName}}
		status, body := c.apiRequest(http.MethodGet, "/files?"+query.Encode(), nil, "", true)
		requireStatus(s, http.StatusOK, status, body)
		var namedFiles filesPage
		decodeJSON(s, body, &namedFiles)
		s.Require().Len(namedFiles.Files, 1)
		s.Require().Equal(publicFile.ID, namedFiles.Files[0].ID)

		status, body = c.apiRequest(http.MethodGet, "/files?limit=1", nil, "", true)
		requireStatus(s, http.StatusOK, status, body)
		var firstPage filesPage
		decodeJSON(s, body, &firstPage)
		s.Require().Len(firstPage.Files, 1)
		s.Require().NotEmpty(firstPage.NextCursor)

		query = url.Values{"limit": {"1"}, "cursor": {firstPage.NextCursor}}
		status, body = c.apiRequest(http.MethodGet, "/files?"+query.Encode(), nil, "", true)
		requireStatus(s, http.StatusOK, status, body)
		var secondPage filesPage
		decodeJSON(s, body, &secondPage)
		s.Require().Len(secondPage.Files, 1)
	})

	s.Run("download file metadata and content", func() {
		status, body := c.apiRequest(http.MethodGet, "/files/"+publicFile.ID, nil, "", true)
		requireStatus(s, http.StatusOK, status, body)
		var fetched file
		decodeJSON(s, body, &fetched)
		s.Require().Equal(publicFile.ID, fetched.ID)
		s.Require().False(fetched.Private)

		status, body = c.apiRequest(http.MethodGet, "/files/"+publicFile.ID+"/content", nil, "", true)
		requireStatus(s, http.StatusOK, status, body)
		s.Require().Equal(publicContent, body)
	})

	s.Run("update metadata and content", func() {
		patch := marshalJSON(s, map[string]any{
			"name":    renamedName,
			"private": true,
			"type":    "md",
		})
		status, body := c.apiRequest(http.MethodPatch, "/files/"+publicFile.ID, patch, "application/json", true)
		requireStatus(s, http.StatusOK, status, body)
		var updated file
		decodeJSON(s, body, &updated)
		s.Require().Equal(renamedName, updated.Name)
		s.Require().True(updated.Private)
		s.Require().Equal("markdown", updated.Type)

		status, body = c.apiRequest(http.MethodPut, "/files/"+publicFile.ID+"/content?ext=md", updatedContent, "application/octet-stream", true)
		requireStatus(s, http.StatusOK, status, body)
		decodeJSON(s, body, &updated)
		s.Require().Equal(uint64(len(updatedContent)), updated.Size)
		s.Require().Equal("markdown", updated.Type)

		status, body = c.apiRequest(http.MethodGet, "/files/"+publicFile.ID+"/content", nil, "", true)
		requireStatus(s, http.StatusOK, status, body)
		s.Require().Equal(updatedContent, body)
	})

	s.Run("list and fetch revisions", func() {
		status, body := c.apiRequest(http.MethodGet, "/files/"+publicFile.ID+"/revisions?limit=1", nil, "", true)
		requireStatus(s, http.StatusOK, status, body)
		var revisionList revisionsPage
		decodeJSON(s, body, &revisionList)
		s.Require().Len(revisionList.Revisions, 1)
		s.Require().GreaterOrEqual(revisionList.Revisions[0].Sequence, int64(1))

		sequence := revisionList.Revisions[0].Sequence
		status, body = c.apiRequest(http.MethodGet, fmt.Sprintf("/files/%s/revisions/%d", publicFile.ID, sequence), nil, "", true)
		requireStatus(s, http.StatusOK, status, body)
		var fetchedRevision revision
		decodeJSON(s, body, &fetchedRevision)
		s.Require().Equal(sequence, fetchedRevision.Sequence)
		s.Require().NotEmpty(fetchedRevision.Diff)
	})

	s.Run("sign and access private file", func() {
		signRequest := marshalJSON(s, map[string]int{"ttl_seconds": 300})
		status, body := c.apiRequest(http.MethodPost, "/files/"+publicFile.ID+"/sign", signRequest, "application/json", true)
		requireStatus(s, http.StatusCreated, status, body)
		var signed struct {
			URL       string `json:"url"`
			ExpiresAt string `json:"expires_at"`
		}
		decodeJSON(s, body, &signed)
		s.Require().NotEmpty(signed.URL)
		s.Require().NotEmpty(signed.ExpiresAt)
		s.Require().Contains(signed.URL, "sig=")

		status, body = c.request(http.MethodGet, c.baseURL+"/f/"+publicFile.ID+"?r=1", nil, "", false)
		requireStatus(s, http.StatusNotFound, status, body)
		status, body = c.request(http.MethodGet, signed.URL, nil, "", false)
		requireStatus(s, http.StatusOK, status, body)
		s.Require().Equal(updatedContent, body)
	})

	s.Run("delete files", func() {
		c.deleteFile(privateFile.ID)
		status, body := c.apiRequest(http.MethodGet, "/files/"+privateFile.ID, nil, "", true)
		requireStatus(s, http.StatusNotFound, status, body)

		c.deleteFile(publicFile.ID)
		status, body = c.apiRequest(http.MethodGet, "/files/"+publicFile.ID, nil, "", true)
		requireStatus(s, http.StatusNotFound, status, body)
	})
}

func newAPIClient(s *APIIntegrationSuite) *apiClient {
	s.T().Helper()

	cfg, err := config.Load()
	s.Require().NoError(err)
	cfg.EnableGuesser = false

	database, err := db.NewSqlite(s.T().TempDir() + "/snips.db")
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if closer, ok := database.(interface{ Close() error }); ok {
			s.Assert().NoError(closer.Close())
		}
	})

	s.Require().NoError(database.Migrate(s.T().Context()))
	user, err := database.CreateUserWithPublicKey(s.T().Context(), &snips.PublicKey{
		Fingerprint: "SHA256:snips-api-integration-test",
		Type:        "ssh-ed25519",
	})
	s.Require().NoError(err)

	token, tokenHash, err := snips.NewAPIKeyToken()
	s.Require().NoError(err)
	s.Require().NoError(database.CreateAPIKey(s.T().Context(), &snips.APIKey{
		Name:      "integration-test",
		TokenHash: tokenHash,
		UserID:    user.ID,
	}, cfg.Limits.APIKeysPerUser))

	mux := http.NewServeMux()
	web.NewAPI(cfg, database).Register(mux)
	// Raw signed-file access is part of the API signing flow.
	mux.HandleFunc("GET /f/{fileID}", web.FileHandler(cfg, database, nil))
	server := httptest.NewServer(web.WithMiddleware(mux))
	s.T().Cleanup(server.Close)

	externalURL, err := url.Parse(server.URL)
	s.Require().NoError(err)
	cfg.HTTP.External = *externalURL

	return &apiClient{
		suite:      s,
		baseURL:    server.URL,
		apiKey:     token,
		httpClient: server.Client(),
	}
}

func (c *apiClient) createFile(name string, private bool, extension string, content []byte) file {
	c.suite.T().Helper()
	query := url.Values{
		"name":    {name},
		"private": {fmt.Sprintf("%t", private)},
		"ext":     {extension},
	}
	status, body := c.apiRequest(http.MethodPost, "/files?"+query.Encode(), content, "application/octet-stream", true)
	requireStatus(c.suite, http.StatusCreated, status, body)

	var created file
	decodeJSON(c.suite, body, &created)
	c.suite.Require().NotEmpty(created.ID)
	c.createdIDs = append(c.createdIDs, created.ID)
	return created
}

func (c *apiClient) deleteFile(id string) {
	c.suite.T().Helper()
	status, body := c.apiRequest(http.MethodDelete, "/files/"+id, nil, "", true)
	requireStatus(c.suite, http.StatusNoContent, status, body)
	for i, createdID := range c.createdIDs {
		if createdID == id {
			c.createdIDs = append(c.createdIDs[:i], c.createdIDs[i+1:]...)
			return
		}
	}
}

func (c *apiClient) cleanup() {
	for i := len(c.createdIDs) - 1; i >= 0; i-- {
		id := c.createdIDs[i]
		status, body := c.apiRequest(http.MethodDelete, "/files/"+id, nil, "", true)
		c.suite.Assert().Contains([]int{http.StatusNoContent, http.StatusNotFound}, status, "cleanup file %s: %s", id, body)
	}
}

func (c *apiClient) apiRequest(method, path string, body []byte, contentType string, authenticated bool) (int, []byte) {
	c.suite.T().Helper()
	return c.request(method, c.baseURL+"/api/v1"+path, body, contentType, authenticated)
}

func (c *apiClient) request(method, requestURL string, body []byte, contentType string, authenticated bool) (int, []byte) {
	c.suite.T().Helper()
	req, err := http.NewRequest(method, requestURL, bytes.NewReader(body))
	c.suite.Require().NoError(err)
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	// The web endpoint returns raw content to curl clients.
	req.Header.Set("User-Agent", "curl/snips-integration-test")

	res, err := c.httpClient.Do(req)
	c.suite.Require().NoError(err, "%s %s", method, requestURL)
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	c.suite.Require().NoError(err)
	return res.StatusCode, responseBody
}

func requireStatus(s *APIIntegrationSuite, expected, actual int, body []byte) {
	s.T().Helper()
	s.Require().Equal(expected, actual, "response body: %s", body)
}

func decodeJSON(s *APIIntegrationSuite, data []byte, target any) {
	s.T().Helper()
	s.Require().NoError(json.Unmarshal(data, target), "response body: %s", data)
}

func marshalJSON(s *APIIntegrationSuite, value any) []byte {
	s.T().Helper()
	data, err := json.Marshal(value)
	s.Require().NoError(err)
	return data
}

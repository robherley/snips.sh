package web

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/files"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/snips"
	"gopkg.in/yaml.v3"
)

const (
	APIDefaultPageSize         = 50
	APIMaxPageSize             = 100
	APIMaxSignTTLSeconds int64 = (1<<63 - 1) / int64(time.Second)
)

var (
	//go:embed openapi.yaml
	openapiYAML []byte
	// converted at init: a broken spec fails immediately, and is caught by tests
	openapiJSON = mustYAMLToJSON(openapiYAML)
)

var (
	errAPIContentTooLarge = errors.New("content too large")
	errAPIContentEmpty    = errors.New("content empty")
)

type API struct {
	cfg *config.Config
	db  db.DB
}

func NewAPI(cfg *config.Config, database db.DB) *API {
	return &API{cfg: cfg, db: database}
}

func (a *API) Register(mux *http.ServeMux) {
	authed := func(next http.HandlerFunc) http.HandlerFunc {
		return WithAuthentication(a.db, next)
	}

	mux.Handle("GET /meta.json", http.RedirectHandler("/api/v1/meta", http.StatusMovedPermanently))

	mux.HandleFunc("GET /openapi.json", a.OpenAPI)
	mux.HandleFunc("GET /openapi.yaml", a.OpenAPI)
	mux.HandleFunc("GET /openapi.yml", a.OpenAPI)

	mux.HandleFunc("GET /api/v1/meta", a.Meta)
	mux.HandleFunc("GET /api/v1/user", authed(a.User))
	mux.HandleFunc("GET /api/v1/files", authed(a.ListFiles))
	mux.HandleFunc("POST /api/v1/files", authed(a.CreateFile))
	mux.HandleFunc("GET /api/v1/files/{fileID}", authed(a.GetFile))
	mux.HandleFunc("PATCH /api/v1/files/{fileID}", authed(a.UpdateFile))
	mux.HandleFunc("DELETE /api/v1/files/{fileID}", authed(a.DeleteFile))
	mux.HandleFunc("GET /api/v1/files/{fileID}/content", authed(a.GetFileContent))
	mux.HandleFunc("PUT /api/v1/files/{fileID}/content", authed(a.UpdateFileContent))
	mux.HandleFunc("GET /api/v1/files/{fileID}/revisions", authed(a.ListRevisions))
	mux.HandleFunc("GET /api/v1/files/{fileID}/revisions/{sequence}", authed(a.GetRevision))
	mux.HandleFunc("POST /api/v1/files/{fileID}/sign", authed(a.SignFile))
}

func mustYAMLToJSON(in []byte) []byte {
	var doc any
	if err := yaml.Unmarshal(in, &doc); err != nil {
		panic(fmt.Sprintf("invalid openapi spec: %v", err))
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("invalid openapi spec: %v", err))
	}

	return out
}

// OpenAPI serves the specification as json or yaml, based on the extension.
func (a *API) OpenAPI(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, ".json") {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(openapiJSON)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(openapiYAML)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// findFile resolves {fileID} and enforces visibility: a file that doesn't
// exist is a 404, and so is another user's file when it's private (or when
// the operation is owner-only), so existence isn't leaked.
func (a *API) findFile(w http.ResponseWriter, r *http.Request, ownerOnly bool) *snips.File {
	file, err := a.db.FindFile(r.Context(), r.PathValue("fileID"))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil
	}

	userID, _ := UserID(r.Context())
	if file == nil || (file.UserID != userID && (ownerOnly || file.Private)) {
		http.Error(w, "file not found", http.StatusNotFound)
		return nil
	}

	return file
}

// readContent reads the raw request body, enforcing the file size limit.
func (a *API) readContent(r *http.Request) ([]byte, error) {
	maxSize := a.cfg.Limits.FileSize

	content, err := io.ReadAll(io.LimitReader(r.Body, int64(maxSize)+1))
	if err != nil {
		return nil, err
	}

	if uint64(len(content)) > maxSize {
		return nil, errAPIContentTooLarge
	}

	if len(content) == 0 {
		return nil, errAPIContentEmpty
	}

	return content, nil
}

// pageSize parses the limit query parameter, reporting failure with a 400.
func pageSize(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return APIDefaultPageSize, true
	}

	limit, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || limit < 1 || limit > APIMaxPageSize {
		http.Error(w, "limit must be an integer between 1 and "+strconv.Itoa(APIMaxPageSize), http.StatusBadRequest)
		return 0, false
	}

	return limit, true
}

// cursors are opaque so the pagination scheme can change without breaking
// clients; today they are just the base64-encoded row offset of the next page
func encodeCursor(offset uint64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatUint(offset, 10)))
}

func decodeCursor(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	cursor := r.URL.Query().Get("cursor")
	if cursor == "" {
		return 0, true
	}

	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return 0, false
	}

	offset, err := strconv.ParseUint(string(raw), 10, 64)
	if err != nil {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return 0, false
	}

	return offset, true
}

func (a *API) Meta(w http.ResponseWriter, r *http.Request) {
	metadata := map[string]any{
		"limits": map[string]any{
			"file_size": map[string]any{
				"bytes": a.cfg.Limits.FileSize,
				"human": humanize.Bytes(a.cfg.Limits.FileSize),
			},
			"files_per_user":     a.cfg.Limits.FilesPerUser,
			"revisions_per_file": a.cfg.Limits.RevisionsPerFile,
			"api_keys_per_user":  a.cfg.Limits.APIKeysPerUser,
			"session_duration": map[string]any{
				"seconds": a.cfg.Limits.SessionDuration.Seconds(),
				"human":   a.cfg.Limits.SessionDuration.String(),
			},
		},
		"endpoints": map[string]any{
			"http": a.cfg.HTTP.External.String(),
			"ssh":  a.cfg.SSH.External.String(),
		},
		"commit_sha":      config.BuildCommit(),
		"guesser_enabled": a.cfg.EnableGuesser,
	}

	writeJSON(w, http.StatusOK, metadata)
}

func (a *API) User(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())

	user, err := a.db.FindUser(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (a *API) ListFiles(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Files      []*snips.File `json:"files"`
		NextCursor string        `json:"next_cursor,omitempty"`
	}

	userID, _ := UserID(r.Context())

	// names are unique per user, so a name filter returns at most one file
	if name := r.URL.Query().Get("name"); name != "" {
		file, err := a.db.FindFileByName(r.Context(), userID, name)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		matches := []*snips.File{}
		if file != nil {
			matches = append(matches, file)
		}

		writeJSON(w, http.StatusOK, response{Files: matches})
		return
	}

	limit, ok := pageSize(w, r)
	if !ok {
		return
	}

	offset, ok := decodeCursor(w, r)
	if !ok {
		return
	}

	// fetch one extra row to learn whether another page exists
	userFiles, err := a.db.FindFilesByUser(r.Context(), userID, db.WithLimit(limit+1), db.WithOffset(offset))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := response{Files: userFiles}
	if uint64(len(userFiles)) > limit {
		resp.Files = userFiles[:limit]
		resp.NextCursor = encodeCursor(offset + limit)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *API) CreateFile(w http.ResponseWriter, r *http.Request) {
	content, err := a.readContent(r)
	if err != nil {
		switch {
		case errors.Is(err, errAPIContentTooLarge):
			http.Error(w, "content exceeds the file size limit", http.StatusRequestEntityTooLarge)
		case errors.Is(err, errAPIContentEmpty):
			http.Error(w, "content is empty", http.StatusBadRequest)
		default:
			http.Error(w, "unable to read content", http.StatusBadRequest)
		}
		return
	}

	query := r.URL.Query()

	name := ""
	if rawName := query.Get("name"); rawName != "" {
		name, err = snips.NormalizeName(rawName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	private := false
	if rawPrivate := query.Get("private"); rawPrivate != "" {
		private, err = strconv.ParseBool(rawPrivate)
		if err != nil {
			http.Error(w, "private must be a boolean", http.StatusBadRequest)
			return
		}
	}

	userID, _ := UserID(r.Context())
	file := &snips.File{
		Private: private,
		Size:    uint64(len(content)),
		UserID:  userID,
		Type:    renderer.DetectFileType(content, query.Get("ext"), a.cfg.EnableGuesser),
		Name:    name,
	}

	if err := file.SetContent(content, a.cfg.FileCompression); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := a.db.CreateFile(r.Context(), file, a.cfg.Limits.FilesPerUser); err != nil {
		switch {
		case errors.Is(err, db.ErrNameTaken):
			http.Error(w, "you already have a file with that name", http.StatusConflict)
		case errors.Is(err, db.ErrFileLimit):
			http.Error(w, "file limit reached", http.StatusUnprocessableEntity)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	metrics.IncrCounterWithLabels([]string{"file", "create"}, 1, []metrics.Label{
		{Name: "private", Value: strconv.FormatBool(file.Private)},
		{Name: "type", Value: file.Type},
	})
	logger.From(r.Context()).Info("file uploaded", "file_id", file.ID, "user_id", file.UserID, "size", file.Size, "private", file.Private, "file_type", file.Type)

	writeJSON(w, http.StatusCreated, file)
}

func (a *API) GetFile(w http.ResponseWriter, r *http.Request) {
	file := a.findFile(w, r, false)
	if file == nil {
		return
	}

	writeJSON(w, http.StatusOK, file)
}

func (a *API) UpdateFile(w http.ResponseWriter, r *http.Request) {
	file := a.findFile(w, r, true)
	if file == nil {
		return
	}

	var patch struct {
		Name    *string `json:"name"`
		Private *bool   `json:"private"`
		Type    *string `json:"type"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&patch); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if patch.Name == nil && patch.Private == nil && patch.Type == nil {
		http.Error(w, "nothing to update: provide name, private, and/or type", http.StatusBadRequest)
		return
	}

	if patch.Name != nil {
		// an empty name removes it
		name := ""
		if *patch.Name != "" {
			var err error
			name, err = snips.NormalizeName(*patch.Name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		file.Name = name
	}

	if patch.Private != nil {
		file.Private = *patch.Private
	}

	if patch.Type != nil {
		extension := strings.TrimSpace(*patch.Type)
		if extension == "" {
			http.Error(w, "type cannot be empty", http.StatusBadRequest)
			return
		}

		content, err := file.GetContent()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		file.Type = renderer.DetectFileType(content, extension, a.cfg.EnableGuesser)
	}

	if err := a.db.UpdateFile(r.Context(), file); err != nil {
		if errors.Is(err, db.ErrNameTaken) {
			http.Error(w, "you already have a file with that name", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	metrics.IncrCounter([]string{"file", "update"}, 1)
	logger.From(r.Context()).Info("file updated", "file_id", file.ID, "user_id", file.UserID)

	writeJSON(w, http.StatusOK, file)
}

func (a *API) DeleteFile(w http.ResponseWriter, r *http.Request) {
	file := a.findFile(w, r, true)
	if file == nil {
		return
	}

	if err := a.db.DeleteFile(r.Context(), file.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	metrics.IncrCounter([]string{"file", "delete"}, 1)
	logger.From(r.Context()).Info("file deleted", "file_id", file.ID, "user_id", file.UserID)

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) GetFileContent(w http.ResponseWriter, r *http.Request) {
	file := a.findFile(w, r, false)
	if file == nil {
		return
	}

	content, err := file.GetContent()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	contentType := "text/plain; charset=utf-8"
	if file.IsBinary() {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(content)
}

func (a *API) UpdateFileContent(w http.ResponseWriter, r *http.Request) {
	file := a.findFile(w, r, true)
	if file == nil {
		return
	}

	content, err := a.readContent(r)
	if err != nil {
		switch {
		case errors.Is(err, errAPIContentTooLarge):
			http.Error(w, "content exceeds the file size limit", http.StatusRequestEntityTooLarge)
		case errors.Is(err, errAPIContentEmpty):
			http.Error(w, "content is empty", http.StatusBadRequest)
		default:
			http.Error(w, "unable to read content", http.StatusBadRequest)
		}
		return
	}

	if err := files.UpdateContent(r.Context(), a.db, a.cfg, file, content, r.URL.Query().Get("ext")); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	metrics.IncrCounterWithLabels([]string{"file", "update"}, 1, []metrics.Label{
		{Name: "type", Value: file.Type},
	})
	logger.From(r.Context()).Info("file content updated", "file_id", file.ID, "user_id", file.UserID, "size", file.Size, "file_type", file.Type)

	writeJSON(w, http.StatusOK, file)
}

func (a *API) ListRevisions(w http.ResponseWriter, r *http.Request) {
	file := a.findFile(w, r, false)
	if file == nil {
		return
	}

	limit, ok := pageSize(w, r)
	if !ok {
		return
	}

	offset, ok := decodeCursor(w, r)
	if !ok {
		return
	}

	// fetch one extra row to learn whether another page exists
	revisions, err := a.db.FindRevisionsByFileID(r.Context(), file.ID, db.WithLimit(limit+1), db.WithOffset(offset))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	type response struct {
		Revisions  []*snips.Revision `json:"revisions"`
		NextCursor string            `json:"next_cursor,omitempty"`
	}

	resp := response{Revisions: revisions}
	if uint64(len(revisions)) > limit {
		resp.Revisions = revisions[:limit]
		resp.NextCursor = encodeCursor(offset + limit)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *API) GetRevision(w http.ResponseWriter, r *http.Request) {
	file := a.findFile(w, r, false)
	if file == nil {
		return
	}

	sequence, err := strconv.ParseInt(r.PathValue("sequence"), 10, 64)
	if err != nil || sequence < 1 {
		http.Error(w, "sequence must be a positive integer", http.StatusBadRequest)
		return
	}

	rev, err := a.db.FindRevisionByFileIDAndSequence(r.Context(), file.ID, sequence)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if rev == nil {
		http.Error(w, "revision not found", http.StatusNotFound)
		return
	}

	diff, err := rev.GetDiff()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, struct {
		*snips.Revision
		Diff string `json:"diff"`
	}{rev, string(diff)})
}

func (a *API) SignFile(w http.ResponseWriter, r *http.Request) {
	file := a.findFile(w, r, true)
	if file == nil {
		return
	}

	if !file.Private {
		http.Error(w, "only private files can be signed", http.StatusBadRequest)
		return
	}

	var body struct {
		TTLSeconds int64 `json:"ttl_seconds"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if body.TTLSeconds < 1 || body.TTLSeconds > APIMaxSignTTLSeconds {
		http.Error(w, fmt.Sprintf("ttl_seconds must be between 1 and %d", APIMaxSignTTLSeconds), http.StatusBadRequest)
		return
	}

	signedURL, expires := file.GetSignedURL(a.cfg, time.Duration(body.TTLSeconds)*time.Second)

	metrics.IncrCounter([]string{"file", "sign"}, 1)
	logger.From(r.Context()).Info("private file signed", "file_id", file.ID, "expires_at", expires)

	writeJSON(w, http.StatusCreated, map[string]any{
		"url":        signedURL.String(),
		"expires_at": expires.UTC(),
	})
}

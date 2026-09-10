package web

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	APIDefaultPageSize = 50
	APIMaxPageSize     = 100
)

// pageCursor carries the backend-neutral state for the two pagination schemes
// currently in use. SQLite uses Offset for offset pagination. PostgreSQL
// resolves ID to an internal auto-incrementing key and uses that key for
// keyset pagination.
type pageCursor struct {
	Offset uint64 `json:"o"`
	ID     string `json:"i,omitempty"`
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

// Cursors are opaque so each backend can use its preferred pagination scheme.
func encodeCursor(cursor pageCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeCursor(w http.ResponseWriter, r *http.Request) (pageCursor, bool) {
	cursor := r.URL.Query().Get("cursor")
	if cursor == "" {
		return pageCursor{}, true
	}

	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return pageCursor{}, false
	}

	decoded := pageCursor{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return pageCursor{}, false
	}
	if decoded.ID == "" {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return pageCursor{}, false
	}

	return decoded, true
}

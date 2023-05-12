package http

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/signer"
	"github.com/robherley/snips.sh/internal/snips"
)

func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("💚\n"))
}

func MetaHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		metadata := map[string]interface{}{
			"limits": map[string]interface{}{
				"file_size_bytes":         cfg.Limits.FileSize,
				"file_size_human":         humanize.Bytes(cfg.Limits.FileSize),
				"files_per_user":          cfg.Limits.FilesPerUser,
				"ssh_session_dur_seconds": cfg.Limits.SessionDuration.Seconds(),
				"ssh_session_dur_human":   cfg.Limits.SessionDuration.String(),
			},
			"guesser_enabled": cfg.EnableGuesser,
			"http":            cfg.HTTP.External.String(),
			"ssh":             cfg.SSH.External.String(),
		}

		metabites, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		_, _ = w.Write(metabites)
	}
}

func DocHandler(name string, assets *Assets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		content, err := assets.Doc(name)
		if err != nil {
			log.Error().Err(err).Msg("unable to load file")
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		md, err := renderer.ToMarkdown(content)
		if err != nil {
			log.Error().Err(err).Msg("unable to parse file")
			http.Error(w, "unable to parse file", http.StatusInternalServerError)
			return
		}

		vars := map[string]interface{}{
			"FileID":   name,
			"FileSize": humanize.Bytes(uint64(len(content))),
			"FileType": "markdown",
			"HTML":     md,
		}

		err = assets.Template().ExecuteTemplate(w, "file.go.html", vars)
		if err != nil {
			log.Error().Err(err).Msg("unable to render template")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func FileHandler(cfg *config.Config, database db.DB, tmpl *template.Template) http.HandlerFunc {
	signer := signer.New(cfg.HMACKey)
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		fileID := strings.TrimPrefix(r.URL.Path, "/f/")

		if fileID == "" {
			http.NotFound(w, r)
			return
		}

		file, err := database.FindFile(r.Context(), fileID)
		if err != nil {
			log.Error().Err(err).Msg("unable to lookup file")
			http.NotFound(w, r)
			return
		}

		if file == nil {
			http.NotFound(w, r)
			return
		}

		isSignedAndNotExpired := IsSignedAndNotExpired(signer, r)

		if file.Private && !isSignedAndNotExpired {
			log.Warn().Msg("attempted to access private file")
			http.NotFound(w, r)
			return
		}

		if ShouldSendRaw(r) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(file.Content)
			return
		}

		rawHref := "?r=1"
		if isSignedAndNotExpired {
			q := r.URL.Query()
			q.Del("sig")
			q.Add("r", "1")

			rawPathURL := url.URL{
				Path:     r.URL.Path,
				RawQuery: q.Encode(),
			}

			signedRawURL := signer.SignURL(rawPathURL)
			rawHref = signedRawURL.String()
		}

		var html template.HTML

		switch file.Type {
		case snips.FileTypeBinary:
			html = renderer.BinaryHTMLPlaceholder
		case snips.FileTypeMarkdown:
			html, err = renderer.ToMarkdown(file.Content)
			if err != nil {
				log.Error().Err(err).Msg("unable to parse file")
				http.Error(w, "unable to parse file", http.StatusInternalServerError)
				return
			}
		default:
			html, err = renderer.ToSyntaxHighlightedHTML(file.Type, file.Content)
			if err != nil {
				log.Error().Err(err).Msg("unable to parse file")
				http.Error(w, "unable to parse file", http.StatusInternalServerError)
				return
			}
		}

		vars := map[string]interface{}{
			"FileID":    file.ID,
			"FileSize":  humanize.Bytes(file.Size),
			"CreatedAt": humanize.Time(file.CreatedAt),
			"UpdatedAt": humanize.Time(file.UpdatedAt),
			"FileType":  strings.ToLower(file.Type),
			"RawHREF":   rawHref,
			"HTML":      html,
			"Private":   file.Private,
		}

		err = tmpl.ExecuteTemplate(w, "file.go.html", vars)
		if err != nil {
			log.Error().Err(err).Msg("unable to render template")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func IsSignedAndNotExpired(s *signer.Signer, r *http.Request) bool {
	if r.URL == nil {
		return false
	}

	urlToVerify := url.URL{
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}

	if !s.VerifyURL(urlToVerify) {
		return false
	}

	exp := r.URL.Query().Get("exp")
	if exp == "" {
		return false
	}

	expiresUnix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil {
		return false
	}

	return expiresUnix > time.Now().Unix()
}

func ShouldSendRaw(r *http.Request) bool {
	if isCurl := strings.Contains(r.Header.Get("user-agent"), "curl"); isCurl {
		return true
	}

	if _, hasRawParam := r.URL.Query()["r"]; hasRawParam {
		return true
	}

	return false
}

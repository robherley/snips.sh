package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/pprof"
	"net/url"
	"strings"

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

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	profiler := r.PathValue("profile")
	switch profiler {
	case "cmdline":
		pprof.Cmdline(w, r)
	case "profile":
		pprof.Profile(w, r)
	case "symbol":
		pprof.Symbol(w, r)
	case "trace":
		pprof.Trace(w, r)
	default:
		// Available profiles can be found in [runtime/pprof.Profile].
		// https://pkg.go.dev/runtime/pprof#Profile
		pprof.Handler(profiler).ServeHTTP(w, r)
	}
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

func DocHandler(assets Assets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		name := r.PathValue("name")
		if name == "" {
			name = "README.md"
		}

		content, err := assets.Doc(name)
		if err != nil {
			log.Error("unable to load file", "err", err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		md, err := renderer.ToMarkdown(content)
		if err != nil {
			log.Error("unable to parse file", "err", err)
			http.Error(w, "unable to parse file", http.StatusInternalServerError)
			return
		}

		vars := map[string]interface{}{
			"FileID":     name,
			"FileSize":   humanize.Bytes(uint64(len(content))),
			"FileType":   "markdown",
			"HTML":       md,
			"RawContent": string(content),
			"CommitSHA":  config.BuildCommit(),
		}

		err = assets.Template().ExecuteTemplate(w, "file.go.html", vars)
		if err != nil {
			log.Error("unable to render template", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func FileHandler(cfg *config.Config, database db.DB, assets Assets) http.HandlerFunc {
	signer := signer.New(cfg.HMACKey)
	tmpl := assets.Template()
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		fileID := r.PathValue("fileID")
		if fileID == "" {
			http.NotFound(w, r)
			return
		}

		file, err := database.FindFile(r.Context(), fileID)
		if err != nil {
			log.Error("unable to lookup file", "err", err)
			http.NotFound(w, r)
			return
		}

		if file == nil {
			http.NotFound(w, r)
			return
		}

		isSignedAndNotExpired := signer.VerifyURLAndNotExpired(*r.URL)

		if file.Private && !isSignedAndNotExpired {
			log.Warn("attempted to access private file")
			http.NotFound(w, r)
			return
		}

		content, err := file.GetContent()
		if err != nil {
			log.Error("unable to get file content", "err", err)
			http.Error(w, "unable to get file content", http.StatusInternalServerError)
			return
		}

		if ShouldSendRaw(r) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
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

		var (
			html template.HTML
			css  template.CSS
		)

		switch file.Type {
		case snips.FileTypeBinary:
			html = renderer.BinaryHTMLPlaceholder
		case snips.FileTypeMarkdown:
			html, err = renderer.ToMarkdown(content)
			if err != nil {
				log.Error("unable to parse file", "err", err)
				http.Error(w, "unable to parse file", http.StatusInternalServerError)
				return
			}
			css = renderer.GetSyntaxCSS()
		default:
			html, err = renderer.ToSyntaxHighlightedHTML(file.Type, content)
			if err != nil {
				log.Error("unable to parse file", "err", err)
				http.Error(w, "unable to parse file", http.StatusInternalServerError)
				return
			}
			css = renderer.GetSyntaxCSS()
		}

		vars := map[string]interface{}{
			"FileID":     file.ID,
			"FileSize":   humanize.Bytes(file.Size),
			"CreatedAt":  humanize.Time(file.CreatedAt),
			"UpdatedAt":  humanize.Time(file.UpdatedAt),
			"FileType":   strings.ToLower(file.Type),
			"RawHREF":    rawHref,
			"RawContent": string(content),
			"HTML":       html,
			"CSS":        css,
			"Private":    file.Private,
			"CommitSHA":  config.BuildCommit(),
		}

		err = tmpl.ExecuteTemplate(w, "file.go.html", vars)
		if err != nil {
			log.Error("unable to render template", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
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

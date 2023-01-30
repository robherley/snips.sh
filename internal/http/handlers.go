package http

import (
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
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// TODO(robherley): cute landing page
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✂️\n"))
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Healthy\n"))
}

func FileHandler(cfg *config.Config, database *db.DB, tmpl *template.Template) http.HandlerFunc {
	signer := signer.New(cfg.HMACKey)
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		fileID := strings.TrimPrefix(r.URL.Path, "/f/")

		file := db.File{}
		if err := database.First(&file, "id = ?", fileID).Error; err != nil {
			log.Error().Err(err).Msg("unable to lookup file")
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
			w.Write(file.Content)
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
		case "binary":
			html = renderer.BinaryHTMLPlaceholder
		case "markdown":
			html = template.HTML("todo!")
		default:
			code, err := renderer.ToSyntaxHighlightedCode(file.Type, file.Content)
			if err != nil {
				log.Error().Err(err).Msg("unable to parse file")
				http.Error(w, "unable to parse file", http.StatusInternalServerError)
				return
			}

			html = code
		}

		vars := map[string]interface{}{
			"FileID":    file.ID,
			"FileSize":  humanize.Bytes(file.Size),
			"CreatedAt": humanize.Time(file.CreatedAt),
			"FileType":  strings.ToLower(file.Type),
			"RawHREF":   rawHref,
			"HTML":      html,
		}

		tmpl.ExecuteTemplate(w, "file.go.html", vars)
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

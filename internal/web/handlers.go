package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"image/png"
	"net/http"
	"net/http/pprof"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/opengraph"
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
				"file_size": map[string]interface{}{
					"bytes": cfg.Limits.FileSize,
					"human": humanize.Bytes(cfg.Limits.FileSize),
				},
				"files_per_user":     cfg.Limits.FilesPerUser,
				"revisions_per_file": cfg.Limits.RevisionsPerFile,
				"session_duration": map[string]interface{}{
					"seconds": cfg.Limits.SessionDuration.Seconds(),
					"human":   cfg.Limits.SessionDuration.String(),
				},
			},
			"endpoints": map[string]interface{}{
				"http": cfg.HTTP.External.String(),
				"ssh":  cfg.SSH.External.String(),
			},
			"commit_sha":      config.BuildCommit(),
			"guesser_enabled": cfg.EnableGuesser,
		}

		metabites, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		_, _ = w.Write(metabites)
	}
}

func DocHandler(cfg *config.Config, assets Assets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		name := r.PathValue("name")
		if name == "" {
			name = readme
		}

		content, err := assets.Doc(name)
		if err != nil {
			log.Error("unable to load file", "err", err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if AcceptsMarkdown(r) {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Vary", "Accept")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(DocToMarkdown(cfg, name, content))
			return
		}

		md, err := renderer.ToMarkdown(content)
		if err != nil {
			log.Error("unable to parse file", "err", err)
			http.Error(w, "unable to parse file", http.StatusInternalServerError)
			return
		}

		var ogImageURL string
		if name == readme {
			ogImageURL = fmt.Sprintf("%s://%s/og.png", cfg.HTTP.External.Scheme, cfg.HTTP.External.Host)
		} else {
			ogImageURL = fmt.Sprintf("%s://%s/docs/%s/og.png", cfg.HTTP.External.Scheme, cfg.HTTP.External.Host, name)
		}

		ogDescription := fmt.Sprintf("%s · %s · %s", name, "markdown", humanize.Bytes(uint64(len(content))))

		vars := map[string]interface{}{
			"FileID":        name,
			"FileSize":      humanize.Bytes(uint64(len(content))),
			"FileType":      "markdown",
			"HTML":          md,
			"RawContent":    string(content),
			"CommitSHA":     config.BuildCommit(),
			"OGImageURL":    ogImageURL,
			"OGDescription": ogDescription,
		}

		err = assets.Template("file.go.html").Execute(w, vars)
		if err != nil {
			log.Error("unable to render template", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func DocOGImageHandler(cfg *config.Config, assets Assets) http.HandlerFunc {
	ogRenderer := newOGRenderer(assets)

	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		name := r.PathValue("name")
		if name == "" {
			name = readme
		}

		content, err := assets.Doc(name)
		if err != nil {
			log.Error("unable to load doc", "err", err)
			http.NotFound(w, r)
			return
		}

		imgBytes, err := ogRenderer.GenerateImage(&opengraph.FileInfo{
			ID:   name,
			Type: "markdown",
			Size: uint64(len(content)),
		})
		if err != nil {
			log.Error("unable to generate og image", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imgBytes)
	}
}

func FileHandler(cfg *config.Config, database db.DB, assets Assets) http.HandlerFunc {
	signer := signer.New(cfg.HMACKey)
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

		if AcceptsMarkdown(r) {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Vary", "Accept")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(FileToMarkdown(cfg, file, content))
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

		revisionCount, err := database.CountRevisionsByFileID(r.Context(), file.ID)
		if err != nil {
			log.Warn("unable to count revisions", "err", err)
		}

		ogImageURL := fmt.Sprintf("%s://%s/f/%s/og.png", cfg.HTTP.External.Scheme, cfg.HTTP.External.Host, file.ID)
		ogDescription := fmt.Sprintf("%s · %s · %s · %s", file.ID, strings.ToLower(file.Type), humanize.Bytes(file.Size), humanize.Time(file.UpdatedAt))

		vars := map[string]interface{}{
			"FileID":        file.ID,
			"FileSize":      humanize.Bytes(file.Size),
			"CreatedAt":     humanize.Time(file.CreatedAt),
			"UpdatedAt":     humanize.Time(file.UpdatedAt),
			"FileType":      strings.ToLower(file.Type),
			"RawHREF":       rawHref,
			"RawContent":    string(content),
			"HTML":          html,
			"CSS":           css,
			"Private":       file.Private,
			"CommitSHA":     config.BuildCommit(),
			"OGImageURL":    ogImageURL,
			"OGDescription": ogDescription,
			"RevisionCount": revisionCount,
		}

		err = assets.Template("file.go.html").Execute(w, vars)
		if err != nil {
			log.Error("unable to render template", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func newOGRenderer(assets Assets) *opengraph.Renderer {
	loadFont := func(name string) []byte {
		data, ok := assets.StaticFile(name)
		if !ok {
			panic("missing required font: " + name)
		}
		return data
	}

	fonts := &opengraph.Fonts{
		Regular:     loadFont("fonts/GeistMono-Regular.ttf"),
		Display:     loadFont("fonts/GeistPixel-Square.ttf"),
		DisplayLine: loadFont("fonts/GeistPixel-Line.ttf"),
	}

	logoData, ok := assets.StaticFile("img/og-logo.png")
	if !ok {
		panic("missing required asset: img/og-logo.png")
	}

	logo, err := png.Decode(bytes.NewReader(logoData))
	if err != nil {
		panic("unable to decode og logo: " + err.Error())
	}

	renderer, err := opengraph.NewRenderer(fonts, logo)
	if err != nil {
		panic("unable to create og renderer: " + err.Error())
	}

	return renderer
}

func OGImageHandler(cfg *config.Config, database db.DB, assets Assets) http.HandlerFunc {
	sgnr := signer.New(cfg.HMACKey)
	renderer := newOGRenderer(assets)

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

		if file.Private && !sgnr.VerifyURLAndNotExpired(*r.URL) {
			http.NotFound(w, r)
			return
		}

		imgBytes, err := renderer.GenerateImage(&opengraph.FileInfo{
			ID:        file.ID,
			Type:      file.Type,
			Size:      file.Size,
			UpdatedAt: file.UpdatedAt,
		})
		if err != nil {
			log.Error("unable to generate og image", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imgBytes)
	}
}

func RevisionsHandler(cfg *config.Config, database db.DB, assets Assets) http.HandlerFunc {
	sgnr := signer.New(cfg.HMACKey)
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

		isSignedAndNotExpired := sgnr.VerifyURLAndNotExpired(*r.URL)

		if file.Private && !isSignedAndNotExpired {
			log.Warn("attempted to access private file revisions")
			http.NotFound(w, r)
			return
		}

		revisions, err := database.FindRevisionsByFileID(r.Context(), file.ID)
		if err != nil {
			log.Error("unable to lookup revisions", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		type revisionItem struct {
			ID        int64
			CreatedAt string
			Size      string
			Type      string
		}

		items := make([]revisionItem, len(revisions))
		for i, rev := range revisions {
			items[i] = revisionItem{
				ID:        rev.ID,
				CreatedAt: humanize.Time(rev.CreatedAt),
				Size:      humanize.Bytes(rev.Size),
				Type:      strings.ToLower(rev.Type),
			}
		}

		vars := map[string]interface{}{
			"FileID":       file.ID,
			"FileSize":     humanize.Bytes(file.Size),
			"FileType":     strings.ToLower(file.Type),
			"UpdatedAt":    humanize.Time(file.UpdatedAt),
			"Private":      file.Private,
			"Revisions":    items,
			"MaxRevisions": cfg.Limits.RevisionsPerFile,
			"CommitSHA":    config.BuildCommit(),
		}

		err = assets.Template("revisions.go.html").Execute(w, vars)
		if err != nil {
			log.Error("unable to render template", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func RevisionDiffHandler(cfg *config.Config, database db.DB, assets Assets) http.HandlerFunc {
	sgnr := signer.New(cfg.HMACKey)
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		fileID := r.PathValue("fileID")
		revisionIDStr := r.PathValue("revisionID")
		if fileID == "" || revisionIDStr == "" {
			http.NotFound(w, r)
			return
		}

		revisionID, err := strconv.ParseInt(revisionIDStr, 10, 64)
		if err != nil {
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

		isSignedAndNotExpired := sgnr.VerifyURLAndNotExpired(*r.URL)

		if file.Private && !isSignedAndNotExpired {
			log.Warn("attempted to access private file revision")
			http.NotFound(w, r)
			return
		}

		revision, err := database.FindRevision(r.Context(), file.ID, revisionID)
		if err != nil {
			log.Error("unable to lookup revision", "err", err)
			http.NotFound(w, r)
			return
		}

		if revision == nil {
			http.NotFound(w, r)
			return
		}

		diffContent, err := revision.GetDiff()
		if err != nil {
			log.Error("unable to decompress diff", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		diffLines := parseDiffLines(string(diffContent))

		vars := map[string]interface{}{
			"FileID":     file.ID,
			"FileSize":   humanize.Bytes(file.Size),
			"FileType":   strings.ToLower(file.Type),
			"Private":    file.Private,
			"RevisionID": revision.ID,
			"CreatedAt":  humanize.Time(revision.CreatedAt),
			"RevSize":    humanize.Bytes(revision.Size),
			"RevType":    strings.ToLower(revision.Type),
			"DiffLines":  diffLines,
			"CommitSHA":  config.BuildCommit(),
		}

		err = assets.Template("revision.go.html").Execute(w, vars)
		if err != nil {
			log.Error("unable to render template", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

type diffLine struct {
	Class   string
	Content string
}

func parseDiffLines(diff string) []diffLine {
	lines := strings.Split(diff, "\n")
	result := make([]diffLine, 0, len(lines))
	for _, line := range lines {
		var class string
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"), strings.HasPrefix(line, "@@"):
			class = "diff-hdr"
		case strings.HasPrefix(line, "+"):
			class = "diff-add"
		case strings.HasPrefix(line, "-"):
			class = "diff-del"
		default:
			class = "diff-ctx"
		}
		result = append(result, diffLine{Class: class, Content: line})
	}
	return result
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

func AcceptsMarkdown(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept"), ",") {
		if strings.TrimSpace(strings.SplitN(part, ";", 2)[0]) == "text/markdown" {
			return true
		}
	}
	return false
}

func FileToMarkdown(cfg *config.Config, file *snips.File, content []byte) []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "---\n")
	fmt.Fprintf(&buf, "id: %s\n", file.ID)
	fmt.Fprintf(&buf, "size: %s\n", humanize.Bytes(file.Size))
	fmt.Fprintf(&buf, "type: %s\n", strings.ToLower(file.Type))
	fmt.Fprintf(&buf, "created: %s\n", file.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&buf, "updated: %s\n", file.UpdatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&buf, "source: %s://%s/f/%s\n", cfg.HTTP.External.Scheme, cfg.HTTP.External.Host, file.ID)
	fmt.Fprintf(&buf, "---\n\n")

	switch file.Type {
	case snips.FileTypeBinary:
		buf.WriteString("_Binary file._\n")
	case snips.FileTypeMarkdown:
		buf.Write(content)
	default:
		fmt.Fprintf(&buf, "```%s\n", file.Type)
		buf.Write(content)
		if len(content) > 0 && content[len(content)-1] != '\n' {
			buf.WriteByte('\n')
		}
		buf.WriteString("```\n")
	}

	return buf.Bytes()
}

func DocToMarkdown(cfg *config.Config, name string, content []byte) []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "---\n")
	fmt.Fprintf(&buf, "name: %s\n", name)
	fmt.Fprintf(&buf, "type: markdown\n")
	if name == readme {
		fmt.Fprintf(&buf, "source: %s://%s/\n", cfg.HTTP.External.Scheme, cfg.HTTP.External.Host)
	} else {
		fmt.Fprintf(&buf, "source: %s://%s/docs/%s\n", cfg.HTTP.External.Scheme, cfg.HTTP.External.Host, name)
	}
	fmt.Fprintf(&buf, "---\n\n")

	buf.Write(content)

	return buf.Bytes()
}

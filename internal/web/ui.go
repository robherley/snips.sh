package web

import (
	"bytes"
	"fmt"
	"html/template"
	"image/png"
	"net/http"
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

type UI struct {
	cfg    *config.Config
	db     *db.DB
	assets Assets
	signer *signer.Signer
	og     *opengraph.Renderer
}

func NewUI(cfg *config.Config, database *db.DB, assets Assets) *UI {
	return &UI{
		cfg:    cfg,
		db:     database,
		assets: assets,
		signer: signer.New(cfg.HMACKey),
		og:     newOG(assets),
	}
}

func (ui *UI) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", ui.Landing)
	mux.HandleFunc("GET /docs/{name}", ui.Doc)
	mux.HandleFunc("GET /og.png", ui.DocOGImage)
	mux.HandleFunc("GET /docs/{name}/og.png", ui.DocOGImage)
	mux.HandleFunc("GET /f/{fileID}", ui.File)
	mux.HandleFunc("GET /f/{fileID}/rev", ui.Revisions)
	mux.HandleFunc("GET /f/{fileID}/rev/{revisionID}", ui.RevisionDiff)
	mux.HandleFunc("GET /f/{fileID}/og.png", ui.OGImage)
	mux.HandleFunc("GET /f/{fileID}/n/{name}", ui.File)
	mux.HandleFunc("GET /f/{fileID}/n/{name}/rev", ui.Revisions)
	mux.HandleFunc("GET /f/{fileID}/n/{name}/rev/{revisionID}", ui.RevisionDiff)
	mux.HandleFunc("GET /f/{fileID}/n/{name}/og.png", ui.OGImage)
	mux.HandleFunc("GET /assets/{asset...}", ui.assets.Serve)
}

func (ui *UI) Landing(w http.ResponseWriter, r *http.Request) {
	sshHost := ui.cfg.SSH.External.Hostname()
	sshPort := ui.cfg.SSH.External.Port()

	portFlag := ""
	if sshPort != "" && sshPort != "22" {
		portFlag = fmt.Sprintf("-p %s ", sshPort)
	}

	httpAddr := fmt.Sprintf("%s://%s", ui.cfg.HTTP.External.Scheme, ui.cfg.HTTP.External.Host)

	vars := map[string]interface{}{
		"SSHHost":                  sshHost,
		"SSHPort":                  sshPort,
		"HTTPAddr":                 httpAddr,
		"SSHCommand":               fmt.Sprintf("ssh %s%s", portFlag, sshHost),
		"SSHCommandForFile":        fmt.Sprintf("ssh %sf:<id>@%s", portFlag, sshHost),
		"SSHCommandForNamedFile":   fmt.Sprintf("ssh %sn:<name>@%s", portFlag, sshHost),
		"SSHCommandForFileContent": fmt.Sprintf("ssh %sf:<id>:content@%s", portFlag, sshHost),
		"CommitSHA":                config.BuildCommit(),
		"OGImageURL":               httpAddr + "/og.png",
	}

	log := logger.From(r.Context())

	if AcceptsMarkdown(r) {
		content, err := ui.assets.Doc(readme)
		if err != nil {
			log.Error("unable to load readme", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Vary", "Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(DocToMarkdown(ui.cfg, readme, content))
		return
	}

	err := ui.assets.Template("landing.go.html").Execute(w, vars)
	if err != nil {
		log.Error("unable to render template", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (ui *UI) Doc(w http.ResponseWriter, r *http.Request) {
	log := logger.From(r.Context())

	name := r.PathValue("name")
	if name == "" {
		name = readme
	}

	content, err := ui.assets.Doc(name)
	if err != nil {
		log.Error("unable to load file", "err", err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if AcceptsMarkdown(r) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Vary", "Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(DocToMarkdown(ui.cfg, name, content))
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
		ogImageURL = fmt.Sprintf("%s://%s/og.png", ui.cfg.HTTP.External.Scheme, ui.cfg.HTTP.External.Host)
	} else {
		ogImageURL = fmt.Sprintf("%s://%s/docs/%s/og.png", ui.cfg.HTTP.External.Scheme, ui.cfg.HTTP.External.Host, name)
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

	err = ui.assets.Template("file.go.html").Execute(w, vars)
	if err != nil {
		log.Error("unable to render template", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (ui *UI) DocOGImage(w http.ResponseWriter, r *http.Request) {
	log := logger.From(r.Context())

	name := r.PathValue("name")
	if name == "" {
		name = readme
	}

	content, err := ui.assets.Doc(name)
	if err != nil {
		log.Error("unable to load doc", "err", err)
		http.NotFound(w, r)
		return
	}

	var img bytes.Buffer
	err = ui.og.WriteImage(&img, &opengraph.FileInfo{
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
	_, _ = img.WriteTo(w)
}

// findFile resolves the {fileID} path segment. When the route carries an
// /n/{name} segment, it must match the file's name (case-insensitively)
// or the file is treated as not found, so named links can't be spoofed.
func (ui *UI) findFile(r *http.Request) (*snips.File, error) {
	fileID := r.PathValue("fileID")
	if fileID == "" {
		return nil, nil
	}

	file, err := ui.db.Files.Find(r.Context(), fileID)
	if err != nil || file == nil {
		return nil, err
	}

	if name := r.PathValue("name"); name != "" {
		if file.Name == "" || !strings.EqualFold(name, file.Name) {
			return nil, nil
		}
	}

	return file, nil
}

func filePath(r *http.Request, file *snips.File) string {
	if r.PathValue("name") != "" {
		return fmt.Sprintf("/f/%s/n/%s", file.ID, file.Name)
	}

	return fmt.Sprintf("/f/%s", file.ID)
}

func preferredFilePath(file *snips.File) string {
	if file.Name != "" {
		return fmt.Sprintf("/f/%s/n/%s", file.ID, file.Name)
	}
	return fmt.Sprintf("/f/%s", file.ID)
}

func (ui *UI) File(w http.ResponseWriter, r *http.Request) {
	log := logger.From(r.Context())

	file, err := ui.findFile(r)
	if err != nil {
		log.Error("unable to lookup file", "err", err)
		http.NotFound(w, r)
		return
	}

	if file == nil {
		http.NotFound(w, r)
		return
	}

	isSignedAndNotExpired := ui.signer.VerifyURLAndNotExpired(*r.URL)

	if file.Private && !isSignedAndNotExpired {
		log.Warn("attempted to access private file")
		http.NotFound(w, r)
		return
	}

	content, err := ui.db.Files.FindContent(r.Context(), file.ID)
	if err != nil {
		log.Error("unable to get file content", "err", err)
		http.Error(w, "unable to get file content", http.StatusInternalServerError)
		return
	}

	if AcceptsMarkdown(r) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Vary", "Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(FileToMarkdown(ui.cfg, file, content))
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

		signedRawURL := ui.signer.SignURL(rawPathURL)
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

	revisionCount, err := ui.db.Revisions.CountByFileID(r.Context(), file.ID)
	if err != nil {
		log.Warn("unable to count revisions", "err", err)
	}

	path := filePath(r, file)
	previewURL := fmt.Sprintf("%s://%s%s", ui.cfg.HTTP.External.Scheme, ui.cfg.HTTP.External.Host, preferredFilePath(file))
	ogImageURL := previewURL + "/og.png"
	previewName := file.ID
	if file.Name != "" {
		previewName = file.Name
	}
	ogDescription := fmt.Sprintf("%s · %s · %s · %s", previewName, strings.ToLower(file.Type), humanize.Bytes(file.Size), humanize.Time(file.UpdatedAt))

	vars := map[string]interface{}{
		"FileID":        file.ID,
		"FileName":      file.Name,
		"PreviewName":   previewName,
		"FilePath":      path,
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
		"OGURL":         previewURL,
		"OGDescription": ogDescription,
		"RevisionCount": revisionCount,
	}

	err = ui.assets.Template("file.go.html").Execute(w, vars)
	if err != nil {
		log.Error("unable to render template", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func newOG(assets Assets) *opengraph.Renderer {
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

func (ui *UI) OGImage(w http.ResponseWriter, r *http.Request) {
	log := logger.From(r.Context())

	file, err := ui.findFile(r)
	if err != nil {
		log.Error("unable to lookup file", "err", err)
		http.NotFound(w, r)
		return
	}

	if file == nil {
		http.NotFound(w, r)
		return
	}

	if file.Private && !ui.signer.VerifyURLAndNotExpired(*r.URL) {
		http.NotFound(w, r)
		return
	}

	var img bytes.Buffer
	err = ui.og.WriteImage(&img, &opengraph.FileInfo{
		ID:        file.ID,
		Name:      file.Name,
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
	_, _ = img.WriteTo(w)
}

func (ui *UI) Revisions(w http.ResponseWriter, r *http.Request) {
	log := logger.From(r.Context())

	file, err := ui.findFile(r)
	if err != nil {
		log.Error("unable to lookup file", "err", err)
		http.NotFound(w, r)
		return
	}

	if file == nil {
		http.NotFound(w, r)
		return
	}

	isSignedAndNotExpired := ui.signer.VerifyURLAndNotExpired(*r.URL)

	if file.Private && !isSignedAndNotExpired {
		log.Warn("attempted to access private file revisions")
		http.NotFound(w, r)
		return
	}

	revisions, err := ui.db.Revisions.FindByFileID(r.Context(), file.ID)
	if err != nil {
		log.Error("unable to lookup revisions", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	type revisionItem struct {
		Sequence  int64
		CreatedAt string
		Size      string
		Type      string
	}

	items := make([]revisionItem, len(revisions))
	for i, rev := range revisions {
		items[i] = revisionItem{
			Sequence:  rev.Sequence,
			CreatedAt: humanize.Time(rev.CreatedAt),
			Size:      humanize.Bytes(rev.Size),
			Type:      strings.ToLower(rev.Type),
		}
	}

	vars := map[string]interface{}{
		"FileID":       file.ID,
		"FilePath":     filePath(r, file),
		"FileSize":     humanize.Bytes(file.Size),
		"FileType":     strings.ToLower(file.Type),
		"UpdatedAt":    humanize.Time(file.UpdatedAt),
		"Private":      file.Private,
		"Revisions":    items,
		"MaxRevisions": ui.cfg.Limits.RevisionsPerFile,
		"CommitSHA":    config.BuildCommit(),
	}

	err = ui.assets.Template("revisions.go.html").Execute(w, vars)
	if err != nil {
		log.Error("unable to render template", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (ui *UI) RevisionDiff(w http.ResponseWriter, r *http.Request) {
	log := logger.From(r.Context())

	seqStr := r.PathValue("revisionID")
	if seqStr == "" {
		http.NotFound(w, r)
		return
	}

	seq, err := strconv.ParseInt(seqStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	file, err := ui.findFile(r)
	if err != nil {
		log.Error("unable to lookup file", "err", err)
		http.NotFound(w, r)
		return
	}

	if file == nil {
		http.NotFound(w, r)
		return
	}

	isSignedAndNotExpired := ui.signer.VerifyURLAndNotExpired(*r.URL)

	if file.Private && !isSignedAndNotExpired {
		log.Warn("attempted to access private file revision")
		http.NotFound(w, r)
		return
	}

	revision, err := ui.db.Revisions.FindByFileIDAndSequence(r.Context(), file.ID, seq)
	if err != nil {
		log.Error("unable to lookup revision", "err", err)
		http.NotFound(w, r)
		return
	}

	if revision == nil {
		http.NotFound(w, r)
		return
	}

	diffContent, err := ui.db.Revisions.FindDiff(r.Context(), revision.ID)
	if err != nil {
		log.Error("unable to decompress diff", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	diffLines := parseDiffLines(string(diffContent))

	vars := map[string]interface{}{
		"FileID":           file.ID,
		"FilePath":         filePath(r, file),
		"FileSize":         humanize.Bytes(file.Size),
		"FileType":         strings.ToLower(file.Type),
		"Private":          file.Private,
		"RevisionSequence": revision.Sequence,
		"CreatedAt":        humanize.Time(revision.CreatedAt),
		"RevSize":          humanize.Bytes(revision.Size),
		"RevType":          strings.ToLower(revision.Type),
		"DiffLines":        diffLines,
		"CommitSHA":        config.BuildCommit(),
	}

	err = ui.assets.Template("revision.go.html").Execute(w, vars)
	if err != nil {
		log.Error("unable to render template", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
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
	if file.Name != "" {
		fmt.Fprintf(&buf, "name: %s\n", file.Name)
	}
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

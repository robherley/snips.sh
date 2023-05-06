package http

import (
	"compress/gzip"
	"embed"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

const (
	cssPath = "web/static/css"
	cssMime = "text/css"

	jsPath = "web/static/js"
	jsMime = "application/javascript"
)

var (
	cssFiles = []string{
		"index.css",
		"code.css",
		"chroma.css",
		"markdown.css",
	}

	jsFiles = []string{
		"snips.js",
	}
)

type Assets struct {
	fs   *embed.FS
	tmpl *template.Template
	mini *minify.M
	css  []byte
	js   []byte
}

// NewAssets holds the templates, static content and minifies accordingly.
func NewAssets(fs *embed.FS) (*Assets, error) {
	assets := &Assets{
		fs:   fs,
		mini: minify.New(),
	}

	var err error

	if assets.tmpl, err = template.ParseFS(fs, "web/templates/*"); err != nil {
		return nil, err
	}

	assets.mini.AddFunc(cssMime, css.Minify)
	if assets.css, err = assets.minify(cssPath, cssFiles, cssMime); err != nil {
		return nil, err
	}

	assets.mini.AddFunc(jsMime, js.Minify)
	if assets.js, err = assets.minify(jsPath, jsFiles, jsMime); err != nil {
		return nil, err
	}

	return assets, nil
}

func (a *Assets) JS() []byte {
	return a.js
}

func (a *Assets) CSS() []byte {
	return a.css
}

func (a *Assets) Templates() *template.Template {
	return a.tmpl
}

func (a *Assets) ServeJS(w http.ResponseWriter, r *http.Request) {
	serve(w, r, a.js, jsMime)
}

func (a *Assets) ServeCSS(w http.ResponseWriter, r *http.Request) {
	serve(w, r, a.css, cssMime)
}

// serve serves the content, gzipped if the client accepts it.
func serve(w http.ResponseWriter, r *http.Request, content []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)

	hasGzip := strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip")
	if !hasGzip {
		_, _ = w.Write(content)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")
	gw := gzip.NewWriter(w)
	_, _ = gw.Write(content)
	gw.Close()
}

// minify combines all the files and minifies them.
func (a *Assets) minify(path string, files []string, mime string) ([]byte, error) {
	sb := strings.Builder{}

	for _, file := range files {
		bites, err := a.fs.ReadFile(filepath.Join(path, file))
		if err != nil {
			return nil, err
		}

		sb.Write(bites)
		sb.WriteByte('\n')
	}

	content, err := a.mini.String(mime, sb.String())
	if err != nil {
		return nil, err
	}

	return []byte(content), nil
}

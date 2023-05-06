package http

import (
	"compress/gzip"
	"embed"
	"html/template"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

const (
	tmplPattern = "web/templates/*"

	docsPath = "docs/"

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
	webFS  *embed.FS
	docsFS *embed.FS
	readme []byte
	css    []byte
	js     []byte
	tmpl   *template.Template
	mini   *minify.M
}

// NewAssets holds the templates, static content and minifies accordingly.
func NewAssets(webFS *embed.FS, docsFS *embed.FS, readme []byte) (*Assets, error) {
	assets := &Assets{
		webFS:  webFS,
		docsFS: docsFS,
		readme: readme,
	}

	assets.mini = minify.New()
	assets.mini.AddFunc(cssMime, css.Minify)
	assets.mini.AddFunc(jsMime, js.Minify)

	var err error

	if assets.tmpl, err = template.ParseFS(webFS, tmplPattern); err != nil {
		return nil, err
	}

	if assets.css, err = assets.minify(cssPath, cssFiles, cssMime); err != nil {
		return nil, err
	}

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

func (a *Assets) ReadME() []byte {
	return a.readme
}

func (a *Assets) Doc(filename string) ([]byte, error) {
	if filename == "README.md" {
		return a.readme, nil
	}

	return a.docsFS.ReadFile(path.Join(docsPath, filename))
}

func (a *Assets) Template() *template.Template {
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
		bites, err := a.webFS.ReadFile(filepath.Join(path, file))
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

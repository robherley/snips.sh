package http

import (
	"compress/gzip"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
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

//go:generate mockery --name Assets --filename assets_mock.go --inpackage --with-expecter
type Assets interface {
	Doc(filename string) ([]byte, error)
	Template() *template.Template
	ServeJS(w http.ResponseWriter, r *http.Request)
	ServeCSS(w http.ResponseWriter, r *http.Request)
}

type StaticAssets struct {
	webFS  fs.FS
	docsFS fs.FS
	readme []byte
	css    []byte
	js     []byte
	tmpl   *template.Template
	mini   *minify.M
}

func (a *StaticAssets) CSS() []byte {
	return a.css
}

func (a *StaticAssets) JS() []byte {
	return a.js
}

func (a *StaticAssets) README() []byte {
	return a.readme
}

// NewAssets holds the templates, static content and minifies accordingly.
func NewAssets(webFS fs.FS, docsFS fs.FS, readme []byte, extendHeadFile string) (*StaticAssets, error) {
	assets := &StaticAssets{
		webFS:  webFS,
		docsFS: docsFS,
		readme: readme,
	}

	assets.mini = minify.New()
	assets.mini.AddFunc(cssMime, css.Minify)
	assets.mini.AddFunc(jsMime, js.Minify)

	var (
		err               error
		extendHeadContent string
	)

	if extendHeadFile != "" {
		if headContent, err := os.ReadFile(extendHeadFile); err == nil {
			extendHeadContent = string(headContent)
		} else {
			log.Warn().Err(err).Msg("unable to extend head content")
		}
	}

	tmpl := template.New("file")
	tmpl.Funcs(template.FuncMap{
		"ExtendedHeadContent": func() template.HTML {
			return template.HTML(extendHeadContent)
		},
	})

	if assets.tmpl, err = tmpl.ParseFS(webFS, tmplPattern); err != nil {
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

func (a *StaticAssets) Doc(filename string) ([]byte, error) {
	if filename == "README.md" {
		return a.readme, nil
	}

	file, err := a.docsFS.Open(path.Join(docsPath, filename))
	if err != nil {
		return nil, err
	}

	defer file.Close()

	return io.ReadAll(file)
}

func (a *StaticAssets) Template() *template.Template {
	return a.tmpl
}

func (a *StaticAssets) ServeJS(w http.ResponseWriter, r *http.Request) {
	serve(w, r, a.js, jsMime)
}

func (a *StaticAssets) ServeCSS(w http.ResponseWriter, r *http.Request) {
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
func (a *StaticAssets) minify(path string, files []string, mime string) ([]byte, error) {
	sb := strings.Builder{}

	for _, file := range files {
		file, err := a.webFS.Open(filepath.Join(path, file))
		if err != nil {
			return nil, err
		}

		defer file.Close()

		bites, err := io.ReadAll(file)
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

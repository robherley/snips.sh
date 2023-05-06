package http

import (
	"compress/gzip"
	"embed"
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
		"file.js",
	}
)

type StaticAssets struct {
	fs   *embed.FS
	mini *minify.M
	css  []byte
	js   []byte
}

// MinififyStaticAssets minifies the static assets.
func MinififyStaticAssets(fs *embed.FS) (*StaticAssets, error) {
	s := &StaticAssets{
		fs:   fs,
		mini: minify.New(),
	}

	s.mini.AddFunc(cssMime, css.Minify)
	s.mini.AddFunc(jsMime, js.Minify)

	var err error

	s.css, err = s.minify(cssPath, cssFiles, cssMime)
	if err != nil {
		return nil, err
	}

	s.js, err = s.minify(jsPath, jsFiles, jsMime)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *StaticAssets) JS() string {
	return string(s.js)
}

func (s *StaticAssets) CSS() string {
	return string(s.css)
}

// Handler serves the static assets, gzipped.
func (s *StaticAssets) Handler(w http.ResponseWriter, r *http.Request) {
	switch filepath.Ext(r.URL.Path) {
	case ".css":
		serve(w, r, s.css, cssMime)
	case ".js":
		serve(w, r, s.js, jsMime)
	default:
		http.NotFound(w, r)
		return
	}
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
func (s *StaticAssets) minify(path string, files []string, mime string) ([]byte, error) {
	sb := strings.Builder{}

	for _, file := range files {
		bites, err := s.fs.ReadFile(filepath.Join(path, file))
		if err != nil {
			return nil, err
		}

		sb.Write(bites)
		sb.WriteByte('\n')
	}

	content, err := s.mini.String(mime, sb.String())
	if err != nil {
		return nil, err
	}

	return []byte(content), nil
}

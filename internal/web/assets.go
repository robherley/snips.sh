package web

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

const (
	tmplPattern = "web/templates/*"

	docsPath = "docs/"
	readme   = "README.md"

	cssPath = "web/static/css"
	cssMime = "text/css"

	jsPath = "web/static/js"
	jsMime = "application/javascript"
)

var (
	cssFiles = []string{
		"index.css",
		"code.css",
		"markdown.css",
	}

	jsFiles = []string{
		"snips.js",
	}
)

type Assets interface {
	Doc(filename string) ([]byte, error)
	StaticFile(name string) ([]byte, bool)
	Template() *template.Template
	Serve(w http.ResponseWriter, r *http.Request)
}

// compressedAsset holds raw and pre-compressed bytes for a bundled asset.
type compressedAsset struct {
	raw      []byte
	gzip     []byte
	zstd     []byte
	hash     string // first 8 hex chars of SHA-256
	filename string // e.g. "index.a1b2c3d4.css"
}

// staticFile holds metadata for a static file (fonts, images).
type staticFile struct {
	data []byte
	etag string // quoted ETag value
}

type StaticAssets struct {
	webFS  fs.FS
	docsFS fs.FS
	readme []byte
	css    *compressedAsset
	js     *compressedAsset
	tmpl   *template.Template
	mini   *minify.M

	// assetPaths maps logical names (e.g. "index.css") to hashed paths (e.g. "/assets/index.a1b2c3d4.css")
	assetPaths map[string]string
	// staticFiles maps asset paths (e.g. "img/favicon.png") to pre-loaded file data
	staticFiles map[string]*staticFile
}

func (a *StaticAssets) CSS() []byte {
	return a.css.raw
}

func (a *StaticAssets) JS() []byte {
	return a.js.raw
}

func (a *StaticAssets) README() []byte {
	return a.readme
}

// AssetPath returns the hashed asset path for a given logical name (e.g. "index.css" → "/assets/index.a1b2c3d4.css").
func (a *StaticAssets) AssetPath(name string) string {
	if p, ok := a.assetPaths[name]; ok {
		return p
	}
	return "/assets/" + name
}

// NewAssets holds the templates, static content and minifies accordingly.
func NewAssets(webFS fs.FS, docsFS fs.FS, readme []byte, extendHeadFile string) (*StaticAssets, error) {
	assets := &StaticAssets{
		webFS:       webFS,
		docsFS:      docsFS,
		readme:      readme,
		assetPaths:  make(map[string]string),
		staticFiles: make(map[string]*staticFile),
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
			slog.Warn("unable to extend head content", "err", err)
		}
	}

	// Build CSS and JS compressed assets
	cssRaw, err := assets.minifyFiles(cssPath, cssFiles, cssMime)
	if err != nil {
		return nil, err
	}
	if assets.css, err = newCompressedAsset(cssRaw, "index", ".css"); err != nil {
		return nil, err
	}
	assets.assetPaths["index.css"] = "/assets/" + assets.css.filename

	jsRaw, err := assets.minifyFiles(jsPath, jsFiles, jsMime)
	if err != nil {
		return nil, err
	}
	if assets.js, err = newCompressedAsset(jsRaw, "index", ".js"); err != nil {
		return nil, err
	}
	assets.assetPaths["index.js"] = "/assets/" + assets.js.filename

	// Pre-load static files (fonts, images) for ETag-based caching
	for _, dir := range []string{"fonts", "img"} {
		dirPath := path.Join("web/static", dir)
		entries, err := fs.ReadDir(webFS, dirPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			filePath := path.Join(dirPath, entry.Name())
			f, err := webFS.Open(filePath)
			if err != nil {
				continue
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				continue
			}
			h := sha256.Sum256(data)
			etag := `"` + hex.EncodeToString(h[:8]) + `"`
			assetKey := path.Join(dir, entry.Name())
			assets.staticFiles[assetKey] = &staticFile{
				data: data,
				etag: etag,
			}
		}
	}

	tmpl := template.New("file")
	tmpl.Funcs(template.FuncMap{
		"ExtendedHeadContent": func() template.HTML {
			return template.HTML(extendHeadContent)
		},
		"AssetPath": func(name string) string {
			if p, ok := assets.assetPaths[name]; ok {
				return p
			}
			return "/assets/" + name
		},
	})

	if assets.tmpl, err = tmpl.ParseFS(webFS, tmplPattern); err != nil {
		return nil, err
	}

	return assets, nil
}

func (a *StaticAssets) Doc(filename string) ([]byte, error) {
	if filename == readme {
		return a.readme, nil
	}

	file, err := a.docsFS.Open(path.Join(docsPath, filename))
	if err != nil {
		return nil, err
	}

	defer file.Close()

	return io.ReadAll(file)
}

func (a *StaticAssets) StaticFile(name string) ([]byte, bool) {
	if sf, ok := a.staticFiles[name]; ok {
		return sf.data, true
	}
	return nil, false
}

func (a *StaticAssets) Template() *template.Template {
	return a.tmpl
}

func (a *StaticAssets) Serve(w http.ResponseWriter, r *http.Request) {
	asset := r.PathValue("asset")

	switch asset {
	case a.css.filename: // Hashed CSS → immutable cache
		serveCompressed(w, r, a.css, cssMime, true)
	case a.js.filename: // Hashed JS → immutable cache
		serveCompressed(w, r, a.js, jsMime, true)
	case "index.css": // Unhashed CSS → short cache with revalidation
		serveCompressed(w, r, a.css, cssMime, false)
	case "index.js": // Unhashed JS → short cache with revalidation
		serveCompressed(w, r, a.js, jsMime, false)
	default:
		// Try static files (fonts, images)
		if sf, ok := a.staticFiles[asset]; ok {
			serveStaticFile(w, r, sf, asset)
			return
		}
		// Fallback: try to serve from webFS
		file, err := a.webFS.Open(path.Join("web/static", asset))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		contentType := mime.TypeByExtension(filepath.Ext(asset))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = io.Copy(w, file)
	}
}

// serveCompressed serves a pre-compressed asset with content negotiation.
func serveCompressed(w http.ResponseWriter, r *http.Request, ca *compressedAsset, contentType string, immutable bool) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Vary", "Accept-Encoding")

	if immutable {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=60, must-revalidate")
		w.Header().Set("ETag", `"`+ca.hash+`"`)
		if r.Header.Get("If-None-Match") == `"`+ca.hash+`"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	accept := strings.ToLower(r.Header.Get("Accept-Encoding"))

	if strings.Contains(accept, "zstd") && len(ca.zstd) > 0 {
		w.Header().Set("Content-Encoding", "zstd")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(ca.zstd)))
		_, _ = w.Write(ca.zstd)
		return
	}

	if strings.Contains(accept, "gzip") && len(ca.gzip) > 0 {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(ca.gzip)))
		_, _ = w.Write(ca.gzip)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(ca.raw)))
	_, _ = w.Write(ca.raw)
}

// serveStaticFile serves a static file (fonts, images) with ETag-based caching.
func serveStaticFile(w http.ResponseWriter, r *http.Request, sf *staticFile, assetPath string) {
	contentType := mime.TypeByExtension(filepath.Ext(assetPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Header().Set("ETag", sf.etag)

	if r.Header.Get("If-None-Match") == sf.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(sf.data)))
	_, _ = w.Write(sf.data)
}

// newCompressedAsset creates a compressedAsset from raw bytes.
func newCompressedAsset(raw []byte, baseName, ext string) (*compressedAsset, error) {
	h := sha256.Sum256(raw)
	hash := hex.EncodeToString(h[:4]) // first 8 hex chars

	// Pre-compress with gzip
	var gzBuf bytes.Buffer
	gw, _ := gzip.NewWriterLevel(&gzBuf, gzip.BestCompression)
	_, _ = gw.Write(raw)
	gw.Close()

	// Pre-compress with zstd
	var zstdBuf bytes.Buffer
	zw, err := zstd.NewWriter(&zstdBuf, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, fmt.Errorf("creating zstd writer: %w", err)
	}
	_, _ = zw.Write(raw)
	zw.Close()

	return &compressedAsset{
		raw:      raw,
		gzip:     gzBuf.Bytes(),
		zstd:     zstdBuf.Bytes(),
		hash:     hash,
		filename: fmt.Sprintf("%s.%s%s", baseName, hash, ext),
	}, nil
}

// minifyFiles combines all the files and minifies them.
func (a *StaticAssets) minifyFiles(path string, files []string, mime string) ([]byte, error) {
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

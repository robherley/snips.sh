package http

import (
	"compress/gzip"
	"io"
	"net/http"
)

var _ http.ResponseWriter = (*GZipResponseWriter)(nil)

type GZipResponseWriter struct {
	http.ResponseWriter
	io.WriteCloser
}

func (gzrw *GZipResponseWriter) Write(b []byte) (int, error) {
	return gzrw.WriteCloser.Write(b)
}

func (gzrw *GZipResponseWriter) Close() error {
	return gzrw.WriteCloser.Close()
}

func NewGZipResponseWriter(rw http.ResponseWriter) *GZipResponseWriter {
	gz := gzip.NewWriter(rw)
	return &GZipResponseWriter{
		WriteCloser:    gz,
		ResponseWriter: rw,
	}
}

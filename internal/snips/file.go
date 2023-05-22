package snips

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/signer"
)

const (
	FileTypeBinary   = "binary"
	FileTypeMarkdown = "markdown"
)

type File struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Content   []byte
	Size      uint64
	Private   bool
	Type      string
	UserID    string

	content []byte
}

func (f *File) IsBinary() bool {
	return f.Type == FileTypeBinary
}

func (f *File) IsMarkdown() bool {
	return f.Type == FileTypeMarkdown
}

func (f *File) GetSignedURL(cfg *config.Config, ttl time.Duration) (url.URL, time.Time) {
	expires := time.Now().Add(ttl).UTC()

	pathToSign := url.URL{
		Path: fmt.Sprintf("/f/%s", f.ID),
		RawQuery: url.Values{
			"exp": []string{strconv.FormatInt(expires.Unix(), 10)},
		}.Encode(),
	}

	signedFileURL := signer.New(cfg.HMACKey).SignURL(pathToSign)
	signedFileURL.Scheme = cfg.HTTP.External.Scheme
	signedFileURL.Host = cfg.HTTP.External.Host

	return signedFileURL, expires
}

func (f *File) Visibility() string {
	if f.Private {
		return "private"
	}

	return "public"
}

var decoder, _ = zstd.NewReader(nil)

func (f *File) GetContent() ([]byte, error) {
	// check if it has been compressed
	if !f.isCompressed() {
		return f.content, nil
	}

	defer decoder.Close()
	decodedBytes, err := decoder.DecodeAll(f.content, nil)

	return decodedBytes, err
}

var encoder, _ = zstd.NewWriter(nil)

func (f *File) SetContent(in []byte, compress bool) {
	if compress {
		f.content = encoder.EncodeAll(in, nil)
	} else {
		f.content = in
	}
}

func (f *File) isCompressed() bool {
	// check if first 4 bytes are ZSTD magic number
	// https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md#zstandard-frames
	return len(data) > 4 && binary.BigEndian.Uint32(data) == 0x28B52FFD
}

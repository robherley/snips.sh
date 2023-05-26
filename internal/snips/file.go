package snips

import (
	"encoding/binary"
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
	ID         string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Size       uint64
	RawContent []byte
	Private    bool
	Type       string
	UserID     string
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

func (f *File) GetContent() ([]byte, error) {
	if !f.isCompressed() {
		return f.RawContent, nil
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}

	defer decoder.Close()
	decodedBytes, err := decoder.DecodeAll(f.RawContent, nil)

	return decodedBytes, err
}

func (f *File) SetContent(in []byte, compress bool) error {
	if !compress {
		f.RawContent = in
		return nil
	}

	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return err
	}

	f.RawContent = encoder.EncodeAll(in, nil)
	return encoder.Close()
}

func (f *File) isCompressed() bool {
	// check if first 4 bytes are ZSTD magic number
	// https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md#zstandard-frames
	return len(f.RawContent) > 4 && binary.BigEndian.Uint32(f.RawContent) == 0x28B52FFD
}

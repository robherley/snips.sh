package snips

import (
	"fmt"
	"net/url"
	"time"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/signer"
)

const (
	FileTypeBinary   = "binary"
	FileTypeMarkdown = "markdown"
)

type File struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Size      uint64    `json:"size"`
	Private   bool      `json:"private"`
	Type      string    `json:"type"`
	UserID    string    `json:"-"`
	Name      string    `json:"name,omitempty"`
}

func (f *File) DisplayName() string {
	if f.Name != "" {
		return fmt.Sprintf("%s (%s)", f.Name, f.ID)
	}
	return f.ID
}

func (f *File) IsBinary() bool {
	return f.Type == FileTypeBinary
}

func (f *File) IsMarkdown() bool {
	return f.Type == FileTypeMarkdown
}

func (f *File) GetSignedURL(cfg *config.Config, ttl time.Duration) (url.URL, time.Time) {
	pathToSign := url.URL{
		Path: fmt.Sprintf("/f/%s", f.ID),
	}

	signedFileURL, expires := signer.New(cfg.HMACKey).SignURLWithTTL(pathToSign, ttl)
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

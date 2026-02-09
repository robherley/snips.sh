package testutil

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/robherley/snips.sh/internal/web"
)

// Assets returns the _real_ assets used to render the web server.
// Instead of being embedded normally, this will hacky load the assets at runtime.
func Assets(t *testing.T) web.Assets {
	t.Helper()

	_, filename, _, _ := runtime.Caller(0)

	root := filepath.Join(filepath.Dir(filename), "..", "..")

	rootFS := os.DirFS(root)

	readmeFile, err := rootFS.Open("README.md")
	if err != nil {
		t.Fatal(err)
	}

	readmeBytes, err := io.ReadAll(readmeFile)
	if err != nil {
		t.Fatal(err)
	}

	assets, err := web.NewAssets(
		rootFS, // webFS
		rootFS, // docFS
		readmeBytes,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	return assets
}

package testutil

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/robherley/snips.sh/internal/http"
)

// Assets returns the _real_ assets used to render the web server.
// Instead of being embeded normally, this will hacky load the assets at runtime.
func Assets(t *testing.T) http.Assets {
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

	assets, err := http.NewAssets(
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

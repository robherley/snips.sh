package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempFile creates a temporary file with the given filename and content.
// The file's directory is deleted when the test finishes.
func TempFile(t *testing.T, filename, content string) string {
	t.Helper()

	f, err := os.Create(filepath.Join(t.TempDir(), filename))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}

	return f.Name()
}

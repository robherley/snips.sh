package renderer_test

import (
	"testing"

	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestDetectFileType(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
		hint    string
		want    string
	}{
		{
			name:    "binary content",
			content: []byte{0x00, 0x01, 0x02, 0x03},
			hint:    "go",
			want:    "binary",
		},
		{
			name:    "hint used as-is",
			content: []byte("package main"),
			hint:    "go",
			want:    "go",
		},
		{
			name:    "hint is normalized: leading dot, case, whitespace",
			content: []byte("package main"),
			hint:    " .Go ",
			want:    "go",
		},
		{
			name:    "hint of only a dot falls back to detection",
			content: []byte("plain words with no obvious language"),
			hint:    ".",
			want:    "plaintext",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, renderer.DetectFileType(tc.content, tc.hint, false))
		})
	}
}

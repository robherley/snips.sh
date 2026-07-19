package snips_test

import (
	"strings"
	"testing"

	"github.com/robherley/snips.sh/internal/snips"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeName(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  string
		err   error
	}{
		{
			name:  "simple",
			input: "notes",
			want:  "notes",
		},
		{
			name:  "casing preserved",
			input: "DeployNotes",
			want:  "DeployNotes",
		},
		{
			name:  "hyphen separators",
			input: "deploy-notes-2024",
			want:  "deploy-notes-2024",
		},
		{
			name:  "dot separators",
			input: "main.go",
			want:  "main.go",
		},
		{
			name:  "underscore separators",
			input: "my_notes",
			want:  "my_notes",
		},
		{
			name:  "mixed separators and casing",
			input: "Deploy-Notes.v2_final",
			want:  "Deploy-Notes.v2_final",
		},
		{
			name:  "leading underscore",
			input: "_private",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "leading dot",
			input: ".hidden",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "trailing dot",
			input: "notes.",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "consecutive dots",
			input: "a..b",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "dot path segment",
			input: "..",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "surrounding whitespace",
			input: "  notes  ",
			want:  "notes",
		},
		{
			name:  "empty",
			input: "",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "whitespace only",
			input: "   ",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "leading hyphen",
			input: "-notes",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "trailing hyphen",
			input: "notes-",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "consecutive hyphens",
			input: "deploy--notes",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "invalid characters",
			input: "deploy_notes!",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "interior whitespace",
			input: "deploy notes",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "path traversal",
			input: "../secrets",
			err:   snips.ErrInvalidName,
		},
		{
			name:  "too long",
			input: strings.Repeat("a", snips.NameMaxLength+1),
			err:   snips.ErrInvalidName,
		},
		{
			name:  "max length",
			input: strings.Repeat("a", snips.NameMaxLength),
			want:  strings.Repeat("a", snips.NameMaxLength),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := snips.NormalizeName(tc.input)

			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

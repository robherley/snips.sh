package snips

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileContent(t *testing.T) {
	testcases := []struct {
		name       string
		in         []byte
		want       []byte
		compressed bool
		err        error
	}{
		{
			name:       "Compressed content",
			in:         []byte("Hello World"),
			want:       []byte("Hello World"),
			compressed: true,
			err:        nil,
		},
		{
			name:       "Uncompressed content",
			in:         []byte("Hello World"),
			want:       []byte("Hello World"),
			compressed: false,
			err:        nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var f File
			if tc.compressed {
				f.SetContent(tc.in)
			} else {
				f.content = tc.in
			}

			got, err := f.GetContent()
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.want, got)

		})
	}
}

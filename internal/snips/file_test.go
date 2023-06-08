package snips

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileContent(t *testing.T) {
	testcases := []struct {
		name       string
		in         []byte
		want       map[string]any
		compressed bool
		err        error
	}{
		{
			name: "Compressed content",
			in:   []byte("Hello World"),
			want: map[string]any{
				"getContent": []byte("Hello World"),
				"rawContent": []byte{40, 181, 47, 253, 4, 0, 89, 0, 0, 72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100, 194, 91, 36, 25},
			},
			compressed: true,
			err:        nil,
		},
		{
			name: "Uncompressed content",
			in:   []byte("Hello World"),
			want: map[string]any{
				"getContent": []byte("Hello World"),
				"rawContent": []byte("Hello World"),
			},
			compressed: false,
			err:        nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var f File
			err := f.SetContent(tc.in, tc.compressed)
			assert.NoError(t, err)

			gotContent, err := f.GetContent()
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.want["getContent"], gotContent)
			assert.Equal(t, tc.want["rawContent"], f.RawContent)

		})
	}
}

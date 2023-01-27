package ssh_test

import (
	"io"
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestUploadFlags(t *testing.T) {
	testcases := []struct {
		name string
		args []string
		want ssh.UploadFlags
		err  error
	}{
		{
			name: "no flags",
			args: []string{},
			want: ssh.UploadFlags{
				Private:   false,
				Extension: "",
			},
		},
		{
			name: "private",
			args: []string{"-private"},
			want: ssh.UploadFlags{
				Private:   true,
				Extension: "",
			},
		},
		{
			name: "extension",
			args: []string{"-ext", "txt"},
			want: ssh.UploadFlags{
				Private:   false,
				Extension: "txt",
			},
		},
		{
			name: "private and extension",
			args: []string{"-private", "-ext", "txt"},
			want: ssh.UploadFlags{
				Private:   true,
				Extension: "txt",
			},
		},
		{
			name: "trims leading dot and lowercases",
			args: []string{"-ext", ".tXt"},
			want: ssh.UploadFlags{
				Private:   false,
				Extension: "txt",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var got ssh.UploadFlags
			err := got.Parse(io.Discard, tc.args)

			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.Equal(t, tc.want.Extension, got.Extension)
				assert.Equal(t, tc.want.Private, got.Private)
			}
		})
	}
}

func TestSignFlags(t *testing.T) {
	testcases := []struct {
		name string
		args []string
		want ssh.SignFlags
		err  error
	}{
		{
			name: "no flags",
			args: []string{},
			err:  ssh.ErrFlagRequied,
		},
		{
			name: "ttl",
			args: []string{"-ttl", "1h"},
			want: ssh.SignFlags{
				TTL: 1 * time.Hour,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var got ssh.SignFlags
			err := got.Parse(io.Discard, tc.args)

			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.Equal(t, tc.want.TTL, got.TTL)
			}
		})
	}
}

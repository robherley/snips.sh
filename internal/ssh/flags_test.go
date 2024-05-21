package ssh_test

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/robherley/snips.sh/internal/ssh"
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
				TTL:       time.Duration(0),
			},
		},
		{
			name: "private",
			args: []string{"-private"},
			want: ssh.UploadFlags{
				Private:   true,
				Extension: "",
				TTL:       time.Duration(0),
			},
		},
		{
			name: "extension",
			args: []string{"-ext", "txt"},
			want: ssh.UploadFlags{
				Private:   false,
				Extension: "txt",
				TTL:       time.Duration(0),
			},
		},
		{
			name: "private and extension",
			args: []string{"-private", "-ext", "txt"},
			want: ssh.UploadFlags{
				Private:   true,
				Extension: "txt",
				TTL:       time.Duration(0),
			},
		},
		{
			name: "trims leading dot and lowercases",
			args: []string{"-ext", ".tXt"},
			want: ssh.UploadFlags{
				Private:   false,
				Extension: "txt",
				TTL:       time.Duration(0),
			},
		},
		{
			name: "private and ttl",
			args: []string{"-private", "-ttl", "1w2d3m4s"},
			want: ssh.UploadFlags{
				Private:   true,
				Extension: "",
				TTL:       1*7*24*time.Hour + 2*24*time.Hour + 3*time.Minute + 4*time.Second,
			},
		},
		{
			name: "ttl only",
			args: []string{"-ttl", "30s"},
			want: ssh.UploadFlags{
				Private:   false,
				Extension: "",
				TTL:       30 * time.Second,
			},
			err: ssh.ErrFlagRequied,
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
			args: []string{"-ttl", "1w2d3m4s"},
			want: ssh.SignFlags{
				TTL: 1*7*24*time.Hour + 2*24*time.Hour + 3*time.Minute + 4*time.Second,
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

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
				TTL:       0,
			},
		},
		{
			name: "private",
			args: []string{"-private"},
			want: ssh.UploadFlags{
				Private:   true,
				Extension: "",
				TTL:       0,
			},
		},
		{
			name: "extension",
			args: []string{"-ext", "txt"},
			want: ssh.UploadFlags{
				Private:   false,
				Extension: "txt",
				TTL:       0,
			},
		},
		{
			name: "private and extension",
			args: []string{"-private", "-ext", "txt"},
			want: ssh.UploadFlags{
				Private:   true,
				Extension: "txt",
				TTL:       0,
			},
		},
		{
			name: "trims leading dot and lowercases",
			args: []string{"-ext", ".tXt"},
			want: ssh.UploadFlags{
				Private:   false,
				Extension: "txt",
				TTL:       0,
			},
		},
		{
			name: "private and ttl",
			args: []string{"-private", "-ttl", "30s"},
			want: ssh.UploadFlags{
				Private:   true,
				Extension: "",
				TTL:       time.Duration(30),
			},
		},
		{
			name: "ttl only",
			args: []string{"-ttl", "30s"},
			want: ssh.UploadFlags{
				Private:   true,
				Extension: "",
				TTL:       30,
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

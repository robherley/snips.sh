package signer_test

import (
	"net/url"
	"testing"

	"github.com/robherley/snips.sh/internal/signer"
)

var (
	testKey = "foobar"
)

func parseURL(path string) url.URL {
	u, _ := url.Parse(path)
	return *u
}

func TestSigner_SignURL(t *testing.T) {
	signer := signer.New(testKey)

	testcases := []struct {
		name string
		url  url.URL
		want url.URL
	}{
		{
			name: "no query parameters",
			url:  parseURL("https://snips.sh/f/5yiAwU0Ax"),
			want: parseURL("https://snips.sh/f/5yiAwU0Ax?sig=bA27c2irHRcEevqsi2ceGYVJ6GYxUeAe_0kTb8-cDLY%3D"),
		},
		{
			name: "with query parameters",
			url:  parseURL("https://snips.sh/f/5yiAwU0Ax?a=1&b=2&c=3"),
			want: parseURL("https://snips.sh/f/5yiAwU0Ax?a=1&b=2&c=3&sig=B_6rhUVsDwirco14MFZp_7lJWA8CADQyQqk7Qb2uCwM%3D"),
		},
		{
			name: "with query parameters out of order",
			url:  parseURL("https://snips.sh/f/5yiAwU0Ax?c=3&b=2&a=1"),
			want: parseURL("https://snips.sh/f/5yiAwU0Ax?a=1&b=2&c=3&sig=B_6rhUVsDwirco14MFZp_7lJWA8CADQyQqk7Qb2uCwM%3D"),
		},
		{
			name: "just path",
			url:  parseURL("/f/5yiAwU0Ax"),
			want: parseURL("/f/5yiAwU0Ax?sig=JBKBf9df4nXmm5tgmIacQ9ASE6ZyJlnirW46R5Hzm5Y%3D"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := signer.SignURL(tc.url)

			if got.String() != tc.want.String() {
				t.Errorf("got %s, want %s", got.String(), tc.want.String())
			}
		})
	}
}

func TestSigner_VerifyURL(t *testing.T) {
	signer := signer.New(testKey)

	testcases := []struct {
		name string
		url  url.URL
		want bool
	}{
		{
			name: "valid - no query parameters",
			url:  parseURL("https://snips.sh/f/5yiAwU0Ax?sig=bA27c2irHRcEevqsi2ceGYVJ6GYxUeAe_0kTb8-cDLY%3D"),
			want: true,
		},
		{
			name: "valid - with query parameters",
			url:  parseURL("https://snips.sh/f/5yiAwU0Ax?a=1&b=2&c=3&sig=B_6rhUVsDwirco14MFZp_7lJWA8CADQyQqk7Qb2uCwM%3D"),
			want: true,
		},
		{
			name: "valid - with query parameters out of order",
			url:  parseURL("https://snips.sh/f/5yiAwU0Ax?a=1&b=2&c=3&sig=B_6rhUVsDwirco14MFZp_7lJWA8CADQyQqk7Qb2uCwM%3D"),
			want: true,
		},
		{
			name: "valid - just path",
			url:  parseURL("/f/5yiAwU0Ax?sig=JBKBf9df4nXmm5tgmIacQ9ASE6ZyJlnirW46R5Hzm5Y%3D"),
			want: true,
		},
		{
			name: "invalid - no signature",
			url:  parseURL("https://snips.sh/f/5yiAwU0Ax"),
			want: false,
		},
		{
			name: "invalid - invalid signature",
			url:  parseURL("https://snips.sh/f/5yiAwU0Ax?sig=invalid"),
			want: false,
		},
		{
			name: "invalid - mutated path",
			url:  parseURL("https://notsnips.sh/f/5yiAwU0Ax?sig=bA27c2irHRcEevqsi2ceGYVJ6GYxUeAe_0kTb8-cDLY%3D"),
			want: false,
		},
		{
			name: "invalid - mutated query parameters",
			url:  parseURL("https://snips.sh/f/5yiAwU0Ax?a=2&b=2&c=3&sig=B_6rhUVsDwirco14MFZp_7lJWA8CADQyQqk7Qb2uCwM%3D"),
			want: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := signer.VerifyURL(tc.url)

			if got != tc.want {
				t.Errorf("got %t, want %t", got, tc.want)
			}
		})
	}
}

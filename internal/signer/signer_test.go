package signer_test

import (
	"net/url"
	"strconv"
	"testing"
	"time"

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

func TestSigner_SignURLWithTTL(t *testing.T) {
	signer := signer.New(testKey)

	url := parseURL("https://snips.sh/f/5yiAwU0Ax")
	ttl := 5 * time.Minute

	now := time.Now().UTC()
	got, _ := signer.SignURLWithTTL(url, ttl)

	expires := got.Query().Get("exp")
	if expires == "" {
		t.Errorf("expected exp to be set")
	}

	expiresUnix, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		t.Errorf("expected exp to be an integer: %s", err.Error())
	}

	expiresTime := time.Unix(expiresUnix, 0)

	skew := 10 * time.Second
	if expiresTime.Before(now.Add(ttl-skew)) || expiresTime.After(now.Add(ttl+skew)) {
		t.Errorf("expected exp to be within 10 seconds of %s, got %s", now.Add(ttl).String(), expiresTime.String())
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

func TestSigner_VerifyURLAndNotExpired(t *testing.T) {
	signer := signer.New(testKey)

	testcases := []struct {
		name string
		url  func() url.URL
		ttl  time.Duration
		want bool
	}{
		{
			name: "valid - with exp param",
			url: func() url.URL {
				url, _ := signer.SignURLWithTTL(parseURL("https://snips.sh/f/5yiAwU0Ax"), 5*time.Minute)
				return url
			},
			want: true,
		},
		{
			name: "valid - with multiple parameters",
			url: func() url.URL {
				url, _ := signer.SignURLWithTTL(parseURL("https://snips.sh/f/5yiAwU0Ax?r=1"), 5*time.Minute)
				return url
			},
			want: true,
		},
		{
			name: "invalid - no params",
			url: func() url.URL {
				return parseURL("https://snips.sh/f/5yiAwU0Ax")
			},
			want: false,
		},
		{
			name: "invalid - expired",
			url: func() url.URL {
				url, _ := signer.SignURLWithTTL(parseURL("https://snips.sh/f/5yiAwU0Ax"), -5*time.Minute)
				return url
			},
			want: false,
		},
		{
			name: "invalid - expired with query parameters",
			url: func() url.URL {
				url, _ := signer.SignURLWithTTL(parseURL("https://snips.sh/f/5yiAwU0Ax?r=1"), -5*time.Minute)
				return url
			},
			want: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := signer.VerifyURLAndNotExpired(tc.url())

			if got != tc.want {
				t.Errorf("got %t, want %t", got, tc.want)
			}
		})
	}
}

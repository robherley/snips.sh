package timeutil_test

import (
	"testing"
	"time"

	"github.com/robherley/snips.sh/internal/timeutil"
)

func TestParseDuration(t *testing.T) {
	cases := []struct {
		duration string
		want     time.Duration
	}{
		{"1d", 24 * time.Hour},
		{"1d12h30m15s", 24*time.Hour + 12*time.Hour + 30*time.Minute + 15*time.Second},
		{"1w", 7 * 24 * time.Hour},
		{"1w1d12h", 8*24*time.Hour + 12*time.Hour},
	}

	for _, tc := range cases {
		t.Run(tc.duration, func(t *testing.T) {
			got, err := timeutil.ParseDuration(tc.duration)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

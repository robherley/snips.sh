package config_test

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/testutil"
)

// captureHandler records slog records for assertion in tests.
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *captureHandler) hasWarn(substr string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if r.Level == slog.LevelWarn && strings.Contains(r.Message, substr) {
			return true
		}
	}
	return false
}

func withCaptureLogger(t *testing.T) *captureHandler {
	t.Helper()
	h := &captureHandler{}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return h
}

func TestLoad_DefaultHMACKeyWarning(t *testing.T) {
	t.Run("warns when default key is used", func(t *testing.T) {
		h := withCaptureLogger(t)

		if _, err := config.Load(); err != nil {
			t.Fatal(err)
		}

		if !h.hasWarn("SNIPS_HMACKEY") {
			t.Error("expected a warning about the default HMAC key, got none")
		}
	})

	t.Run("no warning when custom key is set", func(t *testing.T) {
		t.Setenv("SNIPS_HMACKEY", "a-custom-secret-key-that-is-not-the-default")
		h := withCaptureLogger(t)

		if _, err := config.Load(); err != nil {
			t.Fatal(err)
		}

		if h.hasWarn("SNIPS_HMACKEY") {
			t.Error("unexpected warning about HMAC key when a custom key is set")
		}
	})
}

func TestConfig_SSHAuthorizedKeys(t *testing.T) {
	t.Run("no keys", func(t *testing.T) {
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}

		authorizedKeys, err := cfg.SSHAuthorizedKeys()
		if err != nil {
			t.Fatal(err)
		}

		if len(authorizedKeys) != 0 {
			t.Fatalf("expected 0 keys, got %d", len(authorizedKeys))
		}
	})

	t.Run("contains invalid key", func(t *testing.T) {
		authorizedKeysFile := testutil.TempFile(t, "authorized_keys", `
		ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEnqsMuqOhEVw3HyWMp2fqqn6l1IZtJHD1UWkOXszUcl
		this is not an authorized key 🦝
		ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKrOJrpYRgEiuGuoNhyPbeEldjIRkwRG/fjjySPUks/y
		`)

		t.Setenv("SNIPS_SSH_AUTHORIZEDKEYSPATH", authorizedKeysFile)
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}

		authorizedKeys, err := cfg.SSHAuthorizedKeys()
		if err != nil {
			t.Fatal(err)
		}

		if len(authorizedKeys) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(authorizedKeys))
		}
	})

	t.Run("valid keys", func(t *testing.T) {
		authorizedKeysFile := testutil.TempFile(t, "authorized_keys", `
		ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEnqsMuqOhEVw3HyWMp2fqqn6l1IZtJHD1UWkOXszUcl
		ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBBMu3TbOgxpvYrcQQG6VHSgrwMzAsFg2s+UX5JMNjNI
		ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKrOJrpYRgEiuGuoNhyPbeEldjIRkwRG/fjjySPUks/y
		`)

		t.Setenv("SNIPS_SSH_AUTHORIZEDKEYSPATH", authorizedKeysFile)
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}

		authorizedKeys, err := cfg.SSHAuthorizedKeys()
		if err != nil {
			t.Fatal(err)
		}

		if len(authorizedKeys) != 3 {
			t.Fatalf("expected 3 keys, got %d", len(authorizedKeys))
		}

		for i, key := range authorizedKeys {
			if key.Type() != "ssh-ed25519" {
				t.Fatalf("key %d has wrong type: %s", i, key.Type())
			}

			if key.Marshal() == nil {
				t.Fatalf("key %d is empty", i)
			}
		}
	})
}

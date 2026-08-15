package config_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/testutil"
)

func TestLoad_EphemeralHMACKeyWarning(t *testing.T) {
	t.Run("warns and generates ephemeral key when unset", func(t *testing.T) {
		recorder := testutil.SetLogRecorder(t)

		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}

		if cfg.HMACKey == "" {
			t.Error("expected generated HMAC key, got empty string")
		}

		recorder.AssertLog(t, slog.LevelWarn, "SNIPS_HMACKEY")
	})

	t.Run("no warning when custom key is set", func(t *testing.T) {
		t.Setenv("SNIPS_HMACKEY", "a-custom-secret-key-that-is-not-the-default")
		recorder := testutil.SetLogRecorder(t)

		if _, err := config.Load(); err != nil {
			t.Fatal(err)
		}

		recorder.RefuteLog(t, slog.LevelWarn, "SNIPS_HMACKEY")
	})
}

func TestConfig_DatabaseURL(t *testing.T) {
	t.Run("legacy filepath warns and is used as fallback", func(t *testing.T) {
		t.Setenv("SNIPS_DB_URL", "temporarily-set-so-it-can-be-unset")
		if err := os.Unsetenv("SNIPS_DB_URL"); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SNIPS_DB_FILEPATH", "legacy.db")
		recorder := testutil.SetLogRecorder(t)

		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DB.URL != "legacy.db" {
			t.Fatalf("DB.URL = %q, want legacy.db", cfg.DB.URL)
		}
		recorder.AssertLog(t, slog.LevelWarn, "SNIPS_DB_FILEPATH is deprecated")
	})

	t.Run("url takes precedence over legacy filepath", func(t *testing.T) {
		t.Setenv("SNIPS_DB_URL", "current.db")
		t.Setenv("SNIPS_DB_FILEPATH", "legacy.db")

		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DB.URL != "current.db" {
			t.Fatalf("DB.URL = %q, want current.db", cfg.DB.URL)
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

	t.Run("keys from environment", func(t *testing.T) {
		t.Setenv("SNIPS_SSH_AUTHORIZEDKEYS", `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEnqsMuqOhEVw3HyWMp2fqqn6l1IZtJHD1UWkOXszUcl
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKrOJrpYRgEiuGuoNhyPbeEldjIRkwRG/fjjySPUks/y`)

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

	t.Run("environment keys take precedence over path", func(t *testing.T) {
		t.Setenv("SNIPS_SSH_AUTHORIZEDKEYS", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEnqsMuqOhEVw3HyWMp2fqqn6l1IZtJHD1UWkOXszUcl")
		t.Setenv("SNIPS_SSH_AUTHORIZEDKEYSPATH", "does-not-exist")

		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		authorizedKeys, err := cfg.SSHAuthorizedKeys()
		if err != nil {
			t.Fatal(err)
		}
		if len(authorizedKeys) != 1 {
			t.Fatalf("expected 1 key, got %d", len(authorizedKeys))
		}
	})
}

func TestConfig_SSHHostKey(t *testing.T) {
	t.Setenv("SNIPS_SSH_HOSTKEY", "private-key-content")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SSH.HostKey != "private-key-content" {
		t.Fatalf("SSH.HostKey = %q, want private-key-content", cfg.SSH.HostKey)
	}
}

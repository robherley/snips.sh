package config_test

import (
	"testing"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/testutil"
)

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

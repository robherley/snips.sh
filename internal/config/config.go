package config

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime/debug"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/kelseyhightower/envconfig"
)

var (
	BuildCommit = sync.OnceValue(readBuildCommit)
)

const (
	ApplicationName = "snips"
	UsageFormat     = `
KEY	TYPE	DEFAULT	DESCRIPTION
{{range .}}{{usage_key .}}	{{usage_type .}}	{{usage_default .}}	{{usage_description .}}
{{end}}`
)

type Config struct {
	Debug bool `default:"False" desc:"enable debug logging and pprof"`

	EnableGuesser bool `default:"True" desc:"enable AI model to detect file types"`

	HMACKey string `desc:"symmetric key used to sign URLs"`

	FileCompression bool `default:"True" desc:"enable compression of file contents"`

	Limits struct {
		FileSize         uint64        `default:"1048576" desc:"maximum file size in bytes"`
		FilesPerUser     uint64        `default:"100" desc:"maximum number of files per user"`
		SessionDuration  time.Duration `default:"15m" desc:"maximum ssh session duration"`
		RevisionsPerFile uint64        `default:"64" desc:"maximum number of revisions per file"`
		APIKeysPerUser   uint64        `default:"16" desc:"maximum number of api keys per user"`
	}

	DB struct {
		URL string `default:"data/snips.db" desc:"database URL or DSN"`
	}

	HTTP struct {
		Internal url.URL `default:"http://localhost:8080" desc:"internal address to listen for http requests"`
		External url.URL `default:"http://localhost:8080" desc:"external http address displayed in commands"`
	}

	HTML struct {
		ExtendHeadFile string `default:"" desc:"path to html file for extra content in <head>"`
	}

	SSH struct {
		Internal           url.URL `default:"ssh://localhost:2222" desc:"internal address to listen for ssh requests"`
		External           url.URL `default:"ssh://localhost:2222" desc:"external ssh address displayed in commands"`
		HostKey            string  `default:"" desc:"PEM-encoded SSH host private key; takes precedence over host key path"`
		HostKeyPath        string  `default:"data/keys/snips" desc:"path to host keys (without extension)"`
		AuthorizedKeys     string  `default:"" desc:"authorized keys content; takes precedence over authorized keys path"`
		AuthorizedKeysPath string  `default:"" desc:"path to authorized keys, if specified will restrict SSH access"`
	}

	Metrics struct {
		Statsd       *url.URL `desc:"statsd server address (e.g. udp://localhost:8125)"`
		UseDogStatsd bool     `default:"False" desc:"use dogstatsd instead of statsd"`
	}
}

func (cfg *Config) PrintUsage() error {
	tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 2, ' ', 0)
	defer tabs.Flush()

	return envconfig.Usagef(ApplicationName, cfg, tabs, UsageFormat)
}

func (cfg *Config) HTTPAddressForFile(fileID string) string {
	httpAddr := cfg.HTTP.External
	httpAddr.Path = fmt.Sprintf("/f/%s", fileID)

	return httpAddr.String()
}

func (cfg *Config) HTTPAddressForNamedFile(fileID, name string) string {
	httpAddr := cfg.HTTP.External
	httpAddr.Path = fmt.Sprintf("/f/%s/n/%s", fileID, name)

	return httpAddr.String()
}

func (cfg *Config) SSHCommandForFile(fileID string) string {
	return cfg.sshCommandFor("f:" + fileID)
}

func (cfg *Config) SSHCommandForNamedFile(name string) string {
	return cfg.sshCommandFor("n:" + name)
}

func (cfg *Config) sshCommandFor(user string) string {
	sshCommand := fmt.Sprintf("ssh %s@%s", user, cfg.SSH.External.Hostname())
	if sshPort := cfg.SSH.External.Port(); sshPort != "" && sshPort != "22" {
		sshCommand += fmt.Sprintf(" -p %s", sshPort)
	}

	return sshCommand
}

// SSHAuthorizedKeys returns the configured authorized keys.
func (cfg *Config) SSHAuthorizedKeys() ([]ssh.PublicKey, error) {
	authorizedKeys := make([]ssh.PublicKey, 0)
	authorizedKeysContent := []byte(cfg.SSH.AuthorizedKeys)
	configured := cfg.SSH.AuthorizedKeys != "" || cfg.SSH.AuthorizedKeysPath != ""

	if len(authorizedKeysContent) == 0 {
		if cfg.SSH.AuthorizedKeysPath == "" {
			return authorizedKeys, nil
		}

		var err error
		authorizedKeysContent, err = os.ReadFile(cfg.SSH.AuthorizedKeysPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read authorized keys file: %w", err)
		}
	}

	for i, keyBytes := range bytes.Split(authorizedKeysContent, []byte("\n")) {
		if len(bytes.TrimSpace(keyBytes)) == 0 {
			continue
		}

		out, _, _, _, err := ssh.ParseAuthorizedKey(keyBytes)
		if err != nil {
			slog.Warn("unable to parse authorized key", "line", i, "err", err)
			continue
		}

		authorizedKeys = append(authorizedKeys, out)
	}

	if configured && len(authorizedKeys) == 0 {
		return nil, fmt.Errorf("authorized keys were configured but no valid keys were found")
	}

	return authorizedKeys, nil
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := envconfig.Process(ApplicationName, cfg); err != nil {
		return nil, err
	}

	if filePath, ok := os.LookupEnv("SNIPS_DB_FILEPATH"); ok {
		slog.Warn("SNIPS_DB_FILEPATH is deprecated; use SNIPS_DB_URL instead")
		if _, hasURL := os.LookupEnv("SNIPS_DB_URL"); !hasURL {
			cfg.DB.URL = filePath
		}
	}

	cfg.EnableGuesser = cfg.EnableGuesser && GuessingSupported

	if cfg.HMACKey == "" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("generate ephemeral HMAC key: %w", err)
		}
		cfg.HMACKey = base64.RawURLEncoding.EncodeToString(key)
		slog.Warn("SNIPS_HMACKEY is unset; generated an ephemeral HMAC key — signed URLs will be invalid after restart; set a strong secret before exposing this instance publicly")
	}

	return cfg, nil
}

func readBuildCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return "unknown"
}

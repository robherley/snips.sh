package config

import (
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kelseyhightower/envconfig"
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

	EnableGuesser bool `default:"True" desc:"enable guesslang model to detect file types"`

	HMACKey string `default:"hmac-and-cheese" desc:"symmetric key used to sign URLs"`

	Limits struct {
		FileSize        uint64        `default:"1048576" desc:"maximum file size in bytes"`
		FilesPerUser    uint64        `default:"100" desc:"maximum number of files per user"`
		SessionDuration time.Duration `default:"15m" desc:"maximum ssh session duration"`
	}

	DB struct {
		FilePath string `default:"data/snips.db" desc:"path to database file"`
	}

	HTTP struct {
		Internal url.URL `default:"http://localhost:8080" desc:"internal address to listen for http requests"`
		External url.URL `default:"http://localhost:8080" desc:"external http address displayed in commands"`
	}

	HTML struct {
		ExtendHeadFile string `default:"" desc:"path to html file for extra content in <head>"`
	}

	SSH struct {
		Internal    url.URL `default:"ssh://localhost:2222" desc:"internal address to listen for ssh requests"`
		External    url.URL `default:"ssh://localhost:2222" desc:"external ssh address displayed in commands"`
		HostKeyPath string  `default:"data/keys/snips" desc:"path to host keys (without extension)"`
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

func (cfg *Config) SSHCommandForFile(fileID string) string {
	sshCommand := fmt.Sprintf("ssh f:%s@%s", fileID, cfg.SSH.External.Hostname())
	if sshPort := cfg.SSH.External.Port(); sshPort != "" && sshPort != "22" {
		sshCommand += fmt.Sprintf(" -p %s", sshPort)
	}

	return sshCommand
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := envconfig.Process(ApplicationName, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

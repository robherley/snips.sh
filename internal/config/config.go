package config

import (
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"

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
	Debug bool `default:"false" desc:"enable debug logging"`

	HMACKey string `default:"correct-horse-battery-staple" desc:"symmetric key used to sign URLs"`

	DB struct {
		FilePath string `default:"data/snips.db" desc:"path to database file"`
	}

	HTTP struct {
		Internal url.URL `default:"http://localhost:8080" desc:"internal address to listen for http requests"`
		External url.URL `default:"http://localhost:8080" desc:"external http address displayed in commands"`
	}

	SSH struct {
		Internal    url.URL `default:"ssh://localhost:2222" desc:"internal address to listen for ssh requests"`
		External    url.URL `default:"ssh://localhost:2222" desc:"external ssh address displayed in commands"`
		HostKeyPath string  `default:"data/keys/snips" desc:"path to host keys (without extension)"`
	}
}

func (cfg *Config) PrintUsage() error {
	tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
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
	if sshPort := cfg.SSH.External.Port(); sshPort != "22" {
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

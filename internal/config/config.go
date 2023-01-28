package config

import (
	"net"
	"net/url"
	"strconv"

	"github.com/kelseyhightower/envconfig"
)

const ApplicationName = "snips"

type Config struct {
	Debug bool `default:"false"`
	URL   struct {
		Internal url.URL `default:"http://0.0.0.0:8080"`
		External url.URL `default:"http://0.0.0.0:8080"`
	}

	HMACKey string `default:"correct-horse-battery-staple"`

	SSH struct {
		Port        int    `default:"2222"`
		HostKeyPath string `default:"tmp/keys/default"`
	}

	DB struct {
		FilePath string `default:"tmp/default.db"`
		Migrate  bool   `default:"false"`
	}
}

func (cfg *Config) Usage() {
	envconfig.Usage(ApplicationName, cfg)
}

func (cfg *Config) SSHListenAddr() string {
	return net.JoinHostPort(cfg.URL.Internal.Hostname(), strconv.Itoa(cfg.SSH.Port))
}

func (cfg *Config) HTTPListenAddr() string {
	return cfg.URL.Internal.Host
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := envconfig.Process(ApplicationName, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

package config

import (
	"net"
	"strconv"

	"github.com/kelseyhightower/envconfig"
)

const ApplicationName = "snips"

type Config struct {
	Debug bool `default:"false"`
	Host  struct {
		Internal string `default:"localhost"`
		External string `default:"localhost"`
	}

	HMACKey string `default:"correct-horse-battery-staple"`

	HTTP struct {
		Port int `default:"8080"`
	}

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
	return net.JoinHostPort(cfg.Host.Internal, strconv.Itoa(cfg.SSH.Port))
}

func (cfg *Config) HTTPListenAddr() string {
	return net.JoinHostPort(cfg.Host.Internal, strconv.Itoa(cfg.HTTP.Port))
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := envconfig.Process(ApplicationName, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

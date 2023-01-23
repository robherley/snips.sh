package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

const ApplicationName = "snips"

type Config struct {
	Debug bool   `default:"false"`
	Host  string `default:"localhost"`

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

func (cfg *Config) SSHAddress() string {
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.SSH.Port)
}

func (cfg *Config) HTTPAddress() string {
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.HTTP.Port)
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := envconfig.Process(ApplicationName, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

const ApplicationName = "snips"

type Config struct {
	Host string `default:"localhost"`

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

func Load() (*Config, error) {
	cfg := &Config{}

	if err := envconfig.Process(ApplicationName, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

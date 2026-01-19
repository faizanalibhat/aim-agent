package config

import (
	"os"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BackendURL string `yaml:"backend_url"`
	APIKey     string `yaml:"api_key"`
	Interval   int    `yaml:"interval"` // in seconds
}

func GetDefaultConfigPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\ProgramData\\snapsec-agent\\config.yaml"
	}
	return "/etc/snapsec-agent.yaml"
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Interval == 0 {
		cfg.Interval = 5 // Default to 5 seconds
	}

	return &cfg, nil
}

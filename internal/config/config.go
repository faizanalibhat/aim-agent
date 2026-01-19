package config

import (
	"os"
	"runtime"

	"gopkg.in/yaml.v3"
)

const Version = "1.0.0"

type Config struct {
	BackendURL string `yaml:"backend_url"`
	APIKey     string `yaml:"api_key"`
	AgentID    string `yaml:"agent_id,omitempty"`
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

func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

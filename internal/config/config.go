package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PollInterval time.Duration `yaml:"poll_interval"`
	Sources      []Source      `yaml:"sources"`
	Targets      []Target      `yaml:"targets"`
	Rules        []Rule        `yaml:"rules"`
	API          APIConfig     `yaml:"api"`
}

type Source struct {
	ID       string `yaml:"id"`
	Host     string `yaml:"host"`     // e.g. imap.example.com:993
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Target struct {
	ID       string `yaml:"id"`
	Host     string `yaml:"host"`     // e.g. smtp.example.com:465
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Email    string `yaml:"email"`
}

type Rule struct {
	ID      string   `yaml:"id"`
	Sources []string `yaml:"sources"`
	Targets []string `yaml:"targets"`
}

type APIConfig struct {
	Address string `yaml:"address"` // e.g. 127.0.0.1:8080
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	
	// Default API address
	if cfg.API.Address == "" {
		cfg.API.Address = "127.0.0.1:8080"
	}
	// Default poll interval
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 60 * time.Second
	}
	
	return &cfg, nil
}

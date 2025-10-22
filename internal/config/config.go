package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration loaded from YAML.
type Config struct {
	Latitude  float64       `yaml:"latitude"`
	Longitude float64       `yaml:"longitude"`
	Weather   WeatherConfig `yaml:"weather"`
	Mailgun   MailgunConfig `yaml:"mailgun"`
	StateDir  string        `yaml:"state_dir"`
}

// WeatherConfig contains settings for communicating with weather.gov.
type WeatherConfig struct {
	UserAgent  string        `yaml:"user_agent"`
	TimeoutRaw string        `yaml:"timeout"` // duration string, e.g. "10s"
	Timeout    time.Duration `yaml:"-"`
}

// MailgunConfig contains credentials and addressing for the Mailgun API.
type MailgunConfig struct {
	Domain string `yaml:"domain"`
	APIKey string `yaml:"api_key"`
	From   string `yaml:"from"`
	To     string `yaml:"to"`
}

// Load reads and validates configuration from the provided YAML file path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validateAndNormalize(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validateAndNormalize() error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	if c.Weather.UserAgent == "" {
		return errors.New("weather.user_agent is required")
	}

	if c.Weather.TimeoutRaw == "" {
		c.Weather.Timeout = 10 * time.Second
	} else {
		d, err := time.ParseDuration(c.Weather.TimeoutRaw)
		if err != nil {
			return fmt.Errorf("invalid weather.timeout: %w", err)
		}
		if d <= 0 {
			return errors.New("weather.timeout must be positive")
		}
		c.Weather.Timeout = d
	}

	if c.Mailgun.Domain == "" {
		return errors.New("mailgun.domain is required")
	}
	if c.Mailgun.APIKey == "" {
		return errors.New("mailgun.api_key is required")
	}
	if c.Mailgun.From == "" {
		return errors.New("mailgun.from is required")
	}
	if c.Mailgun.To == "" {
		return errors.New("mailgun.to is required")
	}
	if c.StateDir == "" {
		return errors.New("state_dir is required")
	}

	// Ensure StateDir is an absolute path for clarity.
	if !filepath.IsAbs(c.StateDir) {
		abs, err := filepath.Abs(c.StateDir)
		if err != nil {
			return fmt.Errorf("resolve state_dir: %w", err)
		}
		c.StateDir = abs
	}

	return nil
}

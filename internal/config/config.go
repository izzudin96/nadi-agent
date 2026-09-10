package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DeviceID        string          `yaml:"device_id"`
	ServerURL       string          `yaml:"server_url"`
	APIKey          string          `yaml:"api_key"`
	IntervalSeconds int             `yaml:"interval_seconds"`
	JitterSeconds   int             `yaml:"jitter_seconds"`
	BufferPath      string          `yaml:"buffer_path"`
	BufferMaxSizeMB int             `yaml:"buffer_max_size_mb"`
	LogLevel        string          `yaml:"log_level"`
	Collectors      map[string]bool `yaml:"collectors"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	cfg.applyDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return &cfg, nil
}

// applyDefaults fills in optional fields so a minimal config still buffers
// safely (a zero buffer_max_size_mb would silently disable buffering).
func (c *Config) applyDefaults() {
	if c.BufferPath == "" {
		c.BufferPath = "agent-buffer.db"
	}
	if c.BufferMaxSizeMB == 0 {
		c.BufferMaxSizeMB = 50
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}

func (c *Config) Validate() error {
	var missing []string
	if c.DeviceID == "" {
		missing = append(missing, "device_id")
	}
	if c.ServerURL == "" {
		missing = append(missing, "server_url")
	}
	if c.APIKey == "" {
		missing = append(missing, "api_key")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}

	u, err := url.Parse(c.ServerURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("server_url must be a valid http(s) URL, got %q", c.ServerURL)
	}

	if c.IntervalSeconds <= 0 {
		return errors.New("interval_seconds must be > 0")
	}
	if c.JitterSeconds < 0 {
		return errors.New("jitter_seconds must be >= 0")
	}
	if c.BufferMaxSizeMB < 0 {
		return errors.New("buffer_max_size_mb must be >= 0")
	}
	return nil
}

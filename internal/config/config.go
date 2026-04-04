package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configurable values for bb.
type Config struct {
	Workspace string `yaml:"workspace"`
	Repo      string `yaml:"repo"`
	Username  string `yaml:"username"`
	Token     string `yaml:"token"`
}

// DefaultPath returns ~/.bbcloud.yaml.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".bbcloud.yaml")
}

// Load reads the config file at path and overlays BITBUCKET_USER / BITBUCKET_TOKEN env vars.
// If the file does not exist, Load returns a zero-value Config (not an error).
func Load(path string) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}
	}
	if v := os.Getenv("BITBUCKET_USER"); v != "" {
		cfg.Username = v
	}
	if v := os.Getenv("BITBUCKET_TOKEN"); v != "" {
		cfg.Token = v
	}
	return cfg, nil
}

// Apply overlays non-empty flag values onto cfg (highest precedence).
func (cfg *Config) Apply(workspace, repo, username, token string) {
	if workspace != "" {
		cfg.Workspace = workspace
	}
	if repo != "" {
		cfg.Repo = repo
	}
	if username != "" {
		cfg.Username = username
	}
	if token != "" {
		cfg.Token = token
	}
}

// Validate returns an error if required fields are missing.
func (cfg *Config) Validate() error {
	if cfg.Username == "" || cfg.Token == "" {
		return fmt.Errorf("no credentials found — run 'bb setup' to configure")
	}
	if cfg.Workspace == "" {
		return fmt.Errorf("no workspace configured — run 'bb setup' to configure")
	}
	return nil
}

// Save writes cfg to path with 0600 permissions.
func (cfg *Config) Save(path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

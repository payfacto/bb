package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Sentinel errors returned by Validate / ValidateCredentials. Use errors.Is
// to detect them — the wrapped strings include CLI hint text that may evolve,
// but the sentinels are part of bb's public contract.
var (
	ErrNoWorkspace   = errors.New("no workspace configured")
	ErrNoRepo        = errors.New("no repo configured")
	ErrNoUsername    = errors.New("no username configured")
	ErrNoCredentials = errors.New("no credentials found")
)

// CloneActionClone is the default; selecting a clone URL runs git clone directly.
const CloneActionClone = "clone"

// CloneActionCopy makes clone menu items copy the command to the clipboard instead.
const CloneActionCopy = "copy"

// ThemeDefault is the colour theme used when none is configured.
const ThemeDefault = "catppuccin"

// AppPasswordDeadline is the date Bitbucket Cloud app passwords stop working.
const AppPasswordDeadline = "2026-06-09"

// Config holds all configurable values for bb.
type Config struct {
	Workspace     string `yaml:"workspace"`
	Repo          string `yaml:"repo"`
	Username      string `yaml:"username"`
	AuthType      string `yaml:"auth_type,omitempty"`
	OAuthClientID string `yaml:"oauth_client_id,omitempty"`
	PageSize      int    `yaml:"page_size,omitempty"`
	CloneAction   string `yaml:"clone_action,omitempty"`
	Theme         string `yaml:"theme,omitempty"`

	// Token is never written to disk; loaded from keyring, env var, or CLI flag at runtime.
	Token string `yaml:"-"`
}

// HasOAuth returns true when the config is set up for OAuth authentication.
func (cfg *Config) HasOAuth() bool {
	return cfg.AuthType == "oauth"
}

// IsLegacyAppPassword reports whether cfg's auth_type denotes the deprecated
// app-password method — the historical default (empty auth_type) or the
// explicit "apppassword". It classifies the auth_type only and does NOT check
// for a token; callers that warn the user (e.g. bb auth status) should also
// confirm a credential exists before surfacing the 2026-06-09 deprecation
// notice. API token ("apitoken") and OAuth ("oauth") configs return false.
func (cfg *Config) IsLegacyAppPassword() bool {
	return cfg.AuthType == "" || cfg.AuthType == "apppassword"
}

// DefaultPath returns ~/.bbcloud.yaml, or .bbcloud.yaml in the current directory
// if the home directory cannot be determined.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bbcloud.yaml"
	}
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
	if cfg.CloneAction == "" {
		cfg.CloneAction = CloneActionClone
	}
	if cfg.Theme == "" {
		cfg.Theme = ThemeDefault
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

// Validate returns an error if workspace or username are missing.
// Token is validated separately after keyring resolution via ValidateCredentials.
// Errors wrap ErrNoWorkspace / ErrNoUsername — use errors.Is to detect them.
func (cfg *Config) Validate() error {
	if cfg.Workspace == "" {
		return fmt.Errorf("%w (run 'bb setup' or 'bb auth login')", ErrNoWorkspace)
	}
	if cfg.Username == "" {
		return fmt.Errorf("%w (run 'bb setup' or 'bb auth login')", ErrNoUsername)
	}
	return nil
}

// ValidateCredentials returns an error wrapping ErrNoCredentials if no token
// is available.
func (cfg *Config) ValidateCredentials() error {
	if cfg.Token == "" {
		return fmt.Errorf("%w (run 'bb auth login' or set BITBUCKET_TOKEN)", ErrNoCredentials)
	}
	return nil
}

// Save writes cfg to path with 0600 permissions. Token is never written (yaml:"-").
func (cfg *Config) Save(path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

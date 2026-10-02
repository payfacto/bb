package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/payfacto/bb/internal/config"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

func clearOAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("BITBUCKET_USER", "")
	t.Setenv("BITBUCKET_TOKEN", "")
	t.Setenv("BB_OAUTH_CLIENT_ID", "")
	t.Setenv("BB_OAUTH_CLIENT_SECRET", "")
}

func TestLoadWithSystemPreseedsClientIDWhenUserFileLacksIt(t *testing.T) {
	clearOAuthEnv(t)
	dir := t.TempDir()
	system := writeFile(t, dir, "system.yaml", "oauth_client_id: sys-id\nworkspace: sys-ws\n")
	user := writeFile(t, dir, "user.yaml", "username: me@example.com\n")

	cfg, err := config.LoadWithSystem(user, system)
	if err != nil {
		t.Fatalf("LoadWithSystem: %v", err)
	}
	if cfg.OAuthClientID != "sys-id" {
		t.Errorf("OAuthClientID = %q, want sys-id", cfg.OAuthClientID)
	}
	if cfg.Workspace != "sys-ws" {
		t.Errorf("Workspace = %q, want sys-ws", cfg.Workspace)
	}
	if cfg.Username != "me@example.com" {
		t.Errorf("Username = %q, want me@example.com", cfg.Username)
	}
}

func TestLoadWithSystemUserFileOverridesSystem(t *testing.T) {
	clearOAuthEnv(t)
	dir := t.TempDir()
	system := writeFile(t, dir, "system.yaml", "oauth_client_id: sys-id\nworkspace: sys-ws\n")
	user := writeFile(t, dir, "user.yaml", "oauth_client_id: user-id\n")

	cfg, err := config.LoadWithSystem(user, system)
	if err != nil {
		t.Fatalf("LoadWithSystem: %v", err)
	}
	if cfg.OAuthClientID != "user-id" {
		t.Errorf("OAuthClientID = %q, want user-id", cfg.OAuthClientID)
	}
	if cfg.Workspace != "sys-ws" {
		t.Errorf("Workspace = %q, want sys-ws (user file omitted it)", cfg.Workspace)
	}
}

func TestLoadWithSystemMissingSystemFileIsNotAnError(t *testing.T) {
	clearOAuthEnv(t)
	dir := t.TempDir()
	user := writeFile(t, dir, "user.yaml", "workspace: ws\n")

	cfg, err := config.LoadWithSystem(user, filepath.Join(dir, "absent.yaml"))
	if err != nil {
		t.Fatalf("LoadWithSystem: %v", err)
	}
	if cfg.Workspace != "ws" {
		t.Errorf("Workspace = %q, want ws", cfg.Workspace)
	}
}

func TestLoadWithSystemMalformedSystemFileIsAnError(t *testing.T) {
	clearOAuthEnv(t)
	dir := t.TempDir()
	system := writeFile(t, dir, "system.yaml", "workspace: [unclosed\n")
	user := writeFile(t, dir, "user.yaml", "workspace: ws\n")

	if _, err := config.LoadWithSystem(user, system); err == nil {
		t.Fatal("expected parse error for malformed system config")
	}
}

func TestLoadEnvOverridesOAuthClientIDAndSetsSecret(t *testing.T) {
	clearOAuthEnv(t)
	dir := t.TempDir()
	system := writeFile(t, dir, "system.yaml", "oauth_client_id: sys-id\n")
	user := writeFile(t, dir, "user.yaml", "oauth_client_id: user-id\n")
	t.Setenv("BB_OAUTH_CLIENT_ID", "env-id")
	t.Setenv("BB_OAUTH_CLIENT_SECRET", "env-secret")

	cfg, err := config.LoadWithSystem(user, system)
	if err != nil {
		t.Fatalf("LoadWithSystem: %v", err)
	}
	if cfg.OAuthClientID != "env-id" {
		t.Errorf("OAuthClientID = %q, want env-id", cfg.OAuthClientID)
	}
	if cfg.OAuthClientSecret != "env-secret" {
		t.Errorf("OAuthClientSecret = %q, want env-secret", cfg.OAuthClientSecret)
	}
}

func TestOAuthClientSecretIsNeverPersisted(t *testing.T) {
	cfg := &config.Config{Workspace: "ws", OAuthClientSecret: "super-secret"}
	path := filepath.Join(t.TempDir(), "out.yaml")
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "super-secret") {
		t.Error("OAuth client secret must not be written to the config file")
	}
}

func TestSystemPathIsAbsolute(t *testing.T) {
	if p := config.SystemPath(); !filepath.IsAbs(p) {
		t.Errorf("SystemPath() = %q, want an absolute path", p)
	}
}

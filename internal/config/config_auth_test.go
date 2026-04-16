package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/payfacto/bb/internal/config"
)

func TestConfigDoesNotPersistToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &config.Config{
		Workspace:     "ws",
		Username:      "user",
		Token:         "secret-token",
		AuthType:      "apppassword",
		OAuthClientID: "",
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) == "" {
		t.Fatal("expected non-empty config file")
	}
	if strings.Contains(string(data), "secret-token") {
		t.Error("token must not be written to config file")
	}
}

func TestConfigAuthTypeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &config.Config{
		Workspace:     "ws",
		Username:      "user",
		AuthType:      "oauth",
		OAuthClientID: "my-client-id",
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.AuthType != "oauth" {
		t.Errorf("auth_type: got %q, want %q", loaded.AuthType, "oauth")
	}
	if loaded.OAuthClientID != "my-client-id" {
		t.Errorf("oauth_client_id: got %q, want %q", loaded.OAuthClientID, "my-client-id")
	}
}

func TestHasOAuth(t *testing.T) {
	cfg := &config.Config{AuthType: "oauth"}
	if !cfg.HasOAuth() {
		t.Error("HasOAuth() should return true for auth_type=oauth")
	}
	cfg2 := &config.Config{AuthType: "apppassword"}
	if cfg2.HasOAuth() {
		t.Error("HasOAuth() should return false for auth_type=apppassword")
	}
}

func TestValidateCredentials(t *testing.T) {
	cfg := &config.Config{Workspace: "ws", Username: "user", Token: "tok"}
	if err := cfg.ValidateCredentials(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	cfg2 := &config.Config{Workspace: "ws", Username: "user"}
	if err := cfg2.ValidateCredentials(); err == nil {
		t.Error("expected error for empty token")
	}
}

func TestValidateNoLongerRequiresToken(t *testing.T) {
	// Validate() should only check workspace + username, NOT token
	cfg := &config.Config{Workspace: "ws", Username: "user"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() should not require token, got: %v", err)
	}
}

func TestCloneActionDefaultsToClone(t *testing.T) {
	dir := t.TempDir()
	// Load from non-existent file — all fields zero-valued before defaulting.
	cfg, err := config.Load(filepath.Join(dir, "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CloneAction != "clone" {
		t.Errorf("CloneAction default: got %q, want %q", cfg.CloneAction, "clone")
	}
}

func TestCloneActionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{Workspace: "ws", Username: "user", CloneAction: "copy"}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.CloneAction != "copy" {
		t.Errorf("CloneAction round-trip: got %q, want %q", loaded.CloneAction, "copy")
	}
}

func TestThemeDefaultsToCatppuccin(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(filepath.Join(dir, "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Theme != "catppuccin" {
		t.Errorf("Theme default: got %q, want %q", cfg.Theme, "catppuccin")
	}
}

func TestThemeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{Workspace: "ws", Username: "user", Theme: "nord"}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Theme != "nord" {
		t.Errorf("Theme round-trip: got %q, want %q", loaded.Theme, "nord")
	}
}

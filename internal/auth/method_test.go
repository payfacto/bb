package auth_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
)

func TestRecommendPrefersOAuthOnADesktopWithKeyring(t *testing.T) {
	rec := auth.Recommend(auth.Env{HasDisplay: true, KeyringOK: true})
	if rec.Method != config.AuthTypeOAuth {
		t.Errorf("Method = %q, want %q", rec.Method, config.AuthTypeOAuth)
	}
	if rec.Reason != "" {
		t.Errorf("Reason = %q, want empty when OAuth is recommended", rec.Reason)
	}
}

func TestRecommendFallsBackToAPITokenWithReason(t *testing.T) {
	tests := []struct {
		name       string
		env        auth.Env
		wantReason string
	}{
		{"no display", auth.Env{HasDisplay: false, KeyringOK: true}, "no display"},
		{"ssh session", auth.Env{HasDisplay: true, InSSH: true, KeyringOK: true}, "SSH"},
		{"no keyring", auth.Env{HasDisplay: true, KeyringOK: false}, "keyring"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := auth.Recommend(tc.env)
			if rec.Method != config.AuthTypeAPIToken {
				t.Errorf("Method = %q, want %q", rec.Method, config.AuthTypeAPIToken)
			}
			if !strings.Contains(rec.Reason, tc.wantReason) {
				t.Errorf("Reason = %q, want it to mention %q", rec.Reason, tc.wantReason)
			}
		})
	}
}

func TestDetectEnvReportsSSHFromEnvironment(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "10.0.0.1 5000 10.0.0.2 22")
	if !auth.DetectEnv().InSSH {
		t.Error("InSSH = false, want true when SSH_CONNECTION is set")
	}
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	if auth.DetectEnv().InSSH {
		t.Error("InSSH = true, want false when no SSH variables are set")
	}
}

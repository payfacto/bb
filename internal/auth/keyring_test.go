package auth_test

import (
	"errors"
	"testing"

	"github.com/payfacto/bb/internal/auth"
	keyring "github.com/zalando/go-keyring"
)

func init() {
	// Use in-memory mock keyring for tests — never touches OS keyring.
	keyring.MockInit()
}

func TestSetAndGetToken(t *testing.T) {
	if err := auth.SetToken("user@example.com", "mytoken123"); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	got, err := auth.GetToken("user@example.com")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if got != "mytoken123" {
		t.Errorf("got %q, want %q", got, "mytoken123")
	}
}

func TestGetTokenNotFound(t *testing.T) {
	_, err := auth.GetToken("nobody@example.com")
	if !errors.Is(err, auth.ErrTokenNotFound) {
		t.Errorf("want ErrTokenNotFound, got %v", err)
	}
}

func TestDeleteToken(t *testing.T) {
	_ = auth.SetToken("delete@example.com", "tok")

	if err := auth.DeleteToken("delete@example.com"); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}

	_, err := auth.GetToken("delete@example.com")
	if !errors.Is(err, auth.ErrTokenNotFound) {
		t.Errorf("want ErrTokenNotFound after delete, got %v", err)
	}
}

func TestRefreshTokenAndClientSecretRoundTrip(t *testing.T) {
	const user = "oauth@example.com"
	if err := auth.SetRefreshToken(user, "refresh-abc"); err != nil {
		t.Fatalf("SetRefreshToken: %v", err)
	}
	if err := auth.SetClientSecret(user, "secret-xyz"); err != nil {
		t.Fatalf("SetClientSecret: %v", err)
	}

	if got, err := auth.GetRefreshToken(user); err != nil || got != "refresh-abc" {
		t.Errorf("GetRefreshToken = (%q, %v), want (%q, nil)", got, err, "refresh-abc")
	}
	if got, err := auth.GetClientSecret(user); err != nil || got != "secret-xyz" {
		t.Errorf("GetClientSecret = (%q, %v), want (%q, nil)", got, err, "secret-xyz")
	}
}

func TestDeleteTokenRemovesAllOAuthCredentials(t *testing.T) {
	const user = "oauthdelete@example.com"
	_ = auth.SetToken(user, "access")
	_ = auth.SetRefreshToken(user, "refresh")
	_ = auth.SetClientSecret(user, "secret")

	if err := auth.DeleteToken(user); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}

	for name, get := range map[string]func(string) (string, error){
		"access token":  auth.GetToken,
		"refresh token": auth.GetRefreshToken,
		"client secret": auth.GetClientSecret,
	} {
		if _, err := get(user); !errors.Is(err, auth.ErrTokenNotFound) {
			t.Errorf("%s: want ErrTokenNotFound after delete, got %v", name, err)
		}
	}
}

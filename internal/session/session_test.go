package session_test

import (
	"testing"

	keyring "github.com/zalando/go-keyring"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/session"
)

func init() {
	// In-memory keyring: tests never touch the OS keyring.
	keyring.MockInit()
}

func TestStoreOAuthCredentialsPersistsAllThree(t *testing.T) {
	tok := &auth.Token{AccessToken: "access-1", RefreshToken: "refresh-1"}
	if err := session.StoreOAuthCredentials("u@example.com", "secret-1", tok); err != nil {
		t.Fatalf("StoreOAuthCredentials: %v", err)
	}
	if got, _ := auth.GetToken("u@example.com"); got != "access-1" {
		t.Errorf("access token = %q, want access-1", got)
	}
	if got, _ := auth.GetRefreshToken("u@example.com"); got != "refresh-1" {
		t.Errorf("refresh token = %q, want refresh-1", got)
	}
	if got, _ := auth.GetClientSecret("u@example.com"); got != "secret-1" {
		t.Errorf("client secret = %q, want secret-1", got)
	}
}

func TestStoreOAuthCredentialsKeepsExistingRefreshTokenWhenNoneReturned(t *testing.T) {
	user := "keep@example.com"
	first := &auth.Token{AccessToken: "a1", RefreshToken: "r1"}
	if err := session.StoreOAuthCredentials(user, "s", first); err != nil {
		t.Fatalf("first store: %v", err)
	}
	second := &auth.Token{AccessToken: "a2"} // Bitbucket did not rotate the refresh token
	if err := session.StoreOAuthCredentials(user, "s", second); err != nil {
		t.Fatalf("second store: %v", err)
	}
	if got, _ := auth.GetRefreshToken(user); got != "r1" {
		t.Errorf("refresh token = %q, want r1 preserved", got)
	}
}

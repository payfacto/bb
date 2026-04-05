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

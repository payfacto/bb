package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/payfacto/bb/internal/auth"
)

func TestBuildAuthURL(t *testing.T) {
	url := auth.BuildAuthURL("my-client-id", "mystate123", "http://localhost:9999/callback")
	if !strings.Contains(url, "client_id=my-client-id") {
		t.Errorf("missing client_id in URL: %s", url)
	}
	if !strings.Contains(url, "state=mystate123") {
		t.Errorf("missing state in URL: %s", url)
	}
	if !strings.Contains(url, "response_type=code") {
		t.Errorf("missing response_type in URL: %s", url)
	}
	if !strings.Contains(url, "redirect_uri=") {
		t.Errorf("missing redirect_uri in URL: %s", url)
	}
}

func TestGenerateState(t *testing.T) {
	s1, err := auth.GenerateState()
	if err != nil {
		t.Fatalf("GenerateState: %v", err)
	}
	s2, _ := auth.GenerateState()
	if s1 == s2 {
		t.Error("two states should not be equal")
	}
	if len(s1) < 16 {
		t.Errorf("state too short: %q", s1)
	}
}

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("wrong grant_type: %s", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "testcode" {
			t.Errorf("wrong code: %s", r.FormValue("code"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-abc",
			"refresh_token": "refresh-xyz",
			"token_type":    "bearer",
			"expires_in":    7200,
		})
	}))
	defer srv.Close()

	tok, err := auth.ExchangeCode(srv.URL, "client-id", "client-secret", "testcode", "http://localhost/cb")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tok.AccessToken != "access-abc" {
		t.Errorf("access_token: got %q", tok.AccessToken)
	}
	if tok.RefreshToken != "refresh-xyz" {
		t.Errorf("refresh_token: got %q", tok.RefreshToken)
	}
}

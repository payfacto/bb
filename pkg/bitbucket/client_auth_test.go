package bitbucket_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestClientUsesBearerTokenWhenSet(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"account_id":"123","username":"user","display_name":"User"}`))
	}))
	defer srv.Close()

	cfg := &config.Config{Username: "user", Token: ""}
	c := bitbucket.NewWithBaseURL(cfg, srv.URL)
	c.SetBearerToken("my-oauth-token")

	_, _ = c.User().Me(context.Background())

	if gotAuth != "Bearer my-oauth-token" {
		t.Errorf("Authorization header: got %q, want %q", gotAuth, "Bearer my-oauth-token")
	}
}

func TestClientUsesBasicAuthWhenNoBearer(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"account_id":"123","username":"user","display_name":"User"}`))
	}))
	defer srv.Close()

	cfg := &config.Config{Username: "user", Token: "apppassword"}
	c := bitbucket.NewWithBaseURL(cfg, srv.URL)

	_, _ = c.User().Me(context.Background())

	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("Authorization header: got %q, want Basic ...", gotAuth)
	}
}

func TestClientAPITokenUsesEmailBasicAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"account_id":"123","username":"user","display_name":"User"}`))
	}))
	defer srv.Close()

	// API token auth: username field holds the Atlassian email, token is the API token.
	cfg := &config.Config{
		Username: "jmadore@payfacto.com",
		Token:    "api-token-123",
		AuthType: "apitoken",
	}
	c := bitbucket.NewWithBaseURL(cfg, srv.URL)

	_, _ = c.User().Me(context.Background())

	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("Authorization header: got %q, want Basic ...", gotAuth)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(gotAuth, "Basic "))
	if err != nil {
		t.Fatalf("decode basic auth: %v", err)
	}
	if string(raw) != "jmadore@payfacto.com:api-token-123" {
		t.Errorf("basic auth payload: got %q, want %q", string(raw), "jmadore@payfacto.com:api-token-123")
	}
}

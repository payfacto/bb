package bitbucket_test

import (
	"context"
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

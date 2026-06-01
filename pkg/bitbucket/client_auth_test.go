package bitbucket_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
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

func TestBearer401TriggersRefreshAndRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer fresh-token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"account_id":"123","username":"user","display_name":"User"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"message":"expired"}}`))
	}))
	defer srv.Close()

	c := bitbucket.NewWithBaseURL(&config.Config{Username: "user"}, srv.URL)
	c.SetBearerToken("stale-token")
	var refreshCalls int32
	c.SetTokenRefresher(func() (string, error) {
		atomic.AddInt32(&refreshCalls, 1)
		return "fresh-token", nil
	})

	u, err := c.User().Me(context.Background())
	if err != nil {
		t.Fatalf("Me after refresh: %v", err)
	}
	if u.DisplayName != "User" {
		t.Errorf("display_name: got %q, want %q", u.DisplayName, "User")
	}
	if got := atomic.LoadInt32(&refreshCalls); got != 1 {
		t.Errorf("refresher called %d times, want 1", got)
	}
}

func TestBearer401WithoutRefresherReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"message":"expired"}}`))
	}))
	defer srv.Close()

	c := bitbucket.NewWithBaseURL(&config.Config{Username: "user"}, srv.URL)
	c.SetBearerToken("stale-token")

	if _, err := c.User().Me(context.Background()); err == nil {
		t.Fatal("expected error from 401 with no refresher")
	}
}

func TestBearer401RefresherErrorSurfaces401(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"message":"expired"}}`))
	}))
	defer srv.Close()

	c := bitbucket.NewWithBaseURL(&config.Config{Username: "user"}, srv.URL)
	c.SetBearerToken("stale-token")
	c.SetTokenRefresher(func() (string, error) {
		return "", errors.New("refresh boom")
	})

	_, err := c.User().Me(context.Background())
	if err == nil {
		t.Fatal("expected error when refresher fails")
	}
	// The original 401 must be surfaced; the request is not retried after a
	// failed refresh.
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("server saw %d attempts, want 1 (no retry after refresh failure)", got)
	}
}

func TestConcurrent401sRefreshOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer fresh-token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"account_id":"123","username":"user","display_name":"User"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"message":"expired"}}`))
	}))
	defer srv.Close()

	c := bitbucket.NewWithBaseURL(&config.Config{Username: "user"}, srv.URL)
	c.SetBearerToken("stale-token")
	var refreshCalls int32
	c.SetTokenRefresher(func() (string, error) {
		atomic.AddInt32(&refreshCalls, 1)
		return "fresh-token", nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.User().Me(context.Background())
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&refreshCalls); got != 1 {
		t.Errorf("refresher called %d times, want exactly 1", got)
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

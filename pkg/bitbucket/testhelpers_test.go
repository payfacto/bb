package bitbucket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/payfactopay/bb/internal/config"
	"github.com/payfactopay/bb/pkg/bitbucket"
)

// newTestClient creates a Client pointed at a test HTTP server.
// The server is automatically closed when the test ends.
func newTestClient(t *testing.T, handler http.Handler) *bitbucket.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := &config.Config{
		Username:  "testuser",
		Token:     "testtoken",
		Workspace: "ws",
		Repo:      "repo",
	}
	return bitbucket.NewWithBaseURL(cfg, srv.URL)
}

// mustEncodeJSON writes v as JSON to w and fails the test if encoding fails.
func mustEncodeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatal(err)
	}
}

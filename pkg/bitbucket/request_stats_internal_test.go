package bitbucket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/payfacto/bb/internal/config"
)

// newStatsClient points a Client at handler and makes rate-limit waits instant.
func newStatsClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewWithBaseURL(&config.Config{Username: "u", Token: "t"}, srv.URL)
	c.sleep = func(context.Context, time.Duration) error { return nil }
	return c
}

func TestStatsCountsEveryAttemptIncludingRateLimitRetries(t *testing.T) {
	calls := 0
	c := newStatsClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})

	if _, err := c.do(context.Background(), "GET", "/x", nil, nil); err != nil {
		t.Fatalf("do: %v", err)
	}

	got := c.Stats()
	if got.Requests != 3 || got.RateLimited != 2 {
		t.Errorf("Stats() = %d requests, %d rate limited; want 3 and 2", got.Requests, got.RateLimited)
	}
}

func TestStatsKeepsTheLatestRateLimitHeaders(t *testing.T) {
	c := newStatsClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "1000")
		w.Header().Set("X-RateLimit-Resource", r.URL.Path)
		w.Header().Set("Retry-After", "7")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	for _, path := range []string{"/first", "/second"} {
		if _, err := c.do(context.Background(), "GET", path, nil, nil); err != nil {
			t.Fatalf("do %s: %v", path, err)
		}
	}

	got := c.Stats().RateLimitHeaders
	want := map[string]string{"X-Ratelimit-Limit": "1000", "X-Ratelimit-Resource": "/second", "Retry-After": "7"}
	if len(got) != len(want) {
		t.Fatalf("RateLimitHeaders = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("RateLimitHeaders[%q] = %q, want %q", k, got[k], v)
		}
	}
}

func TestStatsIsZeroBeforeAnyRequest(t *testing.T) {
	c := New(&config.Config{})
	if got := c.Stats(); got.Requests != 0 || got.RateLimited != 0 || len(got.RateLimitHeaders) != 0 {
		t.Errorf("Stats() = %+v, want zero", got)
	}
}

func TestStatsReturnsACopyOfTheHeaders(t *testing.T) {
	c := newStatsClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "1000")
		_, _ = w.Write([]byte(`{}`))
	})
	if _, err := c.do(context.Background(), "GET", "/x", nil, nil); err != nil {
		t.Fatalf("do: %v", err)
	}

	c.Stats().RateLimitHeaders["X-Ratelimit-Limit"] = "changed"

	if got := c.Stats().RateLimitHeaders["X-Ratelimit-Limit"]; got != "1000" {
		t.Errorf("X-Ratelimit-Limit = %q after editing a returned copy, want 1000", got)
	}
}

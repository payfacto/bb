package bitbucket

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/payfacto/bb/internal/config"
)

// rateLimitedServer answers the first limitedCalls requests with HTTP 429
// (plus retryAfter as the Retry-After header when set), then 200.
type rateLimitedServer struct {
	limitedCalls int
	retryAfter   string
	calls        int
}

func (s *rateLimitedServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.calls++
	if s.calls <= s.limitedCalls {
		if s.retryAfter != "" {
			w.Header().Set("Retry-After", s.retryAfter)
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "Rate limit for this resource has been exceeded"}}`))
		return
	}
	_, _ = w.Write([]byte(`{"ok": true}`))
}

// newRateLimitClient returns a client for handler whose backoff waits are
// recorded instead of slept.
func newRateLimitClient(t *testing.T, handler http.Handler) (*Client, *[]time.Duration) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewWithBaseURL(&config.Config{Username: "u", Token: "t"}, srv.URL)
	waits := &[]time.Duration{}
	c.sleep = func(_ context.Context, d time.Duration) error {
		*waits = append(*waits, d)
		return nil
	}
	return c, waits
}

func assertWaits(t *testing.T, got []time.Duration, want ...time.Duration) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("waits: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("waits: got %v, want %v", got, want)
		}
	}
}

func TestRateLimit_GETRetriesWithBackoffThenSucceeds(t *testing.T) {
	srv := &rateLimitedServer{limitedCalls: 2}
	c, waits := newRateLimitClient(t, srv)

	data, err := c.do(context.Background(), "GET", "/x", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"ok": true}` {
		t.Errorf("body: got %s", data)
	}
	assertWaits(t, *waits, 1*time.Second, 2*time.Second)
}

func TestRateLimit_GETHonoursRetryAfterSeconds(t *testing.T) {
	srv := &rateLimitedServer{limitedCalls: 1, retryAfter: "7"}
	c, waits := newRateLimitClient(t, srv)

	if _, err := c.do(context.Background(), "GET", "/x", nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertWaits(t, *waits, 7*time.Second)
}

func TestRateLimit_GETGivesUpAfterMaxRetries(t *testing.T) {
	srv := &rateLimitedServer{limitedCalls: 100}
	c, waits := newRateLimitClient(t, srv)

	_, err := c.do(context.Background(), "GET", "/x", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusTooManyRequests {
		t.Fatalf("want a 429 APIError, got %v", err)
	}
	if apiErr.RetryAfter != 0 {
		t.Errorf("RetryAfter should be unknown (0) without the header, got %v", apiErr.RetryAfter)
	}
	if srv.calls != 4 {
		t.Errorf("calls: got %d, want 4 (1 try + 3 retries)", srv.calls)
	}
	assertWaits(t, *waits, 1*time.Second, 2*time.Second, 4*time.Second)
}

func TestRateLimit_LongRetryAfterFailsImmediatelyWithTheDelay(t *testing.T) {
	srv := &rateLimitedServer{limitedCalls: 100, retryAfter: "1800"}
	c, waits := newRateLimitClient(t, srv)

	_, err := c.do(context.Background(), "GET", "/x", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want an APIError, got %v", err)
	}
	if apiErr.RetryAfter != 30*time.Minute {
		t.Errorf("RetryAfter: got %v, want 30m", apiErr.RetryAfter)
	}
	if srv.calls != 1 {
		t.Errorf("calls: got %d, want 1", srv.calls)
	}
	assertWaits(t, *waits)
}

func TestRateLimit_TotalWaitIsBounded(t *testing.T) {
	srv := &rateLimitedServer{limitedCalls: 100, retryAfter: "40"}
	c, waits := newRateLimitClient(t, srv)

	_, err := c.do(context.Background(), "GET", "/x", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != 40*time.Second {
		t.Fatalf("want a 429 APIError with RetryAfter 40s, got %v", err)
	}
	// A second 40s wait would pass the 60s budget, so only one is taken.
	assertWaits(t, *waits, 40*time.Second)
}

func TestRateLimit_HugeRetryAfterAfterABackoffDoesNotOverflowTheBudget(t *testing.T) {
	// First 429 has no header (1s backoff); the second asks for an absurd
	// delay that would wrap a naive waited+wait sum into a negative number.
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls > 1 {
			w.Header().Set("Retry-After", "9223372036")
		}
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c, waits := newRateLimitClient(t, handler)

	_, err := c.do(context.Background(), "GET", "/x", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusTooManyRequests {
		t.Fatalf("want a 429 APIError, got %v", err)
	}
	assertWaits(t, *waits, 1*time.Second)
}

func TestRateLimit_WritesAreNeverRetried(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			srv := &rateLimitedServer{limitedCalls: 100, retryAfter: "3"}
			c, waits := newRateLimitClient(t, srv)

			_, err := c.do(context.Background(), method, "/x", map[string]string{"a": "b"}, nil)
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.Status != http.StatusTooManyRequests {
				t.Fatalf("want a 429 APIError, got %v", err)
			}
			if apiErr.RetryAfter != 3*time.Second {
				t.Errorf("RetryAfter: got %v, want 3s", apiErr.RetryAfter)
			}
			if srv.calls != 1 {
				t.Errorf("calls: got %d, want 1", srv.calls)
			}
			assertWaits(t, *waits)
		})
	}
}

func TestRateLimit_CancelledContextAbortsTheWait(t *testing.T) {
	srv := &rateLimitedServer{limitedCalls: 100}
	c, _ := newRateLimitClient(t, srv)
	c.sleep = sleepContext // the real, cancellable wait

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.do(ctx, "GET", "/x", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name, header string
		want         time.Duration
	}{
		{"absent", "", 0},
		{"seconds", "42", 42 * time.Second},
		{"zero", "0", 0},
		{"negative is ignored", "-5", 0},
		{"garbage is ignored", "soon", 0},
		{"http date in the future", "Thu, 19 Nov 2026 12:00:30 GMT", 30 * time.Second},
		{"http date in the past", "Thu, 19 Nov 2026 11:00:00 GMT", 0},
		{"huge seconds are capped, not overflowed", "9223372036854775807", 24 * time.Hour},
		{"seconds past int64 are ignored", "99999999999999999999999", 0},
		{"far-future http date is capped", "Fri, 19 Nov 2476 12:00:00 GMT", 24 * time.Hour},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseRetryAfter(tc.header, now); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

package bitbucket

import (
	"maps"
	"net/http"
	"strings"
	"sync"
)

// rateLimitHeaderPrefix matches Bitbucket's X-RateLimit-* response headers in
// Go's canonical header form.
const rateLimitHeaderPrefix = "X-Ratelimit-"

// RequestStats summarises the HTTP requests a Client has sent to Bitbucket.
type RequestStats struct {
	// Requests counts every request sent, including rate-limit and
	// token-refresh retries.
	Requests int `json:"requests"`
	// RateLimited counts the responses that came back HTTP 429.
	RateLimited int `json:"rate_limited"`
	// RateLimitHeaders holds the X-RateLimit-* and Retry-After headers of the
	// most recent response that carried any.
	RateLimitHeaders map[string]string `json:"rate_limit_headers,omitempty"`
}

// requestRecorder accumulates RequestStats; safe for concurrent requests.
type requestRecorder struct {
	mu    sync.Mutex
	stats RequestStats
}

func (r *requestRecorder) record(resp *http.Response) {
	headers := rateLimitHeaders(resp.Header)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.Requests++
	if resp.StatusCode == http.StatusTooManyRequests {
		r.stats.RateLimited++
	}
	if len(headers) > 0 {
		r.stats.RateLimitHeaders = headers
	}
}

func (r *requestRecorder) recordFailure() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.Requests++
}

// snapshot copies the stats so callers cannot change the recorder's headers.
func (r *requestRecorder) snapshot() RequestStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	stats := r.stats
	stats.RateLimitHeaders = maps.Clone(r.stats.RateLimitHeaders)
	return stats
}

func rateLimitHeaders(h http.Header) map[string]string {
	found := map[string]string{}
	for name := range h {
		if strings.HasPrefix(name, rateLimitHeaderPrefix) || name == "Retry-After" {
			found[name] = h.Get(name)
		}
	}
	return found
}

// Stats reports the requests this client has sent so far.
func (c *Client) Stats() RequestStats {
	return c.recorder.snapshot()
}

// roundTrip sends req once and records it in the client's request stats.
func (c *Client) roundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recorder.recordFailure()
		return nil, err
	}
	c.recorder.record(resp)
	return resp, nil
}

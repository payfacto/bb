package bitbucket

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"
)

const (
	// rateLimitMaxRetries bounds how often one GET is retried after HTTP 429.
	rateLimitMaxRetries = 3
	// rateLimitMaxWait is the most time one request spends waiting on rate
	// limits, summed over its retries. Bitbucket's limits roll over an hour,
	// so a longer delay is reported to the caller instead of blocking the
	// command.
	rateLimitMaxWait = 60 * time.Second
	// rateLimitBaseBackoff is the first wait when the response carries no
	// Retry-After; it doubles on each retry (1s, 2s, 4s).
	rateLimitBaseBackoff = time.Second
)

// sendWithRateLimitRetry runs req through attempt and, for GET requests only,
// retries after an HTTP 429: it waits the Retry-After delay when the server
// sent one, otherwise an exponential backoff. It gives up, returning the 429
// response, after rateLimitMaxRetries or when the next wait would take the
// request's total waiting past rateLimitMaxWait. Writes are never retried, so a request that Bitbucket may
// have partly applied is not sent twice.
func (c *Client) sendWithRateLimitRetry(req *http.Request, attempt func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	backoff := rateLimitBaseBackoff
	var waited time.Duration
	for retries := 0; ; retries++ {
		resp, err := attempt(req)
		if err != nil || !shouldRetryRateLimited(req, resp, retries) {
			return resp, err
		}
		wait := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
		if wait == 0 {
			wait = backoff
			backoff *= 2
		}
		// Written as a subtraction so a huge Retry-After cannot overflow.
		if wait > rateLimitMaxWait-waited {
			return resp, nil
		}
		waited += wait
		next, ok := cloneForRetry(req)
		if !ok {
			return resp, nil
		}
		resp.Body.Close()
		if err := c.wait(req.Context(), wait); err != nil {
			return nil, err
		}
		req = next
	}
}

func shouldRetryRateLimited(req *http.Request, resp *http.Response, retries int) bool {
	return resp.StatusCode == http.StatusTooManyRequests &&
		req.Method == http.MethodGet &&
		retries < rateLimitMaxRetries
}

// wait pauses for d through the client's sleep hook (sleepContext unless a
// test replaced it).
func (c *Client) wait(ctx context.Context, d time.Duration) error {
	if c.sleep != nil {
		return c.sleep(ctx, d)
	}
	return sleepContext(ctx, d)
}

// sleepContext waits for d, or returns ctx's error as soon as ctx is done.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retryAfterCeiling caps a parsed Retry-After. Nothing waits this long (it
// is far above rateLimitMaxWait); the cap keeps a hostile or broken header
// from overflowing time.Duration arithmetic.
const retryAfterCeiling = 24 * time.Hour

// parseRetryAfter reads a Retry-After header, which is either a number of
// seconds or an HTTP date. It returns 0 when the header is absent,
// unparseable, or not in the future, and at most retryAfterCeiling.
func parseRetryAfter(header string, now time.Time) time.Duration {
	if header == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		if seconds > int64(retryAfterCeiling/time.Second) {
			return retryAfterCeiling
		}
		return max(time.Duration(seconds)*time.Second, 0)
	}
	if at, err := http.ParseTime(header); err == nil {
		return min(max(at.Sub(now), 0), retryAfterCeiling)
	}
	return 0
}

// httpError converts a non-2xx response into an *APIError, recording the
// Retry-After delay of a rate-limited response.
func httpError(resp *http.Response, data []byte) error {
	err := parseHTTPError(resp.StatusCode, data)
	var apiErr *APIError
	if resp.StatusCode == http.StatusTooManyRequests && errors.As(err, &apiErr) {
		apiErr.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	}
	return err
}

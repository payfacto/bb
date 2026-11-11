package bitbucket

import (
	"context"
	"time"
)

// defaultPollInterval is used when a caller passes a non-positive interval.
const defaultPollInterval = 5 * time.Second

// withOptionalTimeout bounds ctx by timeout when timeout > 0; otherwise it
// returns ctx unchanged with a no-op cancel. Waits treat timeout <= 0 as
// "wait indefinitely".
func withOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// pollUntil runs check immediately and then once per interval until check
// reports done, returns an error, or ctx ends (returning ctx.Err()). Callers
// bound the wait with context.WithTimeout and treat DeadlineExceeded as their
// timeout outcome, keeping context.Canceled (Ctrl-C) as an interrupt. A
// non-positive interval falls back to defaultPollInterval.
func pollUntil(ctx context.Context, interval time.Duration, check func(context.Context) (bool, error)) error {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		done, err := check(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if done {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

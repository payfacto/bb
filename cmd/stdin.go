package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

// maxStdinBytes caps how much we will read from a piped stdin. 1 MiB is far
// larger than any legitimate Bitbucket request body and small enough to fail
// fast on a misconfigured pipe (e.g. `yes | bb pr create`).
const maxStdinBytes = 1 << 20

// stdinReadTimeout bounds how long we wait for piped stdin data to start
// arriving before giving up and falling back to flags. term.IsTerminal can
// return a false negative for pty-emulated terminals (observed on Git
// Bash/MinTTY on Windows), making bb think stdin is piped when it is actually
// an interactive shell with nothing coming - this previously caused an
// indefinite hang in io.ReadAll waiting for an EOF that never arrives.
const stdinReadTimeout = 750 * time.Millisecond

// readStdinJSON reads piped stdin (or returns consumed=false when stdin is a
// TTY) and unmarshals it into v. Bounded by stdinReadTimeout: if stdin
// appears to be piped (non-TTY) but no data arrives within the timeout,
// timedOut is true and consumed is false, as if nothing were piped - callers
// fall back to flags and may note the timeout on stderr. err is nil in the
// timeout case (same outward shape as "stdin was empty").
//
// Production callers should use this; tests should call readStdinJSONFrom or
// readStdinJSONWithTimeout directly with a synthetic reader.
func readStdinJSON[T any](v *T) (consumed, timedOut bool, err error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return false, false, nil
	}
	return readStdinJSONWithTimeout(os.Stdin, v, stdinReadTimeout)
}

// readStdinJSONWithTimeout races readStdinJSONFrom against timeout. The
// underlying read is not cancellable (os.Stdin has no portable deadline
// support), so on timeout the read goroutine is abandoned rather than waited
// on - it decodes into a local T of its own, and *v is copied from that local
// only on the winning (non-timed-out, error-free) path. A straggling read
// that eventually completes after the deadline therefore can never write
// into, or race with, the caller's v: the caller has already moved on (e.g.
// built a request from flag-supplied input) by the time this function
// returns on timeout, and the abandoned goroutine's result is simply dropped
// when it eventually lands in the buffered channel that nobody reads again.
func readStdinJSONWithTimeout[T any](r io.Reader, v *T, timeout time.Duration) (consumed, timedOut bool, err error) {
	type result struct {
		val      T
		consumed bool
		err      error
	}
	done := make(chan result, 1)
	go func() {
		var local T
		c, e := readStdinJSONFrom(r, &local)
		done <- result{local, c, e}
	}()
	select {
	case res := <-done:
		if res.consumed && res.err == nil {
			*v = res.val
		}
		return res.consumed, false, res.err
	case <-time.After(timeout):
		return false, true, nil
	}
}

// readStdinJSONFrom is the testable core of readStdinJSON. It does not check
// for a TTY; callers decide when to invoke it. Reads at most maxStdinBytes+1
// to detect overflow.
func readStdinJSONFrom(r io.Reader, v any) (bool, error) {
	limited := io.LimitReader(r, maxStdinBytes+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return false, fmt.Errorf("read stdin: %w", err)
	}
	if len(b) > maxStdinBytes {
		return true, fmt.Errorf("stdin input exceeds %d bytes", maxStdinBytes)
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(b, v); err != nil {
		return true, err
	}
	return true, nil
}

// stdinInputOr is the canonical helper used by create/update RunE bodies. It
// reads stdin into target if a pipe is present; otherwise calls buildFromFlags
// to assemble the same value from CLI flags. The returned bool tells callers
// whether stdin was consumed — when false, the caller is responsible for
// enforcing any required-flag invariants (we do not use cobra.MarkFlagRequired
// on stdin-capable commands, because that runs before RunE and would reject
// piped JSON-only invocations).
func stdinInputOr[T any](target *T, buildFromFlags func() T) (consumed bool, err error) {
	consumed, timedOut, err := readStdinJSON(target)
	if err != nil {
		return consumed, newCLIError(ErrCodeValidationFailed, "invalid stdin JSON: "+err.Error(), err)
	}
	if timedOut {
		fmt.Fprintf(os.Stderr, "note: stdin appeared open but sent no data within %s; using flags instead\n", stdinReadTimeout)
	}
	if !consumed {
		*target = buildFromFlags()
	}
	return consumed, nil
}

// requireFlag is the per-field validator stdin-capable commands use in their
// flag-fallback branch. Returns a typed validation_failed CLIError so callers
// can return it directly.
func requireFlag(name, value string) error {
	if value == "" {
		return newCLIError(ErrCodeValidationFailed,
			fmt.Sprintf("flag --%s is required (or pipe JSON on stdin)", name), nil)
	}
	return nil
}

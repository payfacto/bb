package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// maxStdinBytes caps how much we will read from a piped stdin. 1 MiB is far
// larger than any legitimate Bitbucket request body and small enough to fail
// fast on a misconfigured pipe (e.g. `yes | bb pr create`).
const maxStdinBytes = 1 << 20

// readStdinJSON reads piped stdin (or returns consumed=false when stdin is a
// TTY) and unmarshals it into v. Empty stdin is treated as no-input. Bodies
// exceeding maxStdinBytes return a validation_failed-shaped error.
//
// Production callers should use this; tests should call readStdinJSONFrom
// directly with a synthetic reader.
func readStdinJSON(v any) (bool, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return false, nil
	}
	return readStdinJSONFrom(os.Stdin, v)
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
	consumed, err = readStdinJSON(target)
	if err != nil {
		return consumed, newCLIError(ErrCodeValidationFailed, "invalid stdin JSON: "+err.Error(), err)
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

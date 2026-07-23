package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type sampleInput struct {
	Title string `json:"title"`
	Count int    `json:"count"`
}

func TestReadStdinJSONFrom_ValidJSON(t *testing.T) {
	var got sampleInput
	consumed, err := readStdinJSONFrom(strings.NewReader(`{"title":"x","count":3}`), &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !consumed {
		t.Errorf("consumed = false, want true")
	}
	if got.Title != "x" || got.Count != 3 {
		t.Errorf("unexpected unmarshalled value: %+v", got)
	}
}

func TestReadStdinJSONFrom_Empty(t *testing.T) {
	var got sampleInput
	consumed, err := readStdinJSONFrom(strings.NewReader(""), &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumed {
		t.Errorf("consumed = true, want false for empty input")
	}
}

func TestReadStdinJSONFrom_Whitespace(t *testing.T) {
	var got sampleInput
	consumed, err := readStdinJSONFrom(strings.NewReader("   \n\t  "), &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumed {
		t.Errorf("consumed = true, want false for whitespace-only input")
	}
}

func TestReadStdinJSONFrom_InvalidJSON(t *testing.T) {
	var got sampleInput
	consumed, err := readStdinJSONFrom(strings.NewReader("not json"), &got)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !consumed {
		t.Errorf("consumed = false, want true (we read non-empty input)")
	}
}

func TestReadStdinJSONFrom_OverLimit(t *testing.T) {
	// Stream maxStdinBytes+1 bytes of valid-but-oversized JSON.
	oversized := bytes.Repeat([]byte("a"), maxStdinBytes+1)
	var got sampleInput
	consumed, err := readStdinJSONFrom(bytes.NewReader(oversized), &got)
	if err == nil {
		t.Fatal("expected overflow error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("expected overflow error message, got %v", err)
	}
	if !consumed {
		t.Errorf("consumed should be true on overflow path")
	}
}

func TestReadStdinJSONFrom_ErrorReader(t *testing.T) {
	var got sampleInput
	_, err := readStdinJSONFrom(errReader{}, &got)
	if err == nil {
		t.Fatal("expected read error, got nil")
	}
	if !strings.Contains(err.Error(), "read stdin") {
		t.Errorf("expected wrapped read error, got %v", err)
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (int, error) { return 0, errors.New("disk on fire") }

// TestReadStdinJSONWithTimeout_NeverWrites_TimesOutInsteadOfHanging is the
// core regression test for the indefinite-hang bug: a reader that never
// writes and never closes (simulating a pty-emulated terminal masquerading
// as a piped, non-TTY stdin) must cause readStdinJSONWithTimeout to give up
// after the timeout rather than blocking forever in io.ReadAll.
func TestReadStdinJSONWithTimeout_NeverWrites_TimesOutInsteadOfHanging(t *testing.T) {
	pr, _ := io.Pipe() // write end intentionally never written to or closed
	defer pr.Close()

	var got sampleInput
	consumed, timedOut, err := readStdinJSONWithTimeout(pr, &got, 25*time.Millisecond)

	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if consumed {
		t.Errorf("consumed = true, want false")
	}
	if !timedOut {
		t.Errorf("timedOut = false, want true")
	}
}

// TestReadStdinJSONWithTimeout_FastReader_BehavesLikeDirectRead confirms
// that a reader completing well within the timeout is indistinguishable
// from calling readStdinJSONFrom directly.
func TestReadStdinJSONWithTimeout_FastReader_BehavesLikeDirectRead(t *testing.T) {
	var got sampleInput
	consumed, timedOut, err := readStdinJSONWithTimeout(strings.NewReader(`{"title":"x","count":3}`), &got, 200*time.Millisecond)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if timedOut {
		t.Errorf("timedOut = true, want false")
	}
	if !consumed {
		t.Errorf("consumed = false, want true")
	}
	if got.Title != "x" || got.Count != 3 {
		t.Errorf("unexpected unmarshalled value: %+v", got)
	}
}

// TestReadStdinJSONWithTimeout_SlowButWithinDeadline_NotClipped proves a
// genuinely slow-but-real pipe is not punished by the bound: data arriving
// comfortably before the deadline must still be read in full.
func TestReadStdinJSONWithTimeout_SlowButWithinDeadline_NotClipped(t *testing.T) {
	pr, pw := io.Pipe()
	go func() {
		time.Sleep(10 * time.Millisecond)
		_, _ = pw.Write([]byte(`{"title":"slow","count":7}`))
		pw.Close()
	}()

	var got sampleInput
	consumed, timedOut, err := readStdinJSONWithTimeout(pr, &got, 100*time.Millisecond)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if timedOut {
		t.Errorf("timedOut = true, want false")
	}
	if !consumed {
		t.Errorf("consumed = false, want true")
	}
	if got.Title != "slow" || got.Count != 7 {
		t.Errorf("unexpected unmarshalled value: %+v", got)
	}
}

// TestReadStdinJSONWithTimeout_TimeoutThenLateData_DoesNotOverwriteTarget
// closes a race flagged in code review: a genuinely-piped (non-TTY) producer
// that simply takes longer than the timeout to send its first byte (unlike
// the never-writes MinTTY case above, this goroutine DOES eventually
// complete) must never retroactively write into the caller's target once the
// caller has already moved on past the timeout. Run with -race: before the
// fix, the abandoned goroutine decoded straight into the caller's *v, so a
// late arrival here would race with (and clobber) whatever the caller did
// with got after readStdinJSONWithTimeout returned.
func TestReadStdinJSONWithTimeout_TimeoutThenLateData_DoesNotOverwriteTarget(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	go func() {
		time.Sleep(40 * time.Millisecond) // arrives well after the 10ms timeout below
		_, _ = pw.Write([]byte(`{"title":"late","count":99}`))
		pw.Close()
	}()

	got := sampleInput{Title: "sentinel", Count: -1} // stands in for a caller's flag-built value
	consumed, timedOut, err := readStdinJSONWithTimeout(pr, &got, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumed {
		t.Errorf("consumed = true, want false (timed out before data arrived)")
	}
	if !timedOut {
		t.Errorf("timedOut = false, want true")
	}

	// Give the abandoned goroutine time to actually complete its read, to
	// prove it does NOT retroactively overwrite got once we've moved on.
	time.Sleep(80 * time.Millisecond)
	if got.Title != "sentinel" || got.Count != -1 {
		t.Errorf("target was mutated by the abandoned goroutine after timeout: got %+v, want unchanged sentinel value", got)
	}
}

func TestRequireFlag(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		err := requireFlag("title", "")
		var cli *CLIError
		if !errors.As(err, &cli) {
			t.Fatalf("expected *CLIError, got %T", err)
		}
		if cli.Code != ErrCodeValidationFailed {
			t.Errorf("Code = %q, want %q", cli.Code, ErrCodeValidationFailed)
		}
		if !strings.Contains(cli.Message, "--title") {
			t.Errorf("Message %q should mention --title", cli.Message)
		}
	})
	t.Run("present", func(t *testing.T) {
		if err := requireFlag("title", "set"); err != nil {
			t.Errorf("unexpected error for non-empty value: %v", err)
		}
	})
}

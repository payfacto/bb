package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
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

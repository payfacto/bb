package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestAppendRequestLogWritesOneJSONLinePerCall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bb-requests.jsonl")
	entry := requestLogEntry{
		Time:        time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
		Command:     "bb pr wait-comments",
		Version:     "v0.21.0",
		Workspace:   "ws",
		Repo:        "repo",
		DurationSec: 12.5,
		ExitCode:    1,
		RequestStats: bitbucket.RequestStats{
			Requests:         4,
			RateLimited:      1,
			RateLimitHeaders: map[string]string{"X-Ratelimit-Limit": "1000"},
		},
	}

	for range 2 {
		if err := appendRequestLog(path, entry); err != nil {
			t.Fatalf("appendRequestLog: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), data)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &got); err != nil {
		t.Fatalf("line is not JSON: %v", err)
	}
	want := map[string]any{
		"time": "2026-11-22T10:00:00Z", "command": "bb pr wait-comments", "version": "v0.21.0",
		"workspace": "ws", "repo": "repo", "duration_seconds": 12.5, "exit_code": float64(1),
		"requests": float64(4), "rate_limited": float64(1),
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	headers, _ := got["rate_limit_headers"].(map[string]any)
	if headers["X-Ratelimit-Limit"] != "1000" {
		t.Errorf("rate_limit_headers = %v", got["rate_limit_headers"])
	}
}

func TestAppendRequestLogFileIsOwnerOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.jsonl")
	if err := appendRequestLog(path, requestLogEntry{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestLogRequestsToleratesUnknownCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.jsonl")
	t.Setenv(requestLogEnv, path)

	logRequests(nil, time.Now(), 1)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"exit_code":1`) {
		t.Errorf("log line = %q, want exit_code 1", data)
	}
}

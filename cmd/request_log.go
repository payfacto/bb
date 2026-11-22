package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// requestLogEnv names the file that every bb invocation appends one JSON line
// to, recording how many Bitbucket requests it sent. Unset means no logging.
const requestLogEnv = "BB_REQUEST_LOG"

// requestLogEntry is one line of the request log: what ran and what it cost
// in API requests. It never holds credentials, arguments or response data.
type requestLogEntry struct {
	Time        time.Time `json:"time"`
	Command     string    `json:"command"`
	Version     string    `json:"version"`
	Workspace   string    `json:"workspace,omitempty"`
	Repo        string    `json:"repo,omitempty"`
	DurationSec float64   `json:"duration_seconds"`
	ExitCode    int       `json:"exit_code"`
	bitbucket.RequestStats
}

// logRequests appends this invocation to the request log when BB_REQUEST_LOG
// is set. A failure to write is reported on stderr and never changes the
// command's outcome.
func logRequests(executed *cobra.Command, start time.Time, code int) {
	path := os.Getenv(requestLogEnv)
	if path == "" {
		return
	}
	entry := requestLogEntry{
		Time:        start.UTC(),
		Command:     commandPath(executed),
		Version:     Version,
		DurationSec: time.Since(start).Round(time.Millisecond).Seconds(),
		ExitCode:    code,
	}
	if cfg != nil {
		entry.Workspace, entry.Repo = cfg.Workspace, cfg.Repo
	}
	if client != nil {
		entry.RequestStats = client.Stats()
	}
	if err := appendRequestLog(path, entry); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s: %v\n", requestLogEnv, err)
	}
}

// commandPath names the command that ran, or "" when Cobra could not resolve
// one, so logging never panics after the real error has been rendered.
func commandPath(executed *cobra.Command) string {
	if executed == nil {
		return ""
	}
	return executed.CommandPath()
}

// appendRequestLog writes entry as one JSON line at the end of path, creating
// the file readable by its owner only. The line goes out in a single write so
// concurrent bb processes do not interleave.
func appendRequestLog(path string, entry requestLogEntry) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

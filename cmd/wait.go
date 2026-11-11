package cmd

import (
	"context"
	"os"
	"os/signal"
)

// Process exit codes of the wait/watch commands. They agree wherever the
// meaning overlaps: 0 success, 1 failed, 3 timeout (130 is exitInterrupted).
const (
	exitWaitFailed  = 1
	exitManualGate  = 2 // pipeline watch: paused on a manual step
	exitWaitTimeout = 3
	exitNoChecks    = 4 // pr wait-checks: no build status ever appeared
)

// waitInsteadOfLoops is the help-text guidance shared by every wait/watch
// command, so agents reach for them instead of hand-rolled polling loops.
const waitInsteadOfLoops = "To wait on a pipeline, PR build checks, or PR comments, use " +
	"`bb pipeline watch`, `bb pr wait-checks`, or `bb pr wait-comments`; do not write polling loops."

// interruptibleContext returns a context cancelled by Ctrl-C, so a long wait
// unwinds cleanly and exits through watchErr's interrupt path.
func interruptibleContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

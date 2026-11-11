package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

// Exit codes shared by the wait commands; they match `pipeline watch` where the
// meaning overlaps (0 success, 1 failed, 3 timeout).
const (
	exitWaitFailed  = 1
	exitWaitTimeout = 3
	exitNoChecks    = 4
)

// interruptibleContext returns a context cancelled by Ctrl-C, so a long wait
// unwinds cleanly and exits through watchErr's interrupt path.
func interruptibleContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

var (
	prWaitChecksID       int
	prWaitChecksInterval int
	prWaitChecksTimeout  int
	prWaitChecksAppear   int
	prWaitChecksEvents   bool
)

var prWaitChecksCmd = &cobra.Command{
	Use:   "wait-checks",
	Short: "Wait until every build status on a pull request is terminal",
	Long: "Wait until every build status on a pull request is terminal (none INPROGRESS).\n\n" +
		"Exit codes: 0 all successful, 1 any failed or stopped, 3 timeout, " +
		"4 no status appeared within --appear-timeout (the repository likely has no PR pipeline), " +
		"130 interrupted.\n\n" +
		"With --events, stdout carries one `check=NAME state=STATE` line per status change " +
		"and a final `pr=ID outcome=success|failed|timeout|none` line instead of the JSON result.\n\n" +
		waitInsteadOfLoops,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		ctx, stop := interruptibleContext()
		defer stop()
		opts := bitbucket.WaitChecksOptions{
			Interval:      time.Duration(prWaitChecksInterval) * time.Second,
			Timeout:       time.Duration(prWaitChecksTimeout) * time.Second,
			AppearTimeout: time.Duration(prWaitChecksAppear) * time.Second,
		}
		if prWaitChecksEvents {
			opts.OnPoll = newChangeEvents(os.Stdout).checks
		}
		result, err := client.PRs(ws, repo).WaitChecks(ctx, prWaitChecksID, opts)
		if err != nil {
			return watchErr(err)
		}
		exitCode = checksExitCode(result.Status)
		if prWaitChecksEvents {
			fmt.Fprintln(os.Stdout, formatEvent(
				eventField{"pr", strconv.Itoa(prWaitChecksID)},
				eventField{"outcome", string(result.Status)},
			))
			return nil
		}
		return printOutput(result, func() { render.PRStatuses(result.Statuses) })
	},
}

// checksExitCode maps a wait-checks outcome to the process exit code.
func checksExitCode(status bitbucket.ChecksWaitStatus) int {
	switch status {
	case bitbucket.ChecksFailed:
		return exitWaitFailed
	case bitbucket.ChecksTimeout:
		return exitWaitTimeout
	case bitbucket.ChecksNone:
		return exitNoChecks
	default:
		return 0
	}
}

func init() {
	prWaitChecksCmd.Flags().IntVarP(&prWaitChecksID, "pr-id", "p", 0, "pull request ID")
	prWaitChecksCmd.MarkFlagRequired("pr-id")
	prWaitChecksCmd.Flags().IntVar(&prWaitChecksInterval, "interval", 10, "seconds between polls")
	prWaitChecksCmd.Flags().IntVar(&prWaitChecksTimeout, "timeout", 0, "seconds before giving up (0 = no timeout)")
	prWaitChecksCmd.Flags().IntVar(&prWaitChecksAppear, "appear-timeout", 600,
		"seconds to wait for the first status to appear before reporting none (0 = wait as long as --timeout)")
	prWaitChecksCmd.Flags().BoolVar(&prWaitChecksEvents, "events", false,
		"print one stdout line per status change and a final outcome line instead of the JSON result")
	prCmd.AddCommand(prWaitChecksCmd)
}

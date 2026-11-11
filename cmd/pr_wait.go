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

var (
	prWaitCommentsID       int
	prWaitCommentsAuthor   string
	prWaitCommentsSince    string
	prWaitCommentsSettle   int
	prWaitCommentsInterval int
	prWaitCommentsTimeout  int
	prWaitCommentsEvents   bool
)

var prWaitCommentsCmd = &cobra.Command{
	Use:   "wait-comments",
	Short: "Wait until a matching new comment appears on a pull request",
	Long: "Wait until a comment matching the filters appears on a pull request, then print it.\n\n" +
		"By default only comments posted after the wait starts count. --since ID counts " +
		"comments with a larger ID (--since 0 = any existing comment); --since TIME " +
		"(RFC3339) counts comments created after that time. --author-contains matches a " +
		"case-insensitive substring of the author's display name or nickname (e.g. PETRA). " +
		"--settle N keeps waiting until no new match has arrived for N seconds, so a " +
		"reviewer's burst of comments comes back together.\n\n" +
		"Exit codes: 0 found, 3 timeout, 130 interrupted.\n\n" +
		"With --events, stdout carries one `comment=ID author=NAME [parent=ID] [path=FILE]` " +
		"line per match and a final `pr=ID outcome=found|timeout count=N` line instead of " +
		"the JSON result.\n\n" +
		waitInsteadOfLoops,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		opts := bitbucket.WaitCommentsOptions{
			Interval: time.Duration(prWaitCommentsInterval) * time.Second,
			Timeout:  time.Duration(prWaitCommentsTimeout) * time.Second,
			Settle:   time.Duration(prWaitCommentsSettle) * time.Second,
			Filter:   bitbucket.CommentFilter{AuthorContains: prWaitCommentsAuthor},
		}
		if err := applySince(&opts, prWaitCommentsSince); err != nil {
			return err
		}
		ctx, stop := interruptibleContext()
		defer stop()
		result, err := client.Comments(ws, repo, prWaitCommentsID).WaitFor(ctx, opts)
		if err != nil {
			return watchErr(err)
		}
		exitCode = commentsExitCode(result.Status)
		if prWaitCommentsEvents {
			for _, c := range result.Comments {
				fmt.Fprintln(os.Stdout, commentEventLine(c))
			}
			fmt.Fprintln(os.Stdout, formatEvent(
				eventField{"pr", strconv.Itoa(prWaitCommentsID)},
				eventField{"outcome", string(result.Status)},
				eventField{"count", strconv.Itoa(len(result.Comments))},
			))
			return nil
		}
		return printOutput(result, func() { render.CommentList(result.Comments) })
	},
}

// applySince sets the comment baseline from --since: empty means only comments
// posted after the wait starts, an integer is a comment ID to count after, and
// an RFC3339 time counts comments created after it.
func applySince(opts *bitbucket.WaitCommentsOptions, since string) error {
	if since == "" {
		opts.NewOnly = true
		return nil
	}
	if id, err := strconv.Atoi(since); err == nil {
		if id < 0 {
			return newCLIError(ErrCodeValidationFailed, "--since comment ID must not be negative", nil)
		}
		opts.Filter.AfterID = id
		return nil
	}
	t, err := time.Parse(time.RFC3339, since)
	if err != nil {
		return newCLIError(ErrCodeValidationFailed,
			"--since must be a comment ID or an RFC3339 time (e.g. 2026-11-10T10:00:00Z)", nil)
	}
	opts.Filter.After = t
	return nil
}

// commentEventLine is the --events line for one matching comment.
func commentEventLine(c bitbucket.Comment) string {
	fields := []eventField{{"comment", strconv.Itoa(c.ID)}, {"author", c.User.DisplayName}}
	if c.Parent != nil {
		fields = append(fields, eventField{"parent", strconv.Itoa(c.Parent.ID)})
	}
	if c.Inline != nil {
		fields = append(fields, eventField{"path", c.Inline.Path})
	}
	return formatEvent(fields...)
}

// commentsExitCode maps a wait-comments outcome to the process exit code.
func commentsExitCode(status bitbucket.CommentsWaitStatus) int {
	if status == bitbucket.CommentsTimeout {
		return exitWaitTimeout
	}
	return 0
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

	prWaitCommentsCmd.Flags().IntVarP(&prWaitCommentsID, "pr-id", "p", 0, "pull request ID")
	prWaitCommentsCmd.MarkFlagRequired("pr-id")
	prWaitCommentsCmd.Flags().StringVar(&prWaitCommentsAuthor, "author-contains", "",
		"only comments whose author display name or nickname contains this (case-insensitive)")
	prWaitCommentsCmd.Flags().StringVar(&prWaitCommentsSince, "since", "",
		"comment ID or RFC3339 time to count comments after (default: only comments posted after the wait starts)")
	prWaitCommentsCmd.Flags().IntVar(&prWaitCommentsSettle, "settle", 0,
		"after the first match, keep waiting until no new match arrives for this many seconds")
	prWaitCommentsCmd.Flags().IntVar(&prWaitCommentsInterval, "interval", 15, "seconds between polls")
	prWaitCommentsCmd.Flags().IntVar(&prWaitCommentsTimeout, "timeout", 0, "seconds before giving up (0 = no timeout)")
	prWaitCommentsCmd.Flags().BoolVar(&prWaitCommentsEvents, "events", false,
		"print one stdout line per matching comment and a final outcome line instead of the JSON result")

	prCmd.AddCommand(prWaitChecksCmd, prWaitCommentsCmd)
}

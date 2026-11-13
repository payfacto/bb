package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/payfacto/bb/pkg/bitbucket"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	stopPausedDryRun bool
	stopPausedYes    bool
	stopPausedWait   time.Duration
)

var pipelineStopPausedCmd = &cobra.Command{
	Use:   "stop-paused",
	Short: "Stop every paused pipeline older than the newest active one",
	Long: "Find the newest pipeline in the repository (any branch) that has not\n" +
		"completed - running, paused or queued - and keep it, plus the newest\n" +
		"paused pipeline (the one ready to deploy). Stop every older pipeline\n" +
		"paused at a manual or system gate, like the web UI's Close.\n" +
		"Older pipelines that are still running and completed pipelines are never\n" +
		"touched, and a pipeline resumed since the scan is skipped.\n\n" +
		"Scans the newest 250 pipelines. Prompts before stopping; pass --yes to skip\n" +
		"the prompt (required when stdin is not a terminal). Exits 1 when any\n" +
		"pipeline failed to stop.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		ctx, stop := interruptibleContext()
		defer stop()
		res := client.Pipelines(ws, repo)
		found, err := res.FindSupersededPaused(ctx, bitbucket.SupersededScanPages)
		if err != nil {
			return err
		}
		if found.Truncated {
			fmt.Fprintf(os.Stderr, "note: scanned the newest %d pipelines only; older paused pipelines were not checked\n", found.Scanned)
		}
		report := bitbucket.NewStopPausedReport(found)
		report.DryRun = stopPausedDryRun
		if stopPausedDryRun || len(found.Targets) == 0 {
			return printStopPausedReport(report)
		}
		if err := confirmStopPaused(found); err != nil {
			return err
		}
		opts := bitbucket.StopOptions{Interval: stopPollInterval, Timeout: stopPausedWait}
		report.StopPausedBatch = res.StopPaused(ctx, found.Targets, opts)
		if report.HasFailures() {
			exitCode = 1
		}
		return printStopPausedReport(report)
	},
}

// confirmStopPaused asks before stopping unless --yes was passed. Off a
// terminal there is nobody to ask, so it refuses instead of hanging.
func confirmStopPaused(found bitbucket.SupersededPaused) error {
	if stopPausedYes {
		return nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return newCLIError(ErrCodeValidationFailed,
			"stdin is not a terminal: pass --yes to stop without a prompt, or --dry-run to only list", nil)
	}
	describeStopPausedPlan(os.Stderr, found)
	fmt.Fprintf(os.Stderr, "Stop %d pipelines? [y/N] ", len(found.Targets))
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read confirmation: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(answer), "y") {
		return newCLIError(ErrCodeValidationFailed, "aborted: nothing was stopped", nil)
	}
	return nil
}

// describeStopPausedPlan lists the kept pipeline and the targets, one per line.
func describeStopPausedPlan(w io.Writer, found bitbucket.SupersededPaused) {
	if found.Kept == nil {
		fmt.Fprintln(w, "No active pipeline: nothing to stop.")
		return
	}
	describeKept(w, found)
	for _, p := range found.Targets {
		fmt.Fprintf(w, "Stop  #%d %s (%s)\n", p.BuildNumber, p.Branch, stateLabel(p.State))
	}
}

// describeKept lists the pipelines that are never stopped.
func describeKept(w io.Writer, found bitbucket.SupersededPaused) {
	for _, p := range []*bitbucket.Pipeline{found.Kept, found.KeptPaused} {
		if p != nil {
			fmt.Fprintf(w, "Keep  #%d %s (%s)\n", p.BuildNumber, p.Branch, stateLabel(p.State))
		}
	}
}

// stateLabel is the state name, with the stage when Bitbucket sent one.
func stateLabel(state bitbucket.PipelineState) string {
	if state.Stage == nil {
		return state.Name
	}
	return state.Name + "/" + state.Stage.Name
}

func printStopPausedReport(report bitbucket.StopPausedReport) error {
	return printOutput(report, func() {
		if report.DryRun || report.Kept == nil {
			describeStopPausedPlan(os.Stdout, report.SupersededPaused)
			return
		}
		describeKept(os.Stdout, report.SupersededPaused)
		for _, r := range report.Stopped {
			fmt.Printf("#%d %s (%s)\n", r.BuildNumber, r.Result, r.State.Name)
		}
		for _, i := range report.Skipped {
			fmt.Printf("#%d skipped: %s\n", i.BuildNumber, i.Reason)
		}
		for _, i := range report.Failed {
			fmt.Printf("#%d failed: %s\n", i.BuildNumber, i.Reason)
		}
	})
}

func init() {
	pipelineStopPausedCmd.Flags().BoolVar(&stopPausedDryRun, "dry-run", false, "list the pipelines that would be kept and stopped, then exit")
	pipelineStopPausedCmd.Flags().BoolVarP(&stopPausedYes, "yes", "y", false, "stop without asking for confirmation")
	pipelineStopPausedCmd.Flags().DurationVar(&stopPausedWait, "wait", defaultStopWait, "how long to wait for each pipeline to complete (0 = do not wait)")
	pipelineCmd.AddCommand(pipelineStopPausedCmd)
}

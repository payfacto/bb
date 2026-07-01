package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage Bitbucket Pipelines",
}

// pipelineSelector identifies a pipeline by its UUID, its integer build number,
// or (watch only) its branch. get/stop/steps/log require exactly one of
// uuid/build; watch allows at most one of uuid/build/branch (none = latest).
type pipelineSelector struct {
	uuid   string
	build  int
	branch string
}

// validate enforces that exactly one of the UUID / build-number selectors is
// set. A build number counts as "set" when it is > 0: Bitbucket build numbers
// start at 1, so 0 is the flag's unset zero value.
func (s pipelineSelector) validate() error {
	hasUUID := s.uuid != ""
	hasBuild := s.build > 0
	switch {
	case hasUUID && hasBuild:
		return newCLIError(ErrCodeValidationFailed,
			"--pipeline-uuid and --build-number are mutually exclusive; set exactly one", nil)
	case !hasUUID && !hasBuild:
		return newCLIError(ErrCodeValidationFailed,
			"one of --pipeline-uuid or --build-number is required", nil)
	}
	return nil
}

// resolvePipeline validates the selector and fetches the full pipeline it
// refers to, addressing it by build number or UUID as appropriate.
func (s pipelineSelector) resolvePipeline(ctx context.Context, res *bitbucket.PipelineResource) (bitbucket.Pipeline, error) {
	if err := s.validate(); err != nil {
		return bitbucket.Pipeline{}, err
	}
	if s.build > 0 {
		return res.GetByBuildNumber(ctx, s.build)
	}
	return res.Get(ctx, s.uuid)
}

// resolveUUID validates the selector and resolves it to a pipeline UUID. A
// build number is resolved with one fetch; a UUID is returned directly (no
// fetch), so the stop/steps/log commands keep their existing single-request
// behavior when addressed by UUID before calling the UUID-based methods.
func (s pipelineSelector) resolveUUID(ctx context.Context, res *bitbucket.PipelineResource) (string, error) {
	if err := s.validate(); err != nil {
		return "", err
	}
	if s.uuid != "" {
		return s.uuid, nil
	}
	p, err := res.GetByBuildNumber(ctx, s.build)
	if err != nil {
		return "", err
	}
	return p.UUID, nil
}

// validateWatchSelector allows at most one of uuid/build/branch. When none is
// set, watch targets the most recent pipeline in the repository.
func (s pipelineSelector) validateWatchSelector() error {
	set := 0
	if s.uuid != "" {
		set++
	}
	if s.build > 0 {
		set++
	}
	if s.branch != "" {
		set++
	}
	if set > 1 {
		return newCLIError(ErrCodeValidationFailed,
			"--pipeline-uuid, --build-number, and --branch are mutually exclusive", nil)
	}
	return nil
}

// resolveWatchPipeline picks the pipeline to watch: build/uuid address one
// directly, otherwise the latest pipeline (on branch, if set; repo-wide if not).
func (s pipelineSelector) resolveWatchPipeline(ctx context.Context, res *bitbucket.PipelineResource) (bitbucket.Pipeline, error) {
	if err := s.validateWatchSelector(); err != nil {
		return bitbucket.Pipeline{}, err
	}
	switch {
	case s.build > 0:
		return res.GetByBuildNumber(ctx, s.build)
	case s.uuid != "":
		return res.Get(ctx, s.uuid)
	default:
		return res.Latest(ctx, s.branch)
	}
}

// watchExitCode maps a terminal watch status to the process exit code, so that
// `bb pipeline watch ... && next` proceeds only on success (0). Failed is 1,
// a blocked manual gate is 2, and a timeout is 3.
func watchExitCode(status bitbucket.PipelineWatchStatus) int {
	switch status {
	case bitbucket.WatchFailed:
		return 1
	case bitbucket.WatchBlocked:
		return 2
	case bitbucket.WatchTimeout:
		return 3
	default:
		return 0
	}
}

// exitInterrupted is the conventional exit code for a process ended by SIGINT
// (128 + 2). It is distinct from watch's own 0-3 outcome codes.
const exitInterrupted = 130

// watchErr resolves an error from the watch flow. A user interrupt (Ctrl-C
// cancels ctx) is not a failure: it reports on stderr and requests exit 130
// with no error envelope. Any other error is returned unchanged for normal
// mapping. It sets the package-level exitCode on interrupt.
func watchErr(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		fmt.Fprintln(os.Stderr, "watch canceled")
		exitCode = exitInterrupted
		return nil
	}
	return err
}

var pipelineListSort string

var pipelineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent pipelines, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		pipelines, err := client.Pipelines(ws, repo).List(context.Background(), pipelineListSort)
		if err != nil {
			return err
		}
		return printOutput(pipelines, func() { render.PipelineList(pipelines) })
	},
}

var (
	pipelineGetUUID  string
	pipelineGetBuild int
)

var pipelineGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get pipeline details",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		sel := pipelineSelector{uuid: pipelineGetUUID, build: pipelineGetBuild}
		p, err := sel.resolvePipeline(context.Background(), client.Pipelines(ws, repo))
		if err != nil {
			return err
		}
		return printOutput(p, func() { render.PipelineDetail(p) })
	},
}

var pipelineTriggerBranch string

var pipelineTriggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Trigger a new pipeline on a branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		p, err := client.Pipelines(ws, repo).Trigger(context.Background(), pipelineTriggerBranch)
		if err != nil {
			return err
		}
		return printOutput(p, func() {
			fmt.Printf("Pipeline #%d triggered on branch '%s'\nUUID: %s\n",
				p.BuildNumber, pipelineTriggerBranch, p.UUID)
		})
	},
}

var (
	pipelineStopUUID  string
	pipelineStopBuild int
)

var pipelineStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		ctx := context.Background()
		res := client.Pipelines(ws, repo)
		sel := pipelineSelector{uuid: pipelineStopUUID, build: pipelineStopBuild}
		uuid, err := sel.resolveUUID(ctx, res)
		if err != nil {
			return err
		}
		if err := res.Stop(ctx, uuid); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "stopped", "uuid": uuid}, func() {
			fmt.Printf("Pipeline %s stopped.\n", uuid)
		})
	},
}

var (
	pipelineStepsUUID  string
	pipelineStepsBuild int
)

var pipelineStepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "List steps of a pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		ctx := context.Background()
		res := client.Pipelines(ws, repo)
		sel := pipelineSelector{uuid: pipelineStepsUUID, build: pipelineStepsBuild}
		uuid, err := sel.resolveUUID(ctx, res)
		if err != nil {
			return err
		}
		steps, err := res.Steps(ctx, uuid)
		if err != nil {
			return err
		}
		return printOutput(steps, func() { render.PipelineSteps(steps) })
	},
}

var (
	pipelineLogPipelineUUID string
	pipelineLogBuild        int
	pipelineLogStepUUID     string
)

var pipelineLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Get log output for a pipeline step (always plain text)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		ctx := context.Background()
		res := client.Pipelines(ws, repo)
		sel := pipelineSelector{uuid: pipelineLogPipelineUUID, build: pipelineLogBuild}
		uuid, err := sel.resolveUUID(ctx, res)
		if err != nil {
			return err
		}
		log, err := res.Log(ctx, uuid, pipelineLogStepUUID)
		if err != nil {
			return err
		}
		fmt.Print(log)
		return nil
	},
}

var (
	pipelineWatchUUID     string
	pipelineWatchBuild    int
	pipelineWatchBranch   string
	pipelineWatchTailLog  bool
	pipelineWatchInterval int
	pipelineWatchTimeout  int
)

var pipelineWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch a pipeline until it reaches a terminal state",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		// A watch can run for a long time; make Ctrl-C cancel it cleanly rather
		// than hard-killing the process, so the poll loop unwinds and we exit
		// with a controlled code.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		res := client.Pipelines(ws, repo)
		sel := pipelineSelector{uuid: pipelineWatchUUID, build: pipelineWatchBuild, branch: pipelineWatchBranch}
		target, err := sel.resolveWatchPipeline(ctx, res)
		if err != nil {
			return watchErr(ctx, err)
		}
		result, err := res.Watch(ctx, target.UUID, bitbucket.WatchOptions{
			Interval: time.Duration(pipelineWatchInterval) * time.Second,
			Timeout:  time.Duration(pipelineWatchTimeout) * time.Second,
			OnPoll:   watchProgress(ctx, res, pipelineWatchTailLog),
		})
		if err != nil {
			return watchErr(ctx, err)
		}
		exitCode = watchExitCode(result.Status)
		return printOutput(result, func() { render.PipelineWatch(result) })
	},
}

// watchProgress returns the poll callback for `pipeline watch`, or nil when
// there is nothing to emit. Incidental progress lines are written to stderr only
// when stderr is a TTY (so they never pollute a redirected/piped stream), while
// an explicitly requested --tail-log always streams the running step's
// newly-appended log to stderr. Both keep stdout reserved for the final result.
// Log streaming is best-effort: a step's log 404s until it produces output.
func watchProgress(ctx context.Context, res *bitbucket.PipelineResource, tailLog bool) func(bitbucket.Pipeline, []bitbucket.PipelineStep) {
	showProgress := term.IsTerminal(int(os.Stderr.Fd()))
	if !showProgress && !tailLog {
		return nil
	}
	printed := make(map[string]int) // step UUID -> bytes already streamed
	return func(p bitbucket.Pipeline, steps []bitbucket.PipelineStep) {
		if showProgress {
			fmt.Fprintf(os.Stderr, "[watch] #%d %s\n", p.BuildNumber, pipelineStateLabel(p.State))
		}
		if !tailLog {
			return
		}
		step := runningStep(steps)
		if step == nil {
			return
		}
		log, err := res.Log(ctx, p.UUID, step.UUID)
		if err != nil {
			return
		}
		if n := printed[step.UUID]; len(log) > n {
			fmt.Fprint(os.Stderr, log[n:])
			printed[step.UUID] = len(log)
		}
	}
}

// pipelineStateLabel renders a compact "NAME/STAGE RESULT" progress label.
func pipelineStateLabel(s bitbucket.PipelineState) string {
	label := s.Name
	if s.Stage != nil && s.Stage.Name != "" {
		label += "/" + s.Stage.Name
	}
	if s.Result != nil && s.Result.Name != "" {
		label += " " + s.Result.Name
	}
	return label
}

// runningStep returns the first IN_PROGRESS step, or nil when none is running.
func runningStep(steps []bitbucket.PipelineStep) *bitbucket.PipelineStep {
	for i := range steps {
		if steps[i].State.Name == "IN_PROGRESS" {
			return &steps[i]
		}
	}
	return nil
}

func init() {
	pipelineListCmd.Flags().StringVar(&pipelineListSort, "sort", "",
		"sort by Bitbucket field, prefix with - for descending (default -created_on)")

	pipelineGetCmd.Flags().StringVarP(&pipelineGetUUID, "pipeline-uuid", "u", "", "pipeline UUID (alternative to --build-number)")
	pipelineGetCmd.Flags().IntVarP(&pipelineGetBuild, "build-number", "n", 0, "pipeline build number (alternative to --pipeline-uuid)")

	pipelineTriggerCmd.Flags().StringVarP(&pipelineTriggerBranch, "branch", "b", "", "branch to trigger pipeline on (required)")
	pipelineTriggerCmd.MarkFlagRequired("branch")

	pipelineStopCmd.Flags().StringVarP(&pipelineStopUUID, "pipeline-uuid", "u", "", "pipeline UUID (alternative to --build-number)")
	pipelineStopCmd.Flags().IntVarP(&pipelineStopBuild, "build-number", "n", 0, "pipeline build number (alternative to --pipeline-uuid)")

	pipelineStepsCmd.Flags().StringVarP(&pipelineStepsUUID, "pipeline-uuid", "u", "", "pipeline UUID (alternative to --build-number)")
	pipelineStepsCmd.Flags().IntVarP(&pipelineStepsBuild, "build-number", "n", 0, "pipeline build number (alternative to --pipeline-uuid)")

	pipelineLogCmd.Flags().StringVarP(&pipelineLogPipelineUUID, "pipeline-uuid", "u", "", "pipeline UUID (alternative to --build-number)")
	pipelineLogCmd.Flags().IntVarP(&pipelineLogBuild, "build-number", "n", 0, "pipeline build number (alternative to --pipeline-uuid)")
	pipelineLogCmd.Flags().StringVar(&pipelineLogStepUUID, "step-uuid", "", "step UUID (required)")
	pipelineLogCmd.MarkFlagRequired("step-uuid")

	pipelineWatchCmd.Flags().StringVarP(&pipelineWatchUUID, "pipeline-uuid", "u", "", "watch the pipeline with this UUID")
	pipelineWatchCmd.Flags().IntVarP(&pipelineWatchBuild, "build-number", "n", 0, "watch the pipeline with this build number")
	pipelineWatchCmd.Flags().StringVarP(&pipelineWatchBranch, "branch", "b", "", "watch the latest pipeline on this branch")
	pipelineWatchCmd.Flags().BoolVar(&pipelineWatchTailLog, "tail-log", false, "stream the running step's log to stderr while polling")
	pipelineWatchCmd.Flags().IntVar(&pipelineWatchInterval, "interval", 5, "seconds between polls")
	pipelineWatchCmd.Flags().IntVar(&pipelineWatchTimeout, "timeout", 0, "seconds before giving up (0 = no timeout)")

	pipelineCmd.AddCommand(pipelineListCmd, pipelineGetCmd, pipelineTriggerCmd,
		pipelineStopCmd, pipelineStepsCmd, pipelineLogCmd, pipelineWatchCmd)
	rootCmd.AddCommand(pipelineCmd)
}

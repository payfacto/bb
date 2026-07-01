package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage Bitbucket Pipelines",
}

// pipelineSelector identifies a single pipeline by exactly one of its UUID or
// its integer build number, as supplied to the get/stop/steps/log commands.
type pipelineSelector struct {
	uuid  string
	build int
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

	pipelineCmd.AddCommand(pipelineListCmd, pipelineGetCmd, pipelineTriggerCmd,
		pipelineStopCmd, pipelineStepsCmd, pipelineLogCmd)
	rootCmd.AddCommand(pipelineCmd)
}

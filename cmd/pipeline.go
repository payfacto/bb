package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage Bitbucket Pipelines",
}

var pipelineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent pipelines, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		pipelines, err := client.Pipelines(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(pipelines, func() { render.PipelineList(pipelines) })
	},
}

var pipelineGetUUID string

var pipelineGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get pipeline details",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		p, err := client.Pipelines(ws, repo).Get(context.Background(), pipelineGetUUID)
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

var pipelineStopUUID string

var pipelineStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Pipelines(ws, repo).Stop(context.Background(), pipelineStopUUID); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "stopped", "uuid": pipelineStopUUID}, func() {
			fmt.Printf("Pipeline %s stopped.\n", pipelineStopUUID)
		})
	},
}

var pipelineStepsUUID string

var pipelineStepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "List steps of a pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		steps, err := client.Pipelines(ws, repo).Steps(context.Background(), pipelineStepsUUID)
		if err != nil {
			return err
		}
		return printOutput(steps, func() { render.PipelineSteps(steps) })
	},
}

var (
	pipelineLogPipelineUUID string
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
		log, err := client.Pipelines(ws, repo).Log(context.Background(), pipelineLogPipelineUUID, pipelineLogStepUUID)
		if err != nil {
			return err
		}
		fmt.Print(log)
		return nil
	},
}

func init() {
	pipelineGetCmd.Flags().StringVarP(&pipelineGetUUID, "pipeline-uuid", "u", "", "pipeline UUID (required)")
	pipelineGetCmd.MarkFlagRequired("pipeline-uuid")

	pipelineTriggerCmd.Flags().StringVarP(&pipelineTriggerBranch, "branch", "b", "", "branch to trigger pipeline on (required)")
	pipelineTriggerCmd.MarkFlagRequired("branch")

	pipelineStopCmd.Flags().StringVarP(&pipelineStopUUID, "pipeline-uuid", "u", "", "pipeline UUID (required)")
	pipelineStopCmd.MarkFlagRequired("pipeline-uuid")

	pipelineStepsCmd.Flags().StringVarP(&pipelineStepsUUID, "pipeline-uuid", "u", "", "pipeline UUID (required)")
	pipelineStepsCmd.MarkFlagRequired("pipeline-uuid")

	pipelineLogCmd.Flags().StringVarP(&pipelineLogPipelineUUID, "pipeline-uuid", "u", "", "pipeline UUID (required)")
	pipelineLogCmd.Flags().StringVar(&pipelineLogStepUUID, "step-uuid", "", "step UUID (required)")
	pipelineLogCmd.MarkFlagRequired("pipeline-uuid")
	pipelineLogCmd.MarkFlagRequired("step-uuid")

	pipelineCmd.AddCommand(pipelineListCmd, pipelineGetCmd, pipelineTriggerCmd,
		pipelineStopCmd, pipelineStepsCmd, pipelineLogCmd)
	rootCmd.AddCommand(pipelineCmd)
}

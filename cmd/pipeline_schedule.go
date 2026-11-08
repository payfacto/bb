package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	scheduleRefTypeBranch   = "branch"
	scheduleSelectorDefault = "branches"
)

const cronHelp = `The cron expression has 7 fields with second precision and is evaluated in UTC:

  seconds minutes hours day-of-month month day-of-week year

Use "?" for whichever of day-of-month / day-of-week is not set. Examples:

  "0 0 12 * * ? *"     every day at 12:00 UTC
  "0 30 2 ? * MON *"   every Monday at 02:30 UTC
  "0 0 0/6 * * ? *"    every 6 hours`

var pipelineScheduleCmd = &cobra.Command{
	Use:   "pipeline-schedule",
	Short: "Manage scheduled (cron) pipeline runs",
}

var pipelineScheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pipeline schedules for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		schedules, err := client.PipelineSchedules(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(schedules, func() { render.PipelineScheduleList(schedules) })
	},
}

var pipelineScheduleGetUUID string

var pipelineScheduleGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a pipeline schedule by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		s, err := client.PipelineSchedules(ws, repo).Get(context.Background(), pipelineScheduleGetUUID)
		if err != nil {
			return err
		}
		return printOutput(s, func() { render.PipelineScheduleDetail(s) })
	},
}

// scheduleCreateFlags holds the `pipeline-schedule create` flag values.
type scheduleCreateFlags struct {
	branch       string
	cron         string
	selectorType string
	pattern      string
	disabled     bool
}

// input builds the create request from flags. The selector defaults to the
// "branches" pipeline matching the scheduled branch, which is what the
// Bitbucket UI does. Enabled is only sent when --disabled is set, so the API
// default (enabled) applies otherwise.
func (f scheduleCreateFlags) input() bitbucket.CreatePipelineScheduleInput {
	selectorType := f.selectorType
	if selectorType == "" {
		selectorType = scheduleSelectorDefault
	}
	pattern := f.pattern
	if pattern == "" && selectorType == scheduleSelectorDefault {
		pattern = f.branch
	}
	input := bitbucket.CreatePipelineScheduleInput{
		CronPattern: f.cron,
		Target: bitbucket.PipelineScheduleTarget{
			Type:     bitbucket.PipelineRefTargetType,
			RefType:  scheduleRefTypeBranch,
			RefName:  f.branch,
			Selector: bitbucket.PipelineSelector{Type: selectorType, Pattern: pattern},
		},
	}
	if f.disabled {
		enabled := false
		input.Enabled = &enabled
	}
	return input
}

var pipelineScheduleCreate scheduleCreateFlags

var pipelineScheduleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pipeline schedule",
	Long:  "Create a schedule that runs a pipeline on a branch.\n\n" + cronHelp,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		var input bitbucket.CreatePipelineScheduleInput
		consumed, err := stdinInputOr(&input, pipelineScheduleCreate.input)
		if err != nil {
			return err
		}
		if !consumed {
			if err := requireFlag("branch", pipelineScheduleCreate.branch); err != nil {
				return err
			}
			if err := requireFlag("cron", pipelineScheduleCreate.cron); err != nil {
				return err
			}
		}
		s, err := client.PipelineSchedules(ws, repo).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(s, func() { render.PipelineScheduleDetail(s) })
	},
}

// resolveScheduleEnabled turns the mutually exclusive --enabled / --disabled
// flags into the schedule's new enabled state.
func resolveScheduleEnabled(enabled, disabled bool) (bool, error) {
	if enabled == disabled {
		return false, newCLIError(ErrCodeValidationFailed,
			"exactly one of --enabled or --disabled is required (or pipe JSON on stdin)", nil)
	}
	return enabled, nil
}

var (
	pipelineScheduleUpdateUUID     string
	pipelineScheduleUpdateEnabled  bool
	pipelineScheduleUpdateDisabled bool
)

var pipelineScheduleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Enable or disable a pipeline schedule",
	Long: "Enable or disable a pipeline schedule. Bitbucket only allows the enabled state to change;\n" +
		"to change the branch, pipeline, or cron expression, delete the schedule and create a new one.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := requireFlag("uuid", pipelineScheduleUpdateUUID); err != nil {
			return err
		}
		var input bitbucket.UpdatePipelineScheduleInput
		consumed, err := stdinInputOr(&input, func() bitbucket.UpdatePipelineScheduleInput {
			return bitbucket.UpdatePipelineScheduleInput{}
		})
		if err != nil {
			return err
		}
		if !consumed {
			enabled, err := resolveScheduleEnabled(pipelineScheduleUpdateEnabled, pipelineScheduleUpdateDisabled)
			if err != nil {
				return err
			}
			input.Enabled = enabled
		}
		s, err := client.PipelineSchedules(ws, repo).Update(context.Background(), pipelineScheduleUpdateUUID, input)
		if err != nil {
			return err
		}
		return printOutput(s, func() { render.PipelineScheduleDetail(s) })
	},
}

var pipelineScheduleDeleteUUID string

var pipelineScheduleDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a pipeline schedule by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.PipelineSchedules(ws, repo).Delete(context.Background(), pipelineScheduleDeleteUUID); err != nil {
			return err
		}
		return printOutput(map[string]any{"deleted": true, "uuid": pipelineScheduleDeleteUUID}, func() {
			fmt.Printf("Pipeline schedule %s deleted\n", pipelineScheduleDeleteUUID)
		})
	},
}

var pipelineScheduleExecutionsUUID string

var pipelineScheduleExecutionsCmd = &cobra.Command{
	Use:   "executions",
	Short: "List the recorded runs of a pipeline schedule",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		executions, err := client.PipelineSchedules(ws, repo).Executions(context.Background(), pipelineScheduleExecutionsUUID)
		if err != nil {
			return err
		}
		return printOutput(executions, func() { render.PipelineScheduleExecutionList(executions) })
	},
}

func init() {
	pipelineScheduleGetCmd.Flags().StringVar(&pipelineScheduleGetUUID, "uuid", "", "schedule UUID (required)")
	pipelineScheduleGetCmd.MarkFlagRequired("uuid")

	createFlags := pipelineScheduleCreateCmd.Flags()
	createFlags.StringVarP(&pipelineScheduleCreate.branch, "branch", "b", "", "branch to run the pipeline on (required)")
	createFlags.StringVar(&pipelineScheduleCreate.cron, "cron", "", `7-field UTC cron expression, e.g. "0 0 12 * * ? *" (required)`)
	createFlags.StringVar(&pipelineScheduleCreate.selectorType, "selector-type", "",
		"pipeline definition type: branches | tags | bookmarks | default | custom (default branches)")
	createFlags.StringVar(&pipelineScheduleCreate.pattern, "pattern", "",
		"pipeline definition name, e.g. the custom pipeline name (defaults to --branch for branches)")
	createFlags.BoolVar(&pipelineScheduleCreate.disabled, "disabled", false, "create the schedule disabled")
	// no MarkFlagRequired -- branch/cron are validated in RunE so stdin JSON works.

	pipelineScheduleUpdateCmd.Flags().StringVar(&pipelineScheduleUpdateUUID, "uuid", "", "schedule UUID (required)")
	pipelineScheduleUpdateCmd.Flags().BoolVar(&pipelineScheduleUpdateEnabled, "enabled", false, "enable the schedule")
	pipelineScheduleUpdateCmd.Flags().BoolVar(&pipelineScheduleUpdateDisabled, "disabled", false, "disable the schedule")
	// no MarkFlagRequired -- uuid and --enabled/--disabled are validated in RunE so stdin JSON works.

	pipelineScheduleDeleteCmd.Flags().StringVar(&pipelineScheduleDeleteUUID, "uuid", "", "schedule UUID (required)")
	pipelineScheduleDeleteCmd.MarkFlagRequired("uuid")

	pipelineScheduleExecutionsCmd.Flags().StringVar(&pipelineScheduleExecutionsUUID, "uuid", "", "schedule UUID (required)")
	pipelineScheduleExecutionsCmd.MarkFlagRequired("uuid")

	pipelineScheduleCmd.AddCommand(pipelineScheduleListCmd, pipelineScheduleGetCmd, pipelineScheduleCreateCmd,
		pipelineScheduleUpdateCmd, pipelineScheduleDeleteCmd, pipelineScheduleExecutionsCmd)
	rootCmd.AddCommand(pipelineScheduleCmd)
}

package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	scheduleUUIDWidth     = 38
	scheduleCronWidth     = 20
	scheduleBranchWidth   = 24
	schedulePipelineWidth = 24
	executionTypeWidth    = 40
)

// yesNo renders a boolean as a human-friendly yes/no.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// scheduleSelector renders the pipeline definition a schedule runs, e.g.
// "custom: nightly" or "branches: main".
func scheduleSelector(sel bitbucket.PipelineSelector) string {
	if sel.Pattern == "" {
		return sel.Type
	}
	return sel.Type + ": " + sel.Pattern
}

// PipelineScheduleListString returns the formatted text for a list of pipeline schedules.
func PipelineScheduleListString(schedules []bitbucket.PipelineSchedule) string {
	if len(schedules) == 0 {
		return "No pipeline schedules found.\n"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-*s", scheduleUUIDWidth, "UUID")),
		LabelStyle.Render(fmt.Sprintf("%-*s", scheduleCronWidth, "CRON (UTC)")),
		LabelStyle.Render(fmt.Sprintf("%-*s", scheduleBranchWidth, "BRANCH")),
		LabelStyle.Render(fmt.Sprintf("%-*s", schedulePipelineWidth, "PIPELINE")),
		LabelStyle.Render("ENABLED")))
	sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", scheduleUUIDWidth)),
		SepStyle.Render(strings.Repeat("─", scheduleCronWidth)),
		SepStyle.Render(strings.Repeat("─", scheduleBranchWidth)),
		SepStyle.Render(strings.Repeat("─", schedulePipelineWidth)),
		SepStyle.Render(strings.Repeat("─", 7))))
	for _, s := range schedules {
		sb.WriteString(fmt.Sprintf("  %s  %-*s  %s  %-*s  %s\n",
			DimStyle.Render(fmt.Sprintf("%-*s", scheduleUUIDWidth, s.UUID)),
			scheduleCronWidth, truncate(s.CronPattern, scheduleCronWidth),
			BranchStyle.Render(fmt.Sprintf("%-*s", scheduleBranchWidth, truncate(s.Target.RefName, scheduleBranchWidth))),
			schedulePipelineWidth, truncate(scheduleSelector(s.Target.Selector), schedulePipelineWidth),
			yesNo(s.Enabled)))
	}
	return sb.String()
}

// PipelineScheduleList prints the formatted pipeline schedule list to stdout.
func PipelineScheduleList(schedules []bitbucket.PipelineSchedule) {
	fmt.Print(PipelineScheduleListString(schedules))
}

// PipelineScheduleDetailString returns the full detail string for a single pipeline schedule.
func PipelineScheduleDetailString(s bitbucket.PipelineSchedule) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("UUID:      "), DimStyle.Render(s.UUID)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Cron (UTC):"), s.CronPattern))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Branch:    "), BranchStyle.Render(s.Target.RefName)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Pipeline:  "), scheduleSelector(s.Target.Selector)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Enabled:   "), yesNo(s.Enabled)))
	if s.CreatedOn != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Created:   "), s.CreatedOn))
	}
	if s.UpdatedOn != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Updated:   "), s.UpdatedOn))
	}
	return sb.String()
}

// PipelineScheduleDetail prints the formatted pipeline schedule detail to stdout.
func PipelineScheduleDetail(s bitbucket.PipelineSchedule) { fmt.Print(PipelineScheduleDetailString(s)) }

// PipelineScheduleExecutionListString returns the formatted text for a list of
// schedule executions. The API publishes no fixed execution schema, so each row
// shows the object type followed by its remaining fields as compact JSON.
func PipelineScheduleExecutionListString(executions []bitbucket.PipelineScheduleExecution) string {
	if len(executions) == 0 {
		return "No schedule executions found.\n"
	}
	var sb strings.Builder
	for _, e := range executions {
		kind, _ := e["type"].(string)
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-*s", executionTypeWidth, kind)),
			executionFields(e)))
	}
	return sb.String()
}

// executionFields returns the execution's fields other than "type" as compact JSON.
func executionFields(e bitbucket.PipelineScheduleExecution) string {
	rest := make(map[string]any, len(e))
	for k, v := range e {
		if k != "type" {
			rest[k] = v
		}
	}
	data, err := json.Marshal(rest)
	if err != nil {
		return ""
	}
	return string(data)
}

// PipelineScheduleExecutionList prints the formatted schedule execution list to stdout.
func PipelineScheduleExecutionList(executions []bitbucket.PipelineScheduleExecution) {
	fmt.Print(PipelineScheduleExecutionListString(executions))
}

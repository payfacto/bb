package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	runnerNameWidth   = 30
	runnerUUIDWidth   = 38
	runnerStatusWidth = 12
)

// yesNo renders a boolean as "yes" or "no" for detail views.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// RunnerListString returns formatted text for a list of self-hosted runners.
func RunnerListString(runners []bitbucket.PipelineRunner) string {
	if len(runners) == 0 {
		return "No runners found.\n"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-*s", runnerNameWidth, "NAME")),
		LabelStyle.Render(fmt.Sprintf("%-*s", runnerUUIDWidth, "UUID")),
		LabelStyle.Render(fmt.Sprintf("%-*s", runnerStatusWidth, "STATUS")),
		LabelStyle.Render("LABELS"))
	fmt.Fprintf(&sb, "  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", runnerNameWidth)),
		SepStyle.Render(strings.Repeat("─", runnerUUIDWidth)),
		SepStyle.Render(strings.Repeat("─", runnerStatusWidth)),
		SepStyle.Render(strings.Repeat("─", 6)))
	for _, r := range runners {
		fmt.Fprintf(&sb, "  %s  %s  %-*s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-*s", runnerNameWidth, truncate(r.Name, runnerNameWidth))),
			DimStyle.Render(fmt.Sprintf("%-*s", runnerUUIDWidth, r.UUID)),
			runnerStatusWidth, r.State.Status,
			strings.Join(r.Labels, ","))
	}
	return sb.String()
}

// RunnerList prints the formatted runner list to stdout.
func RunnerList(runners []bitbucket.PipelineRunner) { fmt.Print(RunnerListString(runners)) }

// RunnerDetailString returns formatted text for a single runner.
func RunnerDetailString(r bitbucket.PipelineRunner) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "  %s  %s\n", LabelStyle.Render("Name:    "), IDStyle.Render(r.Name))
	fmt.Fprintf(&sb, "  %s  %s\n", LabelStyle.Render("UUID:    "), DimStyle.Render(r.UUID))
	fmt.Fprintf(&sb, "  %s  %s\n", LabelStyle.Render("Status:  "), StateBadge(r.State.Status))
	fmt.Fprintf(&sb, "  %s  %s\n", LabelStyle.Render("Cordoned:"), yesNo(r.State.Cordoned))
	fmt.Fprintf(&sb, "  %s  %s\n", LabelStyle.Render("Labels:  "), strings.Join(r.Labels, ", "))
	fmt.Fprintf(&sb, "  %s  %s\n", LabelStyle.Render("Created: "), r.CreatedOn)
	fmt.Fprintf(&sb, "  %s  %s\n", LabelStyle.Render("Updated: "), r.UpdatedOn)
	return sb.String()
}

// RunnerDetail prints the formatted runner detail to stdout.
func RunnerDetail(r bitbucket.PipelineRunner) { fmt.Print(RunnerDetailString(r)) }

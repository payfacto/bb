package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// PipelineListString returns formatted text for a list of pipelines.
func PipelineListString(pipelines []bitbucket.Pipeline) string {
	if len(pipelines) == 0 {
		return "No pipelines found.\n"
	}

	header := fmt.Sprintf("  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-6s", "BUILD")),
		LabelStyle.Render(fmt.Sprintf("%-30s", "STATE")),
		LabelStyle.Render(fmt.Sprintf("%-20s", "BRANCH")),
		LabelStyle.Render("DATE"))
	divider := fmt.Sprintf("  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 6)),
		SepStyle.Render(strings.Repeat("─", 30)),
		SepStyle.Render(strings.Repeat("─", 20)),
		SepStyle.Render(strings.Repeat("─", 10)))

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)

	for _, p := range pipelines {
		id := IDStyle.Render(fmt.Sprintf("#%-4d", p.BuildNumber))
		state := p.State.Name
		if p.State.Result != nil {
			state += "/" + p.State.Result.Name
		}
		date := ""
		if len(p.CreatedOn) >= datePrefixLen {
			date = p.CreatedOn[:datePrefixLen]
		}
		sb.WriteString(fmt.Sprintf("  %s  %-30s  %-20s  %s\n",
			id, StateBadge(state), BranchStyle.Render(truncate(p.Target.RefName, 20)), DimStyle.Render(date)))
	}
	return sb.String()
}

// PipelineList prints the formatted pipeline list to stdout.
func PipelineList(pipelines []bitbucket.Pipeline) {
	fmt.Print(PipelineListString(pipelines))
}

// PipelineDetailString returns formatted text for a single pipeline.
func PipelineDetailString(p bitbucket.Pipeline) string {
	const labelW = 10

	label := func(s string) string {
		return LabelStyle.Render(fmt.Sprintf("  %-*s", labelW, s))
	}

	state := p.State.Name
	if p.State.Result != nil {
		state += " (" + p.State.Result.Name + ")"
	}
	commit := ""
	if p.Target.Commit != nil {
		commit = p.Target.Commit.Hash
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Build"), IDStyle.Render(fmt.Sprintf("#%d", p.BuildNumber))))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("UUID"), p.UUID))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("State"), state))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Branch"), BranchStyle.Render(p.Target.RefName)))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Commit"), commit))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Created"), DimStyle.Render(p.CreatedOn)))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Completed"), DimStyle.Render(p.CompletedOn)))
	return sb.String()
}

// PipelineDetail prints the formatted pipeline detail to stdout.
func PipelineDetail(p bitbucket.Pipeline) {
	fmt.Print(PipelineDetailString(p))
}

// PipelineStepsString returns formatted text for a list of pipeline steps.
func PipelineStepsString(steps []bitbucket.PipelineStep) string {
	if len(steps) == 0 {
		return "No steps found.\n"
	}

	header := fmt.Sprintf("  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-38s", "UUID")),
		LabelStyle.Render(fmt.Sprintf("%-20s", "NAME")),
		LabelStyle.Render("STATE"))
	divider := fmt.Sprintf("  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 38)),
		SepStyle.Render(strings.Repeat("─", 20)),
		SepStyle.Render(strings.Repeat("─", 20)))

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)

	for _, s := range steps {
		state := s.State.Name
		if s.State.Result != nil {
			state += "/" + s.State.Result.Name
		}
		sb.WriteString(fmt.Sprintf("  %-38s  %-20s  %s\n",
			s.UUID, truncate(s.Name, 20), StateBadge(state)))
	}
	return sb.String()
}

// PipelineSteps prints the formatted pipeline steps to stdout.
func PipelineSteps(steps []bitbucket.PipelineStep) {
	fmt.Print(PipelineStepsString(steps))
}

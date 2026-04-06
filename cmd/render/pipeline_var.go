package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// PipelineVariableListString returns the formatted text for a list of pipeline variables.
func PipelineVariableListString(vars []bitbucket.PipelineVariable) string {
	if len(vars) == 0 {
		return "No pipeline variables found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-30s", "KEY")),
		LabelStyle.Render(fmt.Sprintf("%-36s", "UUID")),
		LabelStyle.Render("SECURED"))
	divider := fmt.Sprintf("  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 30)),
		SepStyle.Render(strings.Repeat("─", 36)),
		SepStyle.Render(strings.Repeat("─", 7)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, v := range vars {
		secured := "no"
		if v.Secured {
			secured = "yes"
		}
		sb.WriteString(fmt.Sprintf("  %-30s  %-36s  %s\n",
			IDStyle.Render(truncate(v.Key, 30)),
			DimStyle.Render(v.UUID),
			secured))
	}
	return sb.String()
}

// PipelineVariableList prints the formatted pipeline variable list to stdout.
func PipelineVariableList(vars []bitbucket.PipelineVariable) { fmt.Print(PipelineVariableListString(vars)) }

// PipelineVariableDetailString returns the full detail string for a single pipeline variable.
func PipelineVariableDetailString(v bitbucket.PipelineVariable) string {
	secured := "no"
	if v.Secured {
		secured = "yes"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Key:    "), IDStyle.Render(v.Key)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("UUID:   "), DimStyle.Render(v.UUID)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Secured:"), secured))
	if !v.Secured && v.Value != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Value:  "), v.Value))
	}
	return sb.String()
}

// PipelineVariableDetail prints the formatted pipeline variable detail to stdout.
func PipelineVariableDetail(v bitbucket.PipelineVariable) { fmt.Print(PipelineVariableDetailString(v)) }

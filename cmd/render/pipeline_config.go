package render

import (
	"fmt"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// PipelineConfigDetailString returns the formatted detail string for a
// repository's pipelines configuration.
func PipelineConfigDetailString(c bitbucket.PipelineConfig) string {
	status := "disabled"
	if c.Enabled {
		status = "enabled"
	}
	return fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Pipelines:"), status)
}

// PipelineConfigDetail prints the formatted pipelines configuration to stdout.
func PipelineConfigDetail(c bitbucket.PipelineConfig) { fmt.Print(PipelineConfigDetailString(c)) }

package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// WorkspaceListString returns the formatted text for a list of workspaces.
func WorkspaceListString(workspaces []bitbucket.Workspace) string {
	if len(workspaces) == 0 {
		return "No workspaces found.\n"
	}
	header := fmt.Sprintf("  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-30s", "SLUG")),
		LabelStyle.Render("NAME"))
	divider := fmt.Sprintf("  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 30)),
		SepStyle.Render(strings.Repeat("─", 40)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, w := range workspaces {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-30s", truncate(w.Slug, 30))),
			BranchStyle.Render(truncate(w.Name, 40))))
	}
	return sb.String()
}

// WorkspaceList prints the formatted workspace list to stdout.
func WorkspaceList(workspaces []bitbucket.Workspace) { fmt.Print(WorkspaceListString(workspaces)) }

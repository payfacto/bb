package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// ProjectListString returns the formatted text for a list of projects.
func ProjectListString(projects []bitbucket.Project) string {
	if len(projects) == 0 {
		return "No projects found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-10s", "KEY")),
		LabelStyle.Render(fmt.Sprintf("%-40s", "NAME")),
		LabelStyle.Render("VISIBILITY"))
	divider := fmt.Sprintf("  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 10)),
		SepStyle.Render(strings.Repeat("─", 40)),
		SepStyle.Render(strings.Repeat("─", 10)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, p := range projects {
		visibility := "private"
		if !p.IsPrivate {
			visibility = "public"
		}
		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-10s", truncate(p.Key, 10))),
			BranchStyle.Render(fmt.Sprintf("%-40s", truncate(p.Name, 40))),
			DimStyle.Render(visibility)))
	}
	return sb.String()
}

// ProjectList prints the formatted project list to stdout.
func ProjectList(projects []bitbucket.Project) { fmt.Print(ProjectListString(projects)) }

// ProjectDetailString returns the full detail string for a single project.
func ProjectDetailString(p bitbucket.Project) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Key:        "), IDStyle.Render(p.Key)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Name:       "), BranchStyle.Render(p.Name)))
	if p.Description != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Description:"), p.Description))
	}
	visibility := "private"
	if !p.IsPrivate {
		visibility = "public"
	}
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Visibility: "), DimStyle.Render(visibility)))
	if p.HasPubliclyVisibleRepos {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Public repos:"), "yes"))
	}
	if p.UUID != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("UUID:       "), DimStyle.Render(p.UUID)))
	}
	if p.CreatedOn != "" {
		created := p.CreatedOn
		if len(created) > 10 {
			created = created[:10]
		}
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Created:    "), created))
	}
	if p.UpdatedOn != "" {
		updated := p.UpdatedOn
		if len(updated) > 10 {
			updated = updated[:10]
		}
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Updated:    "), updated))
	}
	return sb.String()
}

// ProjectDetail prints the formatted project detail to stdout.
func ProjectDetail(p bitbucket.Project) { fmt.Print(ProjectDetailString(p)) }

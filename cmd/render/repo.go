package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// BranchListString returns the formatted text for a list of branches.
func BranchListString(branches []bitbucket.Branch) string {
	if len(branches) == 0 {
		return "No branches found.\n"
	}
	header := fmt.Sprintf("  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-40s", "NAME")),
		LabelStyle.Render("HASH"))
	divider := fmt.Sprintf("  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 40)),
		SepStyle.Render(strings.Repeat("─", 8)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, b := range branches {
		hash := b.Target.Hash
		if len(hash) >= shortHashLen {
			hash = hash[:shortHashLen]
		}
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			BranchStyle.Render(fmt.Sprintf("%-40s", truncate(b.Name, 40))),
			IDStyle.Render(hash)))
	}
	return sb.String()
}

// BranchDetailString returns the full detail string for a single branch (no truncation).
func BranchDetailString(b bitbucket.Branch) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Name:"), BranchStyle.Render(b.Name)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Hash:"), IDStyle.Render(b.Target.Hash)))
	return sb.String()
}

// BranchList prints the formatted branch list to stdout.
func BranchList(branches []bitbucket.Branch) { fmt.Print(BranchListString(branches)) }

// TagListString returns the formatted text for a list of tags.
func TagListString(tags []bitbucket.Tag) string {
	if len(tags) == 0 {
		return "No tags found.\n"
	}
	header := fmt.Sprintf("  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-40s", "NAME")),
		LabelStyle.Render("HASH"))
	divider := fmt.Sprintf("  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 40)),
		SepStyle.Render(strings.Repeat("─", 8)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, t := range tags {
		hash := t.Target.Hash
		if len(hash) >= shortHashLen {
			hash = hash[:shortHashLen]
		}
		sb.WriteString(fmt.Sprintf("  %-40s  %s\n",
			BranchStyle.Render(truncate(t.Name, 40)),
			IDStyle.Render(hash)))
	}
	return sb.String()
}

// TagList prints the formatted tag list to stdout.
func TagList(tags []bitbucket.Tag) { fmt.Print(TagListString(tags)) }

// RepoListString returns the formatted text for a list of repositories.
func RepoListString(repos []bitbucket.Repo) string {
	if len(repos) == 0 {
		return "No repositories found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-30s", "SLUG")),
		LabelStyle.Render(fmt.Sprintf("%-40s", "NAME")),
		LabelStyle.Render("ACCESS"))
	divider := fmt.Sprintf("  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 30)),
		SepStyle.Render(strings.Repeat("─", 40)),
		SepStyle.Render(strings.Repeat("─", 8)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, r := range repos {
		privacy := "public"
		if r.IsPrivate {
			privacy = "private"
		}
		sb.WriteString(fmt.Sprintf("  %-30s  %-40s  %s\n",
			IDStyle.Render(r.Slug), truncate(r.Name, 40), StateBadge(privacy)))
	}
	return sb.String()
}

// RepoList prints the formatted repository list to stdout.
func RepoList(repos []bitbucket.Repo) { fmt.Print(RepoListString(repos)) }

// RepoDetailString returns the full detail string for a single repository.
func RepoDetailString(r bitbucket.Repo) string {
	privacy := "public"
	if r.IsPrivate {
		privacy = "private"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Slug:       "), IDStyle.Render(r.Slug)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Name:       "), BranchStyle.Render(r.Name)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Full name:  "), r.FullName))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Access:     "), StateBadge(privacy)))
	if r.Description != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Description:"), r.Description))
	}
	if r.Project != nil {
		sb.WriteString(fmt.Sprintf("  %s  %s (%s)\n", LabelStyle.Render("Project:    "), r.Project.Name, r.Project.Key))
	}
	if r.Links.HTML.Href != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("URL:        "), DimStyle.Render(r.Links.HTML.Href)))
	}
	return sb.String()
}

// RepoDetail prints the formatted repository detail to stdout.
func RepoDetail(r bitbucket.Repo) { fmt.Print(RepoDetailString(r)) }

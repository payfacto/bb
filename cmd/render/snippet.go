package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// SnippetListString returns the formatted text for a list of snippets.
func SnippetListString(snippets []bitbucket.Snippet) string {
	if len(snippets) == 0 {
		return "No snippets found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-10s", "ID")),
		LabelStyle.Render(fmt.Sprintf("%-40s", "TITLE")),
		LabelStyle.Render(fmt.Sprintf("%-12s", "OWNER")),
		LabelStyle.Render("VISIBILITY"))
	divider := fmt.Sprintf("  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 10)),
		SepStyle.Render(strings.Repeat("─", 40)),
		SepStyle.Render(strings.Repeat("─", 12)),
		SepStyle.Render(strings.Repeat("─", 10)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, s := range snippets {
		visibility := "public"
		if s.IsPrivate {
			visibility = "private"
		}
		sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-10s", truncate(s.ID, 10))),
			BranchStyle.Render(fmt.Sprintf("%-40s", truncate(s.Title, 40))),
			DimStyle.Render(fmt.Sprintf("%-12s", truncate(s.Owner.DisplayName, 12))),
			DimStyle.Render(visibility)))
	}
	return sb.String()
}

// SnippetList prints the formatted snippet list to stdout.
func SnippetList(snippets []bitbucket.Snippet) { fmt.Print(SnippetListString(snippets)) }

// SnippetDetailString returns the full detail string for a single snippet.
func SnippetDetailString(s bitbucket.Snippet) string {
	visibility := "public"
	if s.IsPrivate {
		visibility = "private"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("ID:         "), IDStyle.Render(s.ID)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Title:      "), BranchStyle.Render(s.Title)))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Owner:      "), s.Owner.DisplayName))
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Visibility: "), DimStyle.Render(visibility)))
	if s.CreatedOn != "" {
		created := s.CreatedOn
		if len(created) > 10 {
			created = created[:10]
		}
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Created:    "), created))
	}
	if s.UpdatedOn != "" {
		updated := s.UpdatedOn
		if len(updated) > 10 {
			updated = updated[:10]
		}
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Updated:    "), updated))
	}
	if len(s.Files) > 0 {
		sb.WriteString(fmt.Sprintf("  %s\n", LabelStyle.Render("Files:")))
		for name := range s.Files {
			sb.WriteString(fmt.Sprintf("    %s\n", DimStyle.Render(name)))
		}
	}
	if s.Links.HTML.Href != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("URL:        "), DimStyle.Render(s.Links.HTML.Href)))
	}
	return sb.String()
}

// SnippetDetail prints the formatted snippet detail to stdout.
func SnippetDetail(s bitbucket.Snippet) { fmt.Print(SnippetDetailString(s)) }

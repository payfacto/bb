package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// IssueListString returns the formatted text for a list of issues.
func IssueListString(issues []bitbucket.Issue) string {
	if len(issues) == 0 {
		return "No issues found.\n"
	}

	header := fmt.Sprintf("  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-6s", "ID")),
		LabelStyle.Render(fmt.Sprintf("%-10s", "STATE")),
		LabelStyle.Render(fmt.Sprintf("%-12s", "KIND")),
		LabelStyle.Render("TITLE"))
	divider := fmt.Sprintf("  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 6)),
		SepStyle.Render(strings.Repeat("─", 10)),
		SepStyle.Render(strings.Repeat("─", 12)),
		SepStyle.Render(strings.Repeat("─", 40)))

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, i := range issues {
		sb.WriteString(fmt.Sprintf("  %s  %-10s  %-12s  %s\n",
			IDStyle.Render(fmt.Sprintf("#%-4d", i.ID)),
			StateBadge(i.State), i.Kind, truncate(i.Title, 50)))
	}
	return sb.String()
}

// IssueList prints the formatted issue list to stdout.
func IssueList(issues []bitbucket.Issue) {
	fmt.Print(IssueListString(issues))
}

// IssueDetailString returns the formatted text for a single issue.
func IssueDetailString(issue bitbucket.Issue) string {
	const labelW = 9

	label := func(s string) string {
		return LabelStyle.Render(fmt.Sprintf("  %-*s", labelW, s))
	}
	sep := SepStyle.Render(strings.Repeat("─", 56))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("ID"), IDStyle.Render(fmt.Sprintf("#%d", issue.ID))))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Title"), issue.Title))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("State"), StateBadge(issue.State)))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Kind"), issue.Kind))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Priority"), issue.Priority))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Reporter"), issue.Reporter.DisplayName))
	if issue.Content.Raw != "" {
		sb.WriteString(sep + "\n")
		sb.WriteString(RenderMarkdown(issue.Content.Raw))
	}
	return sb.String()
}

// IssueDetail prints the formatted issue detail to stdout, paging if needed.
func IssueDetail(issue bitbucket.Issue) {
	MaybePage(IssueDetailString(issue))
}

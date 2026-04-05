package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	titleMaxLen  = 45
	authorMaxLen = 12
)

// truncate shortens s to max runes, appending … if truncated.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// PRListString returns the formatted text for a list of PRs.
func PRListString(prs []bitbucket.PR) string {
	if len(prs) == 0 {
		return "No pull requests found.\n"
	}

	sep := SepStyle.Render(strings.Repeat("─", 4))
	titleSep := SepStyle.Render(strings.Repeat("─", titleMaxLen))
	stateSep := SepStyle.Render(strings.Repeat("─", 10))
	authorSep := SepStyle.Render(strings.Repeat("─", authorMaxLen))
	branchSep := SepStyle.Render(strings.Repeat("─", 20))

	header := fmt.Sprintf("  %s  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-4s", "ID")),
		LabelStyle.Render(fmt.Sprintf("%-*s", titleMaxLen, "TITLE")),
		LabelStyle.Render(fmt.Sprintf("%-10s", "STATE")),
		LabelStyle.Render(fmt.Sprintf("%-*s", authorMaxLen, "AUTHOR")),
		LabelStyle.Render("BRANCH"),
	)
	divider := fmt.Sprintf("  %s  %s  %s  %s  %s\n",
		sep, titleSep, stateSep, authorSep, branchSep,
	)

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)

	for _, pr := range prs {
		id := IDStyle.Render(fmt.Sprintf("#%-3d", pr.ID))
		title := truncate(pr.Title, titleMaxLen)
		state := StateBadge(pr.State)
		author := truncate(pr.Author.DisplayName, authorMaxLen)
		branch := fmt.Sprintf("%s → %s",
			BranchStyle.Render(pr.Source.Branch.Name),
			BranchStyle.Render(pr.Destination.Branch.Name),
		)
		sb.WriteString(fmt.Sprintf("  %s  %-*s  %-10s  %-*s  %s\n",
			id, titleMaxLen, title, state, authorMaxLen, author, branch,
		))
	}
	return sb.String()
}

// PRList prints the formatted PR list to stdout.
func PRList(prs []bitbucket.PR) {
	fmt.Print(PRListString(prs))
}

// PRDetailString returns the formatted text for a single PR.
func PRDetailString(pr bitbucket.PR) string {
	const labelW = 7

	label := func(s string) string {
		return LabelStyle.Render(fmt.Sprintf("  %-*s", labelW, s))
	}
	sep := SepStyle.Render(strings.Repeat("─", 56))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("ID"), IDStyle.Render(fmt.Sprintf("#%d", pr.ID))))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Title"), pr.Title))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("State"), StateBadge(pr.State)))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Author"), pr.Author.DisplayName))
	sb.WriteString(fmt.Sprintf("%s  %s → %s\n", label("Branch"),
		BranchStyle.Render(pr.Source.Branch.Name),
		BranchStyle.Render(pr.Destination.Branch.Name),
	))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("URL"), pr.Links.HTML.Href))

	if pr.Description != "" {
		sb.WriteString(sep + "\n")
		sb.WriteString(RenderMarkdown(pr.Description))
	}

	return sb.String()
}

// PRDetail prints the formatted PR detail to stdout, paging if the output
// is taller than the terminal.
func PRDetail(pr bitbucket.PR) {
	MaybePage(PRDetailString(pr))
}

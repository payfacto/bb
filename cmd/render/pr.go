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

const datePrefixLen = 10 // characters in "2026-04-01" portion of ISO-8601
const shortHashLen = 8  // characters to show for a commit hash abbreviation

// PRActivityString returns formatted text for a PR activity timeline.
func PRActivityString(activities []bitbucket.Activity) string {
	if len(activities) == 0 {
		return "No activity found.\n"
	}

	var sb strings.Builder
	for _, a := range activities {
		switch {
		case a.Approval != nil:
			date := a.Approval.Date
			if len(date) >= datePrefixLen {
				date = date[:datePrefixLen]
			}
			sb.WriteString(fmt.Sprintf("  %s  %s approved  %s\n",
				LabelStyle.Render("[approval]"),
				a.Approval.User.DisplayName,
				DimStyle.Render("("+date+")"),
			))
		case a.Comment != nil:
			sb.WriteString(fmt.Sprintf("  %s  %s: %s\n",
				LabelStyle.Render("[comment] "),
				a.Comment.User.DisplayName,
				truncate(a.Comment.Content.Raw, 80),
			))
		case a.Update != nil:
			date := a.Update.Date
			if len(date) >= datePrefixLen {
				date = date[:datePrefixLen]
			}
			sb.WriteString(fmt.Sprintf("  %s  %s → %s  %s\n",
				LabelStyle.Render("[update]  "),
				a.Update.Author.DisplayName,
				StateBadge(a.Update.State),
				DimStyle.Render("("+date+")"),
			))
		}
	}
	return sb.String()
}

// PRActivity prints the formatted PR activity to stdout.
func PRActivity(activities []bitbucket.Activity) {
	fmt.Print(PRActivityString(activities))
}

// PRStatusesString returns formatted text for PR build statuses.
func PRStatusesString(statuses []bitbucket.PRStatus) string {
	if len(statuses) == 0 {
		return "No statuses found.\n"
	}

	sep := SepStyle.Render(strings.Repeat("─", 24))
	stateSep := SepStyle.Render(strings.Repeat("─", 10))
	urlSep := SepStyle.Render(strings.Repeat("─", 30))

	header := fmt.Sprintf("  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-24s", "NAME")),
		LabelStyle.Render(fmt.Sprintf("%-10s", "STATE")),
		LabelStyle.Render("URL"),
	)
	divider := fmt.Sprintf("  %s  %s  %s\n", sep, stateSep, urlSep)

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, s := range statuses {
		sb.WriteString(fmt.Sprintf("  %-24s  %-10s  %s\n",
			truncate(s.Name, 24),
			StateBadge(s.State),
			s.URL,
		))
	}
	return sb.String()
}

// PRStatuses prints the formatted PR build statuses to stdout.
func PRStatuses(statuses []bitbucket.PRStatus) {
	fmt.Print(PRStatusesString(statuses))
}

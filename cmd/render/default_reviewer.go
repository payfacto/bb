package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	reviewerNameWidth      = 30
	reviewerAccountIDWidth = 40
	// repositoryReviewerType labels reviewers from the repository-only list,
	// whose endpoint does not return a reviewer_type.
	repositoryReviewerType = "repository"
)

// DefaultReviewerListString returns formatted text for a list of default
// reviewers, including where each reviewer is configured (repository or
// project).
func DefaultReviewerListString(reviewers []bitbucket.DefaultReviewer) string {
	if len(reviewers) == 0 {
		return "No default reviewers found.\n"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-*s", reviewerNameWidth, "NAME")),
		LabelStyle.Render(fmt.Sprintf("%-*s", reviewerAccountIDWidth, "ACCOUNT ID")),
		LabelStyle.Render("SOURCE"))
	fmt.Fprintf(&sb, "  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", reviewerNameWidth)),
		SepStyle.Render(strings.Repeat("─", reviewerAccountIDWidth)),
		SepStyle.Render(strings.Repeat("─", len(repositoryReviewerType))))
	for _, r := range reviewers {
		source := r.ReviewerType
		if source == "" {
			source = repositoryReviewerType
		}
		fmt.Fprintf(&sb, "  %s  %s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-*s", reviewerNameWidth, truncate(r.User.DisplayName, reviewerNameWidth))),
			DimStyle.Render(fmt.Sprintf("%-*s", reviewerAccountIDWidth, r.User.AccountID)),
			source)
	}
	return sb.String()
}

// DefaultReviewerList prints the formatted default reviewer list to stdout.
func DefaultReviewerList(reviewers []bitbucket.DefaultReviewer) {
	fmt.Print(DefaultReviewerListString(reviewers))
}

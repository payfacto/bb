package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var defaultReviewerCmd = &cobra.Command{
	Use:   "default-reviewer",
	Short: "Inspect repository default reviewers",
}

var defaultReviewerListEffective bool

var defaultReviewerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List default reviewers (--effective includes project-inherited reviewers)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		res := client.DefaultReviewers(ws, repo)
		var reviewers []bitbucket.DefaultReviewer
		if defaultReviewerListEffective {
			reviewers, err = res.Effective(context.Background())
		} else {
			reviewers, err = res.List(context.Background())
		}
		if err != nil {
			return err
		}
		return printOutput(reviewers, func() { render.DefaultReviewerList(reviewers) })
	},
}

func init() {
	defaultReviewerListCmd.Flags().BoolVar(&defaultReviewerListEffective, "effective", false,
		"list the reviewers actually applied to new PRs, including those inherited from the project")

	defaultReviewerCmd.AddCommand(defaultReviewerListCmd)
	rootCmd.AddCommand(defaultReviewerCmd)
}

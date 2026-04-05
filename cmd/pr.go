package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
}

var prListState string

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		prs, err := client.PRs(ws, r).List(context.Background(), prListState)
		if err != nil {
			return err
		}
		return printOutput(prs, func() { render.PRList(prs) })
	},
}

var prGetID int

var prGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get pull request details",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		pr, err := client.PRs(ws, r).Get(context.Background(), prGetID)
		if err != nil {
			return err
		}
		return printOutput(pr, func() { render.PRDetail(pr) })
	},
}

var (
	prCreateTitle       string
	prCreateFromBranch  string
	prCreateToBranch    string
	prCreateDescription string
	prCreateCloseSource bool
)

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		input := bitbucket.CreatePRInput{
			Title:             prCreateTitle,
			Description:       prCreateDescription,
			Source:            bitbucket.NewEndpoint(prCreateFromBranch),
			Destination:       bitbucket.NewEndpoint(prCreateToBranch),
			CloseSourceBranch: prCreateCloseSource,
		}
		pr, err := client.PRs(ws, r).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(pr, func() {
			fmt.Printf("PR created: #%d — %s\n", pr.ID, pr.Links.HTML.Href)
		})
	},
}

var prDiffID int

var prDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Get pull request diff (raw patch)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		diff, err := client.PRs(ws, r).Diff(context.Background(), prDiffID)
		if err != nil {
			return err
		}
		fmt.Print(diff) // always plain text regardless of --format
		return nil
	},
}

var prApproveID int

var prApproveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.PRs(ws, r).Approve(context.Background(), prApproveID); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "approved"}, func() {
			fmt.Println("Approved.")
		})
	},
}

var (
	prMergeID       int
	prMergeStrategy string
)

var prMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.PRs(ws, r).Merge(context.Background(), prMergeID, prMergeStrategy); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "merged"}, func() {
			fmt.Println("Merged.")
		})
	},
}

var prDeclineID int

var prDeclineCmd = &cobra.Command{
	Use:   "decline",
	Short: "Decline a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.PRs(ws, r).Decline(context.Background(), prDeclineID); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "declined"}, func() {
			fmt.Println("Declined.")
		})
	},
}

var prActivityID int

var prActivityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Show the activity timeline for a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		activities, err := client.PRs(ws, r).Activity(context.Background(), prActivityID)
		if err != nil {
			return err
		}
		return printOutput(activities, func() { render.PRActivity(activities) })
	},
}

var prStatusesID int

var prStatusesCmd = &cobra.Command{
	Use:   "statuses",
	Short: "Show build statuses for a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		statuses, err := client.PRs(ws, r).Statuses(context.Background(), prStatusesID)
		if err != nil {
			return err
		}
		return printOutput(statuses, func() { render.PRStatuses(statuses) })
	},
}

func init() {
	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prListCmd, prGetCmd, prCreateCmd, prDiffCmd, prApproveCmd, prMergeCmd, prDeclineCmd, prActivityCmd, prStatusesCmd)

	prListCmd.Flags().StringVar(&prListState, "state", "OPEN",
		"filter by state: OPEN, MERGED, DECLINED, SUPERSEDED")

	prGetCmd.Flags().IntVar(&prGetID, "pr-id", 0, "pull request ID")
	prGetCmd.MarkFlagRequired("pr-id")

	prCreateCmd.Flags().StringVar(&prCreateTitle, "title", "", "PR title")
	prCreateCmd.Flags().StringVar(&prCreateFromBranch, "from-branch", "", "source branch")
	prCreateCmd.Flags().StringVar(&prCreateToBranch, "to-branch", "", "destination branch")
	prCreateCmd.Flags().StringVar(&prCreateDescription, "description", "", "PR description")
	prCreateCmd.Flags().BoolVar(&prCreateCloseSource, "close-source-branch", false,
		"close source branch after merge")
	prCreateCmd.MarkFlagRequired("title")
	prCreateCmd.MarkFlagRequired("from-branch")
	prCreateCmd.MarkFlagRequired("to-branch")

	prDiffCmd.Flags().IntVar(&prDiffID, "pr-id", 0, "pull request ID")
	prDiffCmd.MarkFlagRequired("pr-id")

	prApproveCmd.Flags().IntVar(&prApproveID, "pr-id", 0, "pull request ID")
	prApproveCmd.MarkFlagRequired("pr-id")

	prMergeCmd.Flags().IntVar(&prMergeID, "pr-id", 0, "pull request ID")
	prMergeCmd.Flags().StringVar(&prMergeStrategy, "strategy", "merge_commit",
		"merge strategy: merge_commit, squash, fast_forward")
	prMergeCmd.MarkFlagRequired("pr-id")

	prDeclineCmd.Flags().IntVar(&prDeclineID, "pr-id", 0, "pull request ID")
	prDeclineCmd.MarkFlagRequired("pr-id")

	prActivityCmd.Flags().IntVar(&prActivityID, "pr-id", 0, "pull request ID")
	prActivityCmd.MarkFlagRequired("pr-id")

	prStatusesCmd.Flags().IntVar(&prStatusesID, "pr-id", 0, "pull request ID")
	prStatusesCmd.MarkFlagRequired("pr-id")
}

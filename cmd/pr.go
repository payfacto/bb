package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
}

// --- bb pr list ---

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
		return printOutput(prs, func() {
			if len(prs) == 0 {
				fmt.Println("No pull requests found.")
				return
			}
			for _, pr := range prs {
				fmt.Printf("  PR #%d  [%s]  %s\n", pr.ID, pr.State, pr.Title)
				fmt.Printf("         %s → %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)
				fmt.Printf("         %s\n\n", pr.Links.HTML.Href)
			}
		})
	},
}

// --- bb pr get ---

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
		return printOutput(pr, func() {
			fmt.Printf("PR #%d: %s [%s]\n", pr.ID, pr.Title, pr.State)
			fmt.Printf("  %s → %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)
			fmt.Printf("  Author: %s\n", pr.Author.DisplayName)
			if pr.Description != "" {
				fmt.Printf("  Description: %s\n", pr.Description)
			}
			fmt.Printf("  URL: %s\n", pr.Links.HTML.Href)
		})
	},
}

// --- bb pr create ---

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

// --- bb pr diff ---

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

// --- bb pr approve ---

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

// --- bb pr merge ---

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

// --- bb pr decline ---

var prDeclineID int

var prActivityID int
var prStatusesID int

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

// --- bb pr activity ---

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
		return printOutput(activities, func() {
			if len(activities) == 0 {
				fmt.Println("No activity found.")
				return
			}
			for _, a := range activities {
				switch {
				case a.Approval != nil:
					date := a.Approval.Date
					if len(date) >= 10 {
						date = date[:10]
					}
					fmt.Printf("[approval]  %s approved  (%s)\n",
						a.Approval.User.DisplayName, date)
				case a.Comment != nil:
					fmt.Printf("[comment]   %s: %s\n",
						a.Comment.User.DisplayName, truncate(a.Comment.Content.Raw, 80))
				case a.Update != nil:
					date := a.Update.Date
					if len(date) >= 10 {
						date = date[:10]
					}
					fmt.Printf("[update]    %s → %s  (%s)\n",
						a.Update.Author.DisplayName, a.Update.State, date)
				}
			}
		})
	},
}

// --- bb pr statuses ---

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
		return printOutput(statuses, func() {
			if len(statuses) == 0 {
				fmt.Println("No statuses found.")
				return
			}
			for _, s := range statuses {
				fmt.Printf("%-12s  %-30s  %s\n",
					s.State, truncate(s.Name, 30), s.Description)
			}
		})
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

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/internal/git"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
}

var (
	prListState        string
	prListSourceBranch string
	prListSort         string
	prListSince        string
	prListUntil        string
)

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		prs, err := client.PRs(ws, r).List(context.Background(), bitbucket.PRListOptions{
			State:        prListState,
			SourceBranch: prListSourceBranch,
			Sort:         prListSort,
			Since:        prListSince,
			Until:        prListUntil,
		})
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
	prCreateTitle           string
	prCreateFromBranch      string
	prCreateToBranch        string
	prCreateDescription     string
	prCreateDescriptionFile string
	prCreateCloseSource     bool
	prCreateDraft           bool
)

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Auto-detect workspace/repo from the git origin remote when unset.
		// Config/flags always win; notes go to stderr so stdout stays clean.
		// Notes are held until after workspaceAndRepo() validates, so a user
		// never sees "inferred --workspace=X" for a value that was rejected.
		newWs, newRepo, notes := inferWorkspaceRepo(cfg.Workspace, cfg.Repo, git.OriginURL)
		cfg.Workspace, cfg.Repo = newWs, newRepo
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		for _, n := range notes {
			fmt.Fprintln(os.Stderr, n)
		}
		description, err := resolveTextBody(prCreateDescription, prCreateDescriptionFile, "description", "description-file")
		if err != nil {
			return err
		}
		var input bitbucket.CreatePRInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreatePRInput {
			return bitbucket.CreatePRInput{
				Title:             prCreateTitle,
				Description:       description,
				Source:            bitbucket.NewEndpoint(prCreateFromBranch),
				Destination:       bitbucket.NewEndpoint(prCreateToBranch),
				CloseSourceBranch: prCreateCloseSource,
				Draft:             prCreateDraft,
			}
		})
		if err != nil {
			return err
		}
		if !consumed {
			// Auto-detect the source branch from the current git branch when
			// omitted, then patch the already-built input.
			if b, note := inferFromBranch(prCreateFromBranch, git.CurrentBranch); note != "" {
				prCreateFromBranch = b
				input.Source = bitbucket.NewEndpoint(b)
				fmt.Fprintln(os.Stderr, note)
			}
			if err := requireFlag("title", prCreateTitle); err != nil {
				return err
			}
			if err := requireFlag("from-branch", prCreateFromBranch); err != nil {
				return err
			}
			if err := requireFlag("to-branch", prCreateToBranch); err != nil {
				return err
			}
		}
		pr, err := client.PRs(ws, r).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(pr, func() {
			fmt.Printf("PR created: #%d - %s\n", pr.ID, pr.Links.HTML.Href)
		})
	},
}

var (
	prUpdateID              int
	prUpdateTitle           string
	prUpdateDescription     string
	prUpdateDescriptionFile string
)

var prUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a pull request's title and/or description",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		description, err := resolveTextBody(prUpdateDescription, prUpdateDescriptionFile, "description", "description-file")
		if err != nil {
			return err
		}
		// Only set a pointer for a field the caller actually provided. A nil
		// pointer means "leave unchanged"; a non-nil pointer sets the value
		// (an empty description clears it). On the stdin path, JSON unmarshals
		// straight into the pointer struct: an absent field stays nil, "" clears.
		var input bitbucket.UpdatePRInput
		if _, err := stdinInputOr(&input, func() bitbucket.UpdatePRInput {
			var flagInput bitbucket.UpdatePRInput
			if cmd.Flags().Changed("title") {
				flagInput.Title = &prUpdateTitle
			}
			if cmd.Flags().Changed("description") || cmd.Flags().Changed("description-file") {
				flagInput.Description = &description
			}
			return flagInput
		}); err != nil {
			return err
		}
		// Title cannot be blank: trim a provided title and reject if empty.
		if input.Title != nil {
			trimmed := strings.TrimSpace(*input.Title)
			if trimmed == "" {
				return newCLIError(ErrCodeValidationFailed, "title cannot be empty", nil)
			}
			input.Title = &trimmed
		}
		if input.Title == nil && input.Description == nil {
			return newCLIError(ErrCodeValidationFailed,
				"nothing to update: provide --title and/or --description (or pipe JSON on stdin)", nil)
		}
		pr, err := client.PRs(ws, r).Update(context.Background(), prUpdateID, input)
		if err != nil {
			return err
		}
		return printOutput(pr, func() {
			fmt.Printf("PR #%d updated: %s\n", pr.ID, pr.Links.HTML.Href)
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

var prOpenID int

var prOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open a pull request in your web browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		pr, err := client.PRs(ws, repo).Get(context.Background(), prOpenID)
		if err != nil {
			return err
		}
		url := pr.Links.HTML.Href
		if url == "" {
			return newCLIError(ErrCodeNotFound, fmt.Sprintf("pull request %d has no web link", prOpenID), nil)
		}
		if err := browser.OpenURL(url); err != nil {
			fmt.Fprintf(os.Stderr, "could not open browser: %v\n", err)
		}
		return printOutput(map[string]any{"id": prOpenID, "url": url}, func() {
			fmt.Println(url)
		})
	},
}

var (
	prAddReviewerID        int
	prAddReviewerAccountID string
)

var prAddReviewerCmd = &cobra.Command{
	Use:   "add-reviewer",
	Short: "Add a reviewer to a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.PRs(ws, r).AddReviewer(context.Background(), prAddReviewerID, prAddReviewerAccountID); err != nil {
			return err
		}
		return printOutput(map[string]any{"added": true, "pr_id": prAddReviewerID, "account_id": prAddReviewerAccountID}, func() {
			fmt.Printf("Reviewer %s added to PR #%d\n", prAddReviewerAccountID, prAddReviewerID)
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
	prCmd.AddCommand(prListCmd, prGetCmd, prCreateCmd, prUpdateCmd, prDiffCmd, prApproveCmd, prMergeCmd, prDeclineCmd, prOpenCmd, prActivityCmd, prStatusesCmd, prAddReviewerCmd)

	prListCmd.Flags().StringVarP(&prListState, "state", "s", "OPEN",
		"filter by state: OPEN, MERGED, DECLINED, SUPERSEDED")
	prListCmd.Flags().StringVar(&prListSourceBranch, "source-branch", "",
		"filter to PRs whose source branch matches this name exactly")
	prListCmd.Flags().StringVar(&prListSort, "sort", "",
		"sort by Bitbucket field, prefix with - for descending (e.g. -updated_on); empty preserves API default")
	prListCmd.Flags().StringVar(&prListSince, "since", "",
		"only PRs created on or after this ISO-8601 timestamp (e.g. 2023-01-01T00:00:00+00:00)")
	prListCmd.Flags().StringVar(&prListUntil, "until", "",
		"only PRs created on or before this ISO-8601 timestamp")

	prGetCmd.Flags().IntVarP(&prGetID, "pr-id", "p", 0, "pull request ID")
	prGetCmd.MarkFlagRequired("pr-id")

	prCreateCmd.Flags().StringVarP(&prCreateTitle, "title", "T", "", "PR title")
	prCreateCmd.Flags().StringVar(&prCreateFromBranch, "from-branch", "", "source branch")
	prCreateCmd.Flags().StringVar(&prCreateToBranch, "to-branch", "", "destination branch")
	prCreateCmd.Flags().StringVarP(&prCreateDescription, "description", "d", "", "PR description")
	prCreateCmd.Flags().StringVar(&prCreateDescriptionFile, "description-file", "", "path to a file containing the PR description (mutually exclusive with --description)")
	prCreateCmd.Flags().BoolVar(&prCreateCloseSource, "close-source-branch", false,
		"close source branch after merge")
	prCreateCmd.Flags().BoolVar(&prCreateDraft, "draft", false,
		"create as a draft PR (no reviewer notifications)")
	// no MarkFlagRequired - pr create accepts JSON on stdin as an alternative
	// to flags. RunE validates required fields when stdin is not consumed.

	prUpdateCmd.Flags().IntVarP(&prUpdateID, "pr-id", "p", 0, "pull request ID")
	prUpdateCmd.MarkFlagRequired("pr-id")
	prUpdateCmd.Flags().StringVarP(&prUpdateTitle, "title", "T", "", "new PR title")
	prUpdateCmd.Flags().StringVarP(&prUpdateDescription, "description", "d", "", "new PR description")
	prUpdateCmd.Flags().StringVar(&prUpdateDescriptionFile, "description-file", "",
		"path to a file containing the new PR description (mutually exclusive with --description)")
	// pr-id addresses the PR in the URL (not the body), so it stays required
	// even though title/description may arrive via stdin JSON.

	prDiffCmd.Flags().IntVarP(&prDiffID, "pr-id", "p", 0, "pull request ID")
	prDiffCmd.MarkFlagRequired("pr-id")

	prApproveCmd.Flags().IntVarP(&prApproveID, "pr-id", "p", 0, "pull request ID")
	prApproveCmd.MarkFlagRequired("pr-id")

	prMergeCmd.Flags().IntVarP(&prMergeID, "pr-id", "p", 0, "pull request ID")
	prMergeCmd.Flags().StringVar(&prMergeStrategy, "strategy", "merge_commit",
		"merge strategy: merge_commit, squash, fast_forward")
	prMergeCmd.MarkFlagRequired("pr-id")

	prDeclineCmd.Flags().IntVarP(&prDeclineID, "pr-id", "p", 0, "pull request ID")
	prDeclineCmd.MarkFlagRequired("pr-id")

	prOpenCmd.Flags().IntVarP(&prOpenID, "pr-id", "p", 0, "pull request ID")
	prOpenCmd.MarkFlagRequired("pr-id")

	prActivityCmd.Flags().IntVarP(&prActivityID, "pr-id", "p", 0, "pull request ID")
	prActivityCmd.MarkFlagRequired("pr-id")

	prStatusesCmd.Flags().IntVarP(&prStatusesID, "pr-id", "p", 0, "pull request ID")
	prStatusesCmd.MarkFlagRequired("pr-id")

	prAddReviewerCmd.Flags().IntVarP(&prAddReviewerID, "id", "i", 0, "pull request ID (required)")
	prAddReviewerCmd.Flags().StringVar(&prAddReviewerAccountID, "account-id", "", "reviewer account ID (required)")
	prAddReviewerCmd.MarkFlagRequired("id")
	prAddReviewerCmd.MarkFlagRequired("account-id")
}

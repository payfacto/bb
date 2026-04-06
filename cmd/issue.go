package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage repository issues",
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		issues, err := client.Issues(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(issues, func() { render.IssueList(issues) })
	},
}

var issueGetID int

var issueGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a single issue by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		issue, err := client.Issues(ws, repo).Get(context.Background(), issueGetID)
		if err != nil {
			return err
		}
		return printOutput(issue, func() { render.IssueDetail(issue) })
	},
}

var (
	issueCreateTitle       string
	issueCreateDescription string
	issueCreateKind        string
	issueCreatePriority    string
)

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		input := bitbucket.CreateIssueInput{
			Title:    issueCreateTitle,
			Kind:     issueCreateKind,
			Priority: issueCreatePriority,
		}
		if issueCreateDescription != "" {
			c := bitbucket.Content{Raw: issueCreateDescription}
			input.Content = &c
		}
		issue, err := client.Issues(ws, repo).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(issue, func() {
			fmt.Printf("Issue #%d created: %s\n", issue.ID, issue.Title)
		})
	},
}

var issueCloseID int

var issueCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Close an issue (set status to resolved)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		issue, err := client.Issues(ws, repo).Update(context.Background(), issueCloseID, bitbucket.UpdateIssueInput{Status: "resolved"})
		if err != nil {
			return err
		}
		return printOutput(issue, func() {
			fmt.Printf("Issue #%d closed\n", issue.ID)
		})
	},
}

var issueReopenID int

var issueReopenCmd = &cobra.Command{
	Use:   "reopen",
	Short: "Reopen an issue (set status to open)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		issue, err := client.Issues(ws, repo).Update(context.Background(), issueReopenID, bitbucket.UpdateIssueInput{Status: "open"})
		if err != nil {
			return err
		}
		return printOutput(issue, func() {
			fmt.Printf("Issue #%d reopened\n", issue.ID)
		})
	},
}

func init() {
	issueGetCmd.Flags().IntVarP(&issueGetID, "id", "i", 0, "issue ID (required)")
	issueGetCmd.MarkFlagRequired("id")

	issueCreateCmd.Flags().StringVarP(&issueCreateTitle, "title", "T", "", "issue title (required)")
	issueCreateCmd.Flags().StringVarP(&issueCreateDescription, "description", "d", "", "issue description")
	issueCreateCmd.Flags().StringVarP(&issueCreateKind, "kind", "k", "", "issue kind: bug, enhancement, proposal, task")
	issueCreateCmd.Flags().StringVar(&issueCreatePriority, "priority", "", "issue priority: trivial, minor, major, critical, blocker")
	issueCreateCmd.MarkFlagRequired("title")

	issueCloseCmd.Flags().IntVarP(&issueCloseID, "id", "i", 0, "issue ID (required)")
	issueCloseCmd.MarkFlagRequired("id")

	issueReopenCmd.Flags().IntVarP(&issueReopenID, "id", "i", 0, "issue ID (required)")
	issueReopenCmd.MarkFlagRequired("id")

	issueCmd.AddCommand(issueListCmd, issueGetCmd, issueCreateCmd, issueCloseCmd, issueReopenCmd)
	rootCmd.AddCommand(issueCmd)
}

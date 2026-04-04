package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

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
		return printOutput(issues, func() {
			if len(issues) == 0 {
				fmt.Println("No issues found.")
				return
			}
			for _, i := range issues {
				fmt.Printf("#%-5d  %-10s  %-12s  %s\n",
					i.ID, i.State, i.Kind, truncate(i.Title, 50))
			}
		})
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
		return printOutput(issue, func() {
			fmt.Printf("#%d  %s\n", issue.ID, issue.Title)
			fmt.Printf("State:    %-10s  Kind: %-12s  Priority: %s\n",
				issue.State, issue.Kind, issue.Priority)
			fmt.Printf("Reporter: %s\n", issue.Reporter.DisplayName)
			if issue.Content.Raw != "" {
				fmt.Printf("\n%s\n", issue.Content.Raw)
			}
		})
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

func init() {
	issueGetCmd.Flags().IntVar(&issueGetID, "id", 0, "issue ID (required)")
	issueGetCmd.MarkFlagRequired("id")

	issueCreateCmd.Flags().StringVar(&issueCreateTitle, "title", "", "issue title (required)")
	issueCreateCmd.Flags().StringVar(&issueCreateDescription, "description", "", "issue description")
	issueCreateCmd.Flags().StringVar(&issueCreateKind, "kind", "", "issue kind: bug, enhancement, proposal, task")
	issueCreateCmd.Flags().StringVar(&issueCreatePriority, "priority", "", "issue priority: trivial, minor, major, critical, blocker")
	issueCreateCmd.MarkFlagRequired("title")

	issueCmd.AddCommand(issueListCmd, issueGetCmd, issueCreateCmd)
	rootCmd.AddCommand(issueCmd)
}

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Browse commit history",
}

var commitListBranch string

var commitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List commits on a branch, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		commits, err := client.Commits(ws, repo).List(context.Background(), commitListBranch)
		if err != nil {
			return err
		}
		return printOutput(commits, func() {
			if len(commits) == 0 {
				fmt.Println("No commits found.")
				return
			}
			for _, c := range commits {
				date := c.Date
				if len(date) >= datePrefixLen {
					date = date[:datePrefixLen]
				}
				msg := c.Message
				if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
					msg = msg[:idx]
				}
				fmt.Printf("%s  %s  %-30s  %s\n",
					c.Hash[:shortHashLen], date, truncate(c.Author.Raw, 30), truncate(msg, 72))
			}
		})
	},
}

var commitGetHash string

var commitGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a single commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		c, err := client.Commits(ws, repo).Get(context.Background(), commitGetHash)
		if err != nil {
			return err
		}
		return printOutput(c, func() {
			parents := make([]string, len(c.Parents))
			for i, p := range c.Parents {
				parents[i] = truncate(p.Hash, 8)
			}
			fmt.Printf("Hash:    %s\nDate:    %s\nAuthor:  %s\nMessage: %s\nParents: %s\n",
				c.Hash, c.Date, c.Author.Raw, c.Message, strings.Join(parents, ", "))
		})
	},
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Read file contents from the repository",
}

var (
	fileGetRef  string
	fileGetPath string
)

var fileGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get raw file contents at a ref (branch, tag, or commit hash)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		content, err := client.Commits(ws, repo).File(context.Background(), fileGetRef, fileGetPath)
		if err != nil {
			return err
		}
		fmt.Print(content)
		return nil
	},
}

func init() {
	commitListCmd.Flags().StringVar(&commitListBranch, "branch", "", "branch name (required)")
	commitListCmd.MarkFlagRequired("branch")

	commitGetCmd.Flags().StringVar(&commitGetHash, "hash", "", "commit hash (required)")
	commitGetCmd.MarkFlagRequired("hash")

	fileGetCmd.Flags().StringVar(&fileGetRef, "ref", "", "branch name, tag, or commit hash (required)")
	fileGetCmd.Flags().StringVar(&fileGetPath, "path", "", "file path within the repository (required)")
	fileGetCmd.MarkFlagRequired("ref")
	fileGetCmd.MarkFlagRequired("path")

	commitCmd.AddCommand(commitListCmd, commitGetCmd)
	fileCmd.AddCommand(fileGetCmd)
	rootCmd.AddCommand(commitCmd, fileCmd)
}

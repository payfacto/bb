package cmd

import (
	"context"
	"fmt"

	"github.com/payfacto/bb/cmd/render"
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
		return printOutput(commits, func() { render.CommitList(commits) })
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
		return printOutput(c, func() { render.CommitDetail(c) })
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

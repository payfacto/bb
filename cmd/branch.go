package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Manage repository branches",
}

var branchListSort string

var branchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List branches in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		branches, err := client.Branches(ws, repo).List(context.Background(), branchListSort)
		if err != nil {
			return err
		}
		return printOutput(branches, func() { render.BranchList(branches) })
	},
}

var (
	branchCreateName string
	branchCreateFrom string
)

var branchCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new branch from a branch name or commit hash",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		var input bitbucket.CreateBranchInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreateBranchInput {
			return bitbucket.CreateBranchInput{
				Name:   branchCreateName,
				Target: bitbucket.BranchTarget{Hash: branchCreateFrom},
			}
		})
		if err != nil {
			return err
		}
		if !consumed {
			if err := requireFlag("name", branchCreateName); err != nil {
				return err
			}
			if err := requireFlag("from", branchCreateFrom); err != nil {
				return err
			}
		}
		branch, err := client.Branches(ws, repo).Create(context.Background(), input.Name, input.Target.Hash)
		if err != nil {
			return err
		}
		return printOutput(branch, func() {
			fmt.Printf("Branch '%s' created at %s\n", branch.Name, branch.Target.Hash)
		})
	},
}

var branchDeleteName string

var branchDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Branches(ws, repo).Delete(context.Background(), branchDeleteName); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "deleted", "name": branchDeleteName}, func() {
			fmt.Printf("Branch '%s' deleted.\n", branchDeleteName)
		})
	},
}

func init() {
	branchCreateCmd.Flags().StringVarP(&branchCreateName, "name", "n", "", "name for the new branch (required)")
	branchCreateCmd.Flags().StringVar(&branchCreateFrom, "from", "", "branch name or commit hash to branch from (required)")
	// no MarkFlagRequired — branch create accepts JSON on stdin.

	branchListCmd.Flags().StringVar(&branchListSort, "sort", "",
		"sort by Bitbucket field, prefix with - for descending (e.g. -target.date); empty preserves API default")

	branchDeleteCmd.Flags().StringVarP(&branchDeleteName, "name", "n", "", "branch name to delete (required)")
	branchDeleteCmd.MarkFlagRequired("name")

	branchCmd.AddCommand(branchListCmd, branchCreateCmd, branchDeleteCmd)
	rootCmd.AddCommand(branchCmd)
}

package cmd

import (
	"context"
	"fmt"

	"github.com/payfacto/bb/cmd/render"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "List and browse repositories",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories in the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		// repo list only needs workspace, not repo slug
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		repos, err := client.Repos(ws).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(repos, func() { render.RepoList(repos) })
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)
	rootCmd.AddCommand(repoCmd)
}

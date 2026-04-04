package cmd

import (
	"context"
	"fmt"

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
		return printOutput(repos, func() {
			if len(repos) == 0 {
				fmt.Println("No repositories found.")
				return
			}
			for _, r := range repos {
				privacy := "public"
				if r.IsPrivate {
					privacy = "private"
				}
				fmt.Printf("%-30s  %-40s  (%s)\n", r.Slug, truncate(r.Name, 40), privacy)
			}
		})
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)
	rootCmd.AddCommand(repoCmd)
}

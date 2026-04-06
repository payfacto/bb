package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage deployment environments",
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployment environments in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		envs, err := client.Environments(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(envs, func() { render.EnvList(envs) })
	},
}

func init() {
	envCmd.AddCommand(envListCmd)
	rootCmd.AddCommand(envCmd)
}

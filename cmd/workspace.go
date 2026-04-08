package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage Bitbucket workspaces",
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces accessible to the authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaces, err := client.Workspaces().List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(workspaces, func() { render.WorkspaceList(workspaces) })
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceListCmd)
	rootCmd.AddCommand(workspaceCmd)
}

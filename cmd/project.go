package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage workspace projects",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects in the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		projects, err := client.Projects(ws).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(projects, func() { render.ProjectList(projects) })
	},
}

var projectGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a project by its key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		project, err := client.Projects(ws).Get(context.Background(), args[0])
		if err != nil {
			return err
		}
		return printOutput(project, func() { render.ProjectDetail(project) })
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectGetCmd)
	rootCmd.AddCommand(projectCmd)
}

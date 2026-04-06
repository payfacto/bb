package cmd

import (
	"context"
	"fmt"

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
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
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
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
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

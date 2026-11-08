package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var runnerCmd = &cobra.Command{
	Use:   "runner",
	Short: "Inspect and remove self-hosted Pipelines runners",
}

// runnerWorkspaceLevel is shared by every runner subcommand (persistent flag).
var runnerWorkspaceLevel bool

// runnerResource returns the workspace runner resource when --workspace-level
// is set, otherwise the repository runner resource.
func runnerResource() (*bitbucket.RunnerResource, error) {
	if runnerWorkspaceLevel {
		ws, err := workspaceOnly()
		if err != nil {
			return nil, err
		}
		return client.WorkspaceRunners(ws), nil
	}
	ws, repo, err := workspaceAndRepo()
	if err != nil {
		return nil, err
	}
	return client.RepoRunners(ws, repo), nil
}

var runnerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List runners for the repository (or workspace with --workspace-level)",
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := runnerResource()
		if err != nil {
			return err
		}
		runners, err := res.List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(runners, func() { render.RunnerList(runners) })
	},
}

var runnerGetUUID string

var runnerGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a runner by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := runnerResource()
		if err != nil {
			return err
		}
		r, err := res.Get(context.Background(), runnerGetUUID)
		if err != nil {
			return err
		}
		return printOutput(r, func() { render.RunnerDetail(r) })
	},
}

var runnerDeleteUUID string

var runnerDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a runner by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := runnerResource()
		if err != nil {
			return err
		}
		if err := res.Delete(context.Background(), runnerDeleteUUID); err != nil {
			return err
		}
		return printOutput(map[string]any{"deleted": true, "uuid": runnerDeleteUUID}, func() {
			fmt.Printf("Runner %s deleted\n", runnerDeleteUUID)
		})
	},
}

func init() {
	runnerCmd.PersistentFlags().BoolVar(&runnerWorkspaceLevel, "workspace-level", false,
		"operate on workspace runners instead of repository runners")

	runnerGetCmd.Flags().StringVar(&runnerGetUUID, "uuid", "", "runner UUID (required)")
	runnerGetCmd.MarkFlagRequired("uuid")

	runnerDeleteCmd.Flags().StringVar(&runnerDeleteUUID, "uuid", "", "runner UUID (required)")
	runnerDeleteCmd.MarkFlagRequired("uuid")

	runnerCmd.AddCommand(runnerListCmd, runnerGetCmd, runnerDeleteCmd)
	rootCmd.AddCommand(runnerCmd)
}

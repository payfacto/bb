package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var pipelineConfigCmd = &cobra.Command{
	Use:   "pipeline-config",
	Short: "Manage the repository's Pipelines enabled/disabled setting",
}

var pipelineConfigGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show whether Pipelines is enabled for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		c, err := client.PipelineConfig(ws, repo).Get(context.Background())
		if err != nil {
			return err
		}
		return printOutput(c, func() { render.PipelineConfigDetail(c) })
	},
}

var pipelineConfigEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Pipelines for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		c, err := client.PipelineConfig(ws, repo).Enable(context.Background())
		if err != nil {
			return err
		}
		return printOutput(c, func() { render.PipelineConfigDetail(c) })
	},
}

var pipelineConfigDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Pipelines for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		c, err := client.PipelineConfig(ws, repo).Disable(context.Background())
		if err != nil {
			return err
		}
		return printOutput(c, func() { render.PipelineConfigDetail(c) })
	},
}

func init() {
	pipelineConfigCmd.AddCommand(pipelineConfigGetCmd, pipelineConfigEnableCmd, pipelineConfigDisableCmd)
	rootCmd.AddCommand(pipelineConfigCmd)
}

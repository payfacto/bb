package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var envVarCmd = &cobra.Command{
	Use:   "env-var",
	Short: "Manage deployment environment variables",
}

var envVarListEnvUUID string

var envVarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List variables for a deployment environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := requireFlag("env-uuid", envVarListEnvUUID); err != nil {
			return err
		}
		vars, err := client.EnvironmentVariables(ws, repo, envVarListEnvUUID).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(vars, func() { render.PipelineVariableList(vars) })
	},
}

func init() {
	envVarListCmd.Flags().StringVar(&envVarListEnvUUID, "env-uuid", "", "environment UUID (required)")
	envVarCmd.AddCommand(envVarListCmd)
	rootCmd.AddCommand(envVarCmd)
}

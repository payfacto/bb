package cmd

import (
	"context"
	"fmt"

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

var envGetUUID string

var envGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a deployment environment by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		env, err := client.Environments(ws, repo).Get(context.Background(), envGetUUID)
		if err != nil {
			return err
		}
		return printOutput(env, func() { render.EnvDetail(env) })
	},
}

var envDeleteUUID string

var envDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a deployment environment by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Environments(ws, repo).Delete(context.Background(), envDeleteUUID); err != nil {
			return err
		}
		result := map[string]any{"deleted": true, "uuid": envDeleteUUID}
		return printOutput(result, func() { fmt.Printf("Environment %s deleted\n", envDeleteUUID) })
	},
}

func init() {
	envGetCmd.Flags().StringVar(&envGetUUID, "uuid", "", "Environment UUID (including braces)")
	_ = envGetCmd.MarkFlagRequired("uuid")

	envDeleteCmd.Flags().StringVar(&envDeleteUUID, "uuid", "", "Environment UUID (including braces)")
	_ = envDeleteCmd.MarkFlagRequired("uuid")

	envCmd.AddCommand(envListCmd, envGetCmd, envDeleteCmd)
	rootCmd.AddCommand(envCmd)
}

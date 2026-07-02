package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
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

var (
	envCreateName string
	envCreateType string
)

var envCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a deployment environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		var input bitbucket.CreateEnvironmentInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreateEnvironmentInput {
			return bitbucket.CreateEnvironmentInput{
				Name:            envCreateName,
				EnvironmentType: bitbucket.EnvironmentType{Name: envCreateType},
			}
		})
		if err != nil {
			return err
		}
		if !consumed {
			if err := requireFlag("name", envCreateName); err != nil {
				return err
			}
			if err := requireFlag("type", envCreateType); err != nil {
				return err
			}
		}
		env, err := client.Environments(ws, repo).Create(context.Background(), input)
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

	envCreateCmd.Flags().StringVar(&envCreateName, "name", "", "environment name (required)")
	envCreateCmd.Flags().StringVar(&envCreateType, "type", "", "environment type: Test, Staging, or Production (required)")
	// no MarkFlagRequired -- name/type validated in RunE so stdin JSON works.

	envDeleteCmd.Flags().StringVar(&envDeleteUUID, "uuid", "", "Environment UUID (including braces)")
	_ = envDeleteCmd.MarkFlagRequired("uuid")

	envCmd.AddCommand(envListCmd, envGetCmd, envCreateCmd, envDeleteCmd)
	rootCmd.AddCommand(envCmd)
}

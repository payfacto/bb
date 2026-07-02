package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
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

var (
	envVarCreateEnvUUID string
	envVarCreateKey     string
	envVarCreateValue   string
	envVarCreateSecured bool
)

var envVarCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a variable for a deployment environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := requireFlag("env-uuid", envVarCreateEnvUUID); err != nil {
			return err
		}
		var input bitbucket.CreatePipelineVariableInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreatePipelineVariableInput {
			return bitbucket.CreatePipelineVariableInput{
				Key:     envVarCreateKey,
				Value:   envVarCreateValue,
				Secured: envVarCreateSecured,
			}
		})
		if err != nil {
			return err
		}
		if !consumed {
			if err := requireFlag("key", envVarCreateKey); err != nil {
				return err
			}
		}
		v, err := client.EnvironmentVariables(ws, repo, envVarCreateEnvUUID).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}

var (
	envVarUpdateEnvUUID string
	envVarUpdateUUID    string
	envVarUpdateKey     string
	envVarUpdateValue   string
	envVarUpdateSecured bool
)

var envVarUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a deployment environment variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := requireFlag("env-uuid", envVarUpdateEnvUUID); err != nil {
			return err
		}
		if err := requireFlag("uuid", envVarUpdateUUID); err != nil {
			return err
		}
		res := client.EnvironmentVariables(ws, repo, envVarUpdateEnvUUID)

		var input bitbucket.CreatePipelineVariableInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreatePipelineVariableInput {
			return bitbucket.CreatePipelineVariableInput{
				Key:     envVarUpdateKey,
				Value:   envVarUpdateValue,
				Secured: envVarUpdateSecured,
			}
		})
		if err != nil {
			return err
		}

		if !consumed {
			// Flag path: --value is required; default key/secured from the
			// current variable (found via List, since there is no env-var get)
			// unless overridden. A secured value is unreadable, so it is never reused.
			if err := requireFlag("value", envVarUpdateValue); err != nil {
				return err
			}
			if !cmd.Flags().Changed("key") || !cmd.Flags().Changed("secured") {
				vars, err := res.List(context.Background())
				if err != nil {
					return err
				}
				var current *bitbucket.PipelineVariable
				for i := range vars {
					if vars[i].UUID == envVarUpdateUUID {
						current = &vars[i]
						break
					}
				}
				if current == nil {
					return fmt.Errorf("variable %s not found in environment %s", envVarUpdateUUID, envVarUpdateEnvUUID)
				}
				if !cmd.Flags().Changed("key") {
					input.Key = current.Key
				}
				if !cmd.Flags().Changed("secured") {
					input.Secured = current.Secured
				}
			}
		}

		v, err := res.Update(context.Background(), envVarUpdateUUID, input)
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}

var (
	envVarDeleteEnvUUID string
	envVarDeleteUUID    string
)

var envVarDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a variable from a deployment environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.EnvironmentVariables(ws, repo, envVarDeleteEnvUUID).Delete(context.Background(), envVarDeleteUUID); err != nil {
			return err
		}
		return printOutput(map[string]any{"deleted": true, "uuid": envVarDeleteUUID}, func() {
			fmt.Printf("Environment variable %s deleted\n", envVarDeleteUUID)
		})
	},
}

func init() {
	envVarListCmd.Flags().StringVar(&envVarListEnvUUID, "env-uuid", "", "environment UUID (required)")

	envVarCreateCmd.Flags().StringVar(&envVarCreateEnvUUID, "env-uuid", "", "environment UUID (required)")
	envVarCreateCmd.Flags().StringVarP(&envVarCreateKey, "key", "k", "", "variable key (required)")
	envVarCreateCmd.Flags().StringVarP(&envVarCreateValue, "value", "v", "", "variable value")
	envVarCreateCmd.Flags().BoolVar(&envVarCreateSecured, "secured", false, "mark variable as secured")
	// env-uuid and key validated in RunE so stdin JSON works.

	envVarUpdateCmd.Flags().StringVar(&envVarUpdateEnvUUID, "env-uuid", "", "environment UUID (required)")
	envVarUpdateCmd.Flags().StringVar(&envVarUpdateUUID, "uuid", "", "variable UUID (required)")
	envVarUpdateCmd.Flags().StringVarP(&envVarUpdateKey, "key", "k", "", "new key (defaults to current)")
	envVarUpdateCmd.Flags().StringVarP(&envVarUpdateValue, "value", "v", "", "new value (required unless piping JSON)")
	envVarUpdateCmd.Flags().BoolVar(&envVarUpdateSecured, "secured", false, "mark variable as secured (defaults to current)")

	envVarDeleteCmd.Flags().StringVar(&envVarDeleteEnvUUID, "env-uuid", "", "environment UUID (required)")
	envVarDeleteCmd.Flags().StringVar(&envVarDeleteUUID, "uuid", "", "variable UUID (required)")
	envVarDeleteCmd.MarkFlagRequired("env-uuid")
	envVarDeleteCmd.MarkFlagRequired("uuid")

	envVarCmd.AddCommand(envVarListCmd, envVarCreateCmd, envVarUpdateCmd, envVarDeleteCmd)
	rootCmd.AddCommand(envVarCmd)
}

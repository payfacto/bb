package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var pipelineVarCmd = &cobra.Command{
	Use:   "pipeline-var",
	Short: "Manage repository pipeline variables",
}

var pipelineVarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pipeline variables for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		vars, err := client.PipelineVariables(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(vars, func() { render.PipelineVariableList(vars) })
	},
}

var (
	pipelineVarCreateKey     string
	pipelineVarCreateValue   string
	pipelineVarCreateSecured bool
)

var pipelineVarCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pipeline variable",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		var input bitbucket.CreatePipelineVariableInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreatePipelineVariableInput {
			return bitbucket.CreatePipelineVariableInput{
				Key:     pipelineVarCreateKey,
				Value:   pipelineVarCreateValue,
				Secured: pipelineVarCreateSecured,
			}
		})
		if err != nil {
			return err
		}
		if !consumed {
			if err := requireFlag("key", pipelineVarCreateKey); err != nil {
				return err
			}
		}
		v, err := client.PipelineVariables(ws, repo).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}

var pipelineVarGetUUID string

var pipelineVarGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a pipeline variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		v, err := client.PipelineVariables(ws, repo).Get(context.Background(), pipelineVarGetUUID)
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}

var (
	pipelineVarUpdateUUID    string
	pipelineVarUpdateKey     string
	pipelineVarUpdateValue   string
	pipelineVarUpdateSecured bool
)

var pipelineVarUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a pipeline variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := requireFlag("uuid", pipelineVarUpdateUUID); err != nil {
			return err
		}
		res := client.PipelineVariables(ws, repo)

		var input bitbucket.CreatePipelineVariableInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreatePipelineVariableInput {
			return bitbucket.CreatePipelineVariableInput{
				Key:     pipelineVarUpdateKey,
				Value:   pipelineVarUpdateValue,
				Secured: pipelineVarUpdateSecured,
			}
		})
		if err != nil {
			return err
		}

		if !consumed {
			// Flag path: stdinInputOr already populated input from flag vars; requireFlag validates the --value flag is present.
			// Default key/secured from the current variable unless overridden. (A secured variable's value
			// is not readable, so we never reuse the fetched value.)
			if err := requireFlag("value", pipelineVarUpdateValue); err != nil {
				return err
			}
			current, err := res.Get(context.Background(), pipelineVarUpdateUUID)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("key") {
				input.Key = current.Key
			}
			if !cmd.Flags().Changed("secured") {
				input.Secured = current.Secured
			}
		}

		v, err := res.Update(context.Background(), pipelineVarUpdateUUID, input)
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}

var pipelineVarDeleteUUID string

var pipelineVarDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a pipeline variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.PipelineVariables(ws, repo).Delete(context.Background(), pipelineVarDeleteUUID); err != nil {
			return err
		}
		return printOutput(map[string]any{"deleted": true, "uuid": pipelineVarDeleteUUID}, func() {
			fmt.Printf("Pipeline variable %s deleted\n", pipelineVarDeleteUUID)
		})
	},
}

func init() {
	pipelineVarCreateCmd.Flags().StringVarP(&pipelineVarCreateKey, "key", "k", "", "variable key (required)")
	pipelineVarCreateCmd.Flags().StringVarP(&pipelineVarCreateValue, "value", "v", "", "variable value")
	pipelineVarCreateCmd.Flags().BoolVar(&pipelineVarCreateSecured, "secured", false, "mark variable as secured (value hidden in UI)")
	// no MarkFlagRequired on "key" -- pipeline-var create accepts JSON on stdin.

	pipelineVarGetCmd.Flags().StringVar(&pipelineVarGetUUID, "uuid", "", "variable UUID (required)")
	pipelineVarGetCmd.MarkFlagRequired("uuid")

	pipelineVarUpdateCmd.Flags().StringVar(&pipelineVarUpdateUUID, "uuid", "", "variable UUID (required)")
	pipelineVarUpdateCmd.Flags().StringVarP(&pipelineVarUpdateKey, "key", "k", "", "new key (defaults to current)")
	pipelineVarUpdateCmd.Flags().StringVarP(&pipelineVarUpdateValue, "value", "v", "", "new value (required unless piping JSON)")
	pipelineVarUpdateCmd.Flags().BoolVar(&pipelineVarUpdateSecured, "secured", false, "mark variable as secured (defaults to current)")
	// no MarkFlagRequired -- uuid/value are validated in RunE so stdin JSON works.

	pipelineVarDeleteCmd.Flags().StringVar(&pipelineVarDeleteUUID, "uuid", "", "variable UUID (required)")
	pipelineVarDeleteCmd.MarkFlagRequired("uuid")

	pipelineVarCmd.AddCommand(pipelineVarListCmd, pipelineVarGetCmd, pipelineVarCreateCmd, pipelineVarUpdateCmd, pipelineVarDeleteCmd)
	rootCmd.AddCommand(pipelineVarCmd)
}

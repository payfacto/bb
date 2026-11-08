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
		v, err := createVariable(client.PipelineVariables(ws, repo), bitbucket.CreatePipelineVariableInput{
			Key:     pipelineVarCreateKey,
			Value:   pipelineVarCreateValue,
			Secured: pipelineVarCreateSecured,
		})
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
		v, err := updateVariable(cmd, client.PipelineVariables(ws, repo), pipelineVarUpdateUUID, bitbucket.CreatePipelineVariableInput{
			Key:     pipelineVarUpdateKey,
			Value:   pipelineVarUpdateValue,
			Secured: pipelineVarUpdateSecured,
		})
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

// createVariable creates a pipeline variable (repository or workspace scope)
// from piped stdin JSON, falling back to the flag-built input. --key is
// required on the flag path.
func createVariable(res *bitbucket.PipelineVariableResource, flagInput bitbucket.CreatePipelineVariableInput) (bitbucket.PipelineVariable, error) {
	var input bitbucket.CreatePipelineVariableInput
	consumed, err := stdinInputOr(&input, func() bitbucket.CreatePipelineVariableInput { return flagInput })
	if err != nil {
		return bitbucket.PipelineVariable{}, err
	}
	if !consumed {
		if err := requireFlag("key", flagInput.Key); err != nil {
			return bitbucket.PipelineVariable{}, err
		}
	}
	return res.Create(context.Background(), input)
}

// updateVariable replaces a pipeline variable (repository or workspace scope)
// from piped stdin JSON, falling back to the flag-built input. On the flag
// path --value is required and key/secured default to the current variable
// unless their flags were set. The fetched value is never reused: a secured
// variable's value is not readable.
func updateVariable(cmd *cobra.Command, res *bitbucket.PipelineVariableResource, uuid string, flagInput bitbucket.CreatePipelineVariableInput) (bitbucket.PipelineVariable, error) {
	ctx := context.Background()
	if err := requireFlag("uuid", uuid); err != nil {
		return bitbucket.PipelineVariable{}, err
	}
	var input bitbucket.CreatePipelineVariableInput
	consumed, err := stdinInputOr(&input, func() bitbucket.CreatePipelineVariableInput { return flagInput })
	if err != nil {
		return bitbucket.PipelineVariable{}, err
	}
	if !consumed {
		if err := requireFlag("value", flagInput.Value); err != nil {
			return bitbucket.PipelineVariable{}, err
		}
		current, err := res.Get(ctx, uuid)
		if err != nil {
			return bitbucket.PipelineVariable{}, err
		}
		if !cmd.Flags().Changed("key") {
			input.Key = current.Key
		}
		if !cmd.Flags().Changed("secured") {
			input.Secured = current.Secured
		}
	}
	return res.Update(ctx, uuid, input)
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

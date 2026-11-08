package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var workspaceVarCmd = &cobra.Command{
	Use:   "workspace-var",
	Short: "Manage workspace-level pipeline variables",
}

var workspaceVarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pipeline variables for the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		vars, err := client.WorkspaceVariables(ws).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(vars, func() { render.PipelineVariableList(vars) })
	},
}

var workspaceVarGetUUID string

var workspaceVarGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a workspace pipeline variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		v, err := client.WorkspaceVariables(ws).Get(context.Background(), workspaceVarGetUUID)
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}

var (
	workspaceVarCreateKey     string
	workspaceVarCreateValue   string
	workspaceVarCreateSecured bool
)

var workspaceVarCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a workspace pipeline variable",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		v, err := createVariable(client.WorkspaceVariables(ws), bitbucket.CreatePipelineVariableInput{
			Key:     workspaceVarCreateKey,
			Value:   workspaceVarCreateValue,
			Secured: workspaceVarCreateSecured,
		})
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}

var (
	workspaceVarUpdateUUID    string
	workspaceVarUpdateKey     string
	workspaceVarUpdateValue   string
	workspaceVarUpdateSecured bool
)

var workspaceVarUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a workspace pipeline variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		v, err := updateVariable(cmd, client.WorkspaceVariables(ws), workspaceVarUpdateUUID, bitbucket.CreatePipelineVariableInput{
			Key:     workspaceVarUpdateKey,
			Value:   workspaceVarUpdateValue,
			Secured: workspaceVarUpdateSecured,
		})
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}

var workspaceVarDeleteUUID string

var workspaceVarDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a workspace pipeline variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		if err := client.WorkspaceVariables(ws).Delete(context.Background(), workspaceVarDeleteUUID); err != nil {
			return err
		}
		return printOutput(map[string]any{"deleted": true, "uuid": workspaceVarDeleteUUID}, func() {
			fmt.Printf("Workspace variable %s deleted\n", workspaceVarDeleteUUID)
		})
	},
}

func init() {
	workspaceVarGetCmd.Flags().StringVar(&workspaceVarGetUUID, "uuid", "", "variable UUID (required)")
	workspaceVarGetCmd.MarkFlagRequired("uuid")

	workspaceVarCreateCmd.Flags().StringVarP(&workspaceVarCreateKey, "key", "k", "", "variable key (required)")
	workspaceVarCreateCmd.Flags().StringVarP(&workspaceVarCreateValue, "value", "v", "", "variable value")
	workspaceVarCreateCmd.Flags().BoolVar(&workspaceVarCreateSecured, "secured", false, "mark variable as secured (value hidden in UI)")
	// no MarkFlagRequired on "key" -- workspace-var create accepts JSON on stdin.

	workspaceVarUpdateCmd.Flags().StringVar(&workspaceVarUpdateUUID, "uuid", "", "variable UUID (required)")
	workspaceVarUpdateCmd.Flags().StringVarP(&workspaceVarUpdateKey, "key", "k", "", "new key (defaults to current)")
	workspaceVarUpdateCmd.Flags().StringVarP(&workspaceVarUpdateValue, "value", "v", "", "new value (required unless piping JSON)")
	workspaceVarUpdateCmd.Flags().BoolVar(&workspaceVarUpdateSecured, "secured", false, "mark variable as secured (defaults to current)")
	// no MarkFlagRequired -- uuid/value are validated in RunE so stdin JSON works.

	workspaceVarDeleteCmd.Flags().StringVar(&workspaceVarDeleteUUID, "uuid", "", "variable UUID (required)")
	workspaceVarDeleteCmd.MarkFlagRequired("uuid")

	workspaceVarCmd.AddCommand(workspaceVarListCmd, workspaceVarGetCmd, workspaceVarCreateCmd, workspaceVarUpdateCmd, workspaceVarDeleteCmd)
	rootCmd.AddCommand(workspaceVarCmd)
}

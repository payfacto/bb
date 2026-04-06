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
		v, err := client.PipelineVariables(ws, repo).Create(context.Background(), bitbucket.CreatePipelineVariableInput{
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
	pipelineVarCreateCmd.MarkFlagRequired("key")

	pipelineVarDeleteCmd.Flags().StringVar(&pipelineVarDeleteUUID, "uuid", "", "variable UUID (required)")
	pipelineVarDeleteCmd.MarkFlagRequired("uuid")

	pipelineVarCmd.AddCommand(pipelineVarListCmd, pipelineVarCreateCmd, pipelineVarDeleteCmd)
	rootCmd.AddCommand(pipelineVarCmd)
}

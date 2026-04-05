package cmd

import (
	"context"

	"github.com/payfacto/bb/cmd/render"
	"github.com/spf13/cobra"
)

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "View repository deployments",
}

var deploymentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent deployments, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		deployments, err := client.Deployments(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(deployments, func() { render.DeploymentList(deployments) })
	},
}

func init() {
	deploymentCmd.AddCommand(deploymentListCmd)
	rootCmd.AddCommand(deploymentCmd)
}

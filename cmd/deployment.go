package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "View repository deployments",
}

var deploymentListEnvUUID string
var deploymentListSort string

var deploymentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent deployments, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		opts := bitbucket.DeploymentListOptions{
			EnvUUID: deploymentListEnvUUID,
			Sort:    deploymentListSort,
		}
		deployments, err := client.Deployments(ws, repo).List(context.Background(), opts)
		if err != nil {
			return err
		}
		return printOutput(deployments, func() { render.DeploymentList(deployments) })
	},
}

var deploymentGetUUID string

var deploymentGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a deployment by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		dep, err := client.Deployments(ws, repo).Get(context.Background(), deploymentGetUUID)
		if err != nil {
			return err
		}
		return printOutput(dep, func() { render.DeploymentDetail(dep) })
	},
}

func init() {
	deploymentListCmd.Flags().StringVar(&deploymentListEnvUUID, "env-uuid", "", "filter to a single environment UUID")
	deploymentListCmd.Flags().StringVar(&deploymentListSort, "sort", "", "sort field (e.g. -last_update_time)")
	deploymentGetCmd.Flags().StringVar(&deploymentGetUUID, "uuid", "", "deployment UUID (required)")
	deploymentGetCmd.MarkFlagRequired("uuid")
	deploymentCmd.AddCommand(deploymentListCmd, deploymentGetCmd)
	rootCmd.AddCommand(deploymentCmd)
}

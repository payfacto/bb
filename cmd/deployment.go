package cmd

import (
	"context"
	"fmt"

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
		return printOutput(deployments, func() {
			if len(deployments) == 0 {
				fmt.Println("No deployments found.")
				return
			}
			for _, d := range deployments {
				status := d.State.Name
				if d.State.Status != nil {
					status += "/" + d.State.Status.Name
				}
				commit := ""
				if d.Deployable.Commit != nil && len(d.Deployable.Commit.Hash) >= shortHashLen {
					commit = d.Deployable.Commit.Hash[:shortHashLen]
				}
				date := ""
				if len(d.LastUpdateTime) >= datePrefixLen {
					date = d.LastUpdateTime[:datePrefixLen]
				}
				envUUID := truncate(d.Environment.UUID, 38)
				fmt.Printf("%-38s  %-30s  %-8s  %s\n",
					envUUID, status, commit, date)
			}
		})
	},
}

func init() {
	deploymentCmd.AddCommand(deploymentListCmd)
	rootCmd.AddCommand(deploymentCmd)
}

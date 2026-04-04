package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var deployKeyCmd = &cobra.Command{
	Use:   "deploy-key",
	Short: "Manage repository deploy keys",
}

var deployKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deploy keys for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		keys, err := client.DeployKeys(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(keys, func() {
			if len(keys) == 0 {
				fmt.Println("No deploy keys found.")
				return
			}
			for _, k := range keys {
				fmt.Printf("%-6d  %-30s  %s\n", k.ID, truncate(k.Label, 30), truncate(k.Key, 50))
			}
		})
	},
}

var (
	deployKeyAddLabel string
	deployKeyAddKey   string
)

var deployKeyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a deploy key to the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		k, err := client.DeployKeys(ws, repo).Add(context.Background(), deployKeyAddLabel, deployKeyAddKey)
		if err != nil {
			return err
		}
		return printOutput(k, func() {
			fmt.Printf("Deploy key '%s' added (ID: %d).\n", k.Label, k.ID)
		})
	},
}

var deployKeyDeleteID int

var deployKeyDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a deploy key by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.DeployKeys(ws, repo).Delete(context.Background(), deployKeyDeleteID); err != nil {
			return err
		}
		return printOutput(map[string]any{"result": "deleted", "id": deployKeyDeleteID}, func() {
			fmt.Printf("Deploy key %d deleted.\n", deployKeyDeleteID)
		})
	},
}

func init() {
	deployKeyAddCmd.Flags().StringVar(&deployKeyAddLabel, "label", "", "label for the deploy key (required)")
	deployKeyAddCmd.Flags().StringVar(&deployKeyAddKey, "key", "", "SSH public key string (required)")
	deployKeyAddCmd.MarkFlagRequired("label")
	deployKeyAddCmd.MarkFlagRequired("key")

	deployKeyDeleteCmd.Flags().IntVar(&deployKeyDeleteID, "id", 0, "deploy key ID to delete (required)")
	deployKeyDeleteCmd.MarkFlagRequired("id")

	deployKeyCmd.AddCommand(deployKeyListCmd, deployKeyAddCmd, deployKeyDeleteCmd)
	rootCmd.AddCommand(deployKeyCmd)
}

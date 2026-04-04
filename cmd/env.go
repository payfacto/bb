package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
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
		return printOutput(envs, func() {
			if len(envs) == 0 {
				fmt.Println("No environments found.")
				return
			}
			for _, e := range envs {
				lock := ""
				if e.Lock.Name == "LOCKED" {
					lock = "  [LOCKED]"
				}
				fmt.Printf("%-40s  %-12s  %s%s\n",
					e.UUID, e.EnvironmentType.Name, e.Name, lock)
			}
		})
	},
}

func init() {
	envCmd.AddCommand(envListCmd)
	rootCmd.AddCommand(envCmd)
}

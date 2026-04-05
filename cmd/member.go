package cmd

import (
	"context"
	"fmt"

	"github.com/payfacto/bb/cmd/render"
	"github.com/spf13/cobra"
)

var memberCmd = &cobra.Command{
	Use:   "member",
	Short: "List workspace members",
}

var memberListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all members of the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		members, err := client.Members(ws).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(members, func() { render.MemberList(members) })
	},
}

func init() {
	memberCmd.AddCommand(memberListCmd)
	rootCmd.AddCommand(memberCmd)
}

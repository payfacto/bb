package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

var restrictionCmd = &cobra.Command{
	Use:   "restriction",
	Short: "Manage repository branch restrictions",
}

var restrictionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List branch restrictions for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		restrictions, err := client.Restrictions(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(restrictions, func() {
			if len(restrictions) == 0 {
				fmt.Println("No branch restrictions found.")
				return
			}
			for _, r := range restrictions {
				valueStr := ""
				if r.Value != nil {
					valueStr = fmt.Sprintf("%d", *r.Value)
				}
				fmt.Printf("%-6d  %-40s  %-6s  %s\n", r.ID, truncate(r.Kind, 40), valueStr, r.Pattern)
			}
		})
	},
}

var (
	restrictionCreateKind    string
	restrictionCreatePattern string
	restrictionCreateValue   int
)

var restrictionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a branch restriction",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		input := bitbucket.CreateBranchRestrictionInput{
			Kind:            restrictionCreateKind,
			BranchMatchKind: "glob",
			Pattern:         restrictionCreatePattern,
		}
		if restrictionCreateValue >= 0 {
			v := restrictionCreateValue
			input.Value = &v
		}
		r, err := client.Restrictions(ws, repo).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(r, func() {
			fmt.Printf("Branch restriction %d (%s on '%s') created.\n", r.ID, r.Kind, r.Pattern)
		})
	},
}

var restrictionDeleteID int

var restrictionDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a branch restriction by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Restrictions(ws, repo).Delete(context.Background(), restrictionDeleteID); err != nil {
			return err
		}
		return printOutput(map[string]any{"result": "deleted", "id": restrictionDeleteID}, func() {
			fmt.Printf("Branch restriction %d deleted.\n", restrictionDeleteID)
		})
	},
}

func init() {
	restrictionCreateCmd.Flags().StringVar(&restrictionCreateKind, "kind", "", "restriction kind, e.g. push, force, delete, require_approvals_to_merge (required)")
	restrictionCreateCmd.Flags().StringVar(&restrictionCreatePattern, "pattern", "", "branch glob pattern, e.g. main or feature/* (required)")
	restrictionCreateCmd.Flags().IntVar(&restrictionCreateValue, "value", -1, "integer value for restrictions that require one (e.g. number of approvals)")
	restrictionCreateCmd.MarkFlagRequired("kind")
	restrictionCreateCmd.MarkFlagRequired("pattern")

	restrictionDeleteCmd.Flags().IntVar(&restrictionDeleteID, "id", 0, "branch restriction ID to delete (required)")
	restrictionDeleteCmd.MarkFlagRequired("id")

	restrictionCmd.AddCommand(restrictionListCmd, restrictionCreateCmd, restrictionDeleteCmd)
	rootCmd.AddCommand(restrictionCmd)
}

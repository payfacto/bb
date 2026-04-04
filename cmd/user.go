package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Bitbucket user information",
}

var userMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the authenticated user's profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		u, err := client.User().Me(context.Background())
		if err != nil {
			return err
		}
		return printOutput(u, func() {
			fmt.Printf("%s (@%s)\nAccount ID: %s\n", u.DisplayName, u.Nickname, u.AccountID)
			if u.Links.HTML.Href != "" {
				fmt.Printf("Profile:    %s\n", u.Links.HTML.Href)
			}
		})
	},
}

func init() {
	userCmd.AddCommand(userMeCmd)
	rootCmd.AddCommand(userCmd)
}

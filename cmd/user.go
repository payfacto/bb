package cmd

import (
	"context"
	"fmt"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Bitbucket user information",
}

var userMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the authenticated user's profile",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return err
		}
		cfg.Apply(workspace, repo, username, token)
		if cfg.Username == "" || cfg.Token == "" {
			return fmt.Errorf("no credentials configured — run 'bb setup' to configure")
		}
		client = bitbucket.New(cfg)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		u, err := client.User().Me(context.Background())
		if err != nil {
			return err
		}
		return printOutput(u, func() { render.UserMe(u) })
	},
}

func init() {
	userCmd.AddCommand(userMeCmd)
	rootCmd.AddCommand(userCmd)
}

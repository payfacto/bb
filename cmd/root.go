package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var (
	cfgFile   string
	workspace string
	repo      string
	username  string
	token     string
	format    string

	cfg    *config.Config
	client *bitbucket.Client
)

var rootCmd = &cobra.Command{
	Use:   "bb",
	Short: "Bitbucket Cloud CLI",
	Long:  "A CLI for Bitbucket Cloud REST API 2.0. Run 'bb setup' to configure.",
	// PersistentPreRunE runs before every subcommand except those that override it.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return err
		}
		cfg.Apply(workspace, repo, username, token)
		if err := cfg.Validate(); err != nil {
			return err
		}

		// Credential resolution: keyring → env (already merged by Load) → CLI flag (merged by Apply).
		// CLI --token flag has final priority (already in cfg.Token via Apply).
		if cfg.Token == "" && cfg.Username != "" {
			tok, keyringErr := auth.GetToken(cfg.Username)
			if keyringErr == nil {
				cfg.Token = tok
			} else if !errors.Is(keyringErr, auth.ErrTokenNotFound) && !errors.Is(keyringErr, auth.ErrNoKeyring) {
				fmt.Fprintf(os.Stderr, "warning: keyring error (%v) — set BITBUCKET_TOKEN to authenticate\n", keyringErr)
			}
		}

		if err := cfg.ValidateCredentials(); err != nil {
			return err
		}

		client = bitbucket.New(cfg)
		if cfg.HasOAuth() {
			client.SetBearerToken(cfg.Token)
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.DefaultPath(), "config file path")
	rootCmd.PersistentFlags().StringVar(&workspace, "workspace", "", "Bitbucket workspace slug (overrides config)")
	rootCmd.PersistentFlags().StringVar(&repo, "repo", "", "repository slug (overrides config)")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Bitbucket username (overrides config/env)")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Bitbucket app password (overrides config/env)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "json", "output format: json or text")
}

// workspaceAndRepo returns the resolved workspace and repo, or an error if either is missing.
func workspaceAndRepo() (string, string, error) {
	if cfg.Workspace == "" {
		return "", "", fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
	}
	if cfg.Repo == "" {
		return "", "", fmt.Errorf("no repo configured — run 'bb setup' or pass --repo")
	}
	return cfg.Workspace, cfg.Repo, nil
}

// printOutput prints v as indented JSON (default) or calls textFn for --format text.
func printOutput(v any, textFn func()) error {
	if format == "text" {
		textFn()
		return nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

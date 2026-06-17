package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/tui"
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

// Version is the CLI version. Set at build time via:
//
//	go build -ldflags "-X 'github.com/payfacto/bb/cmd.Version=v1.2.3'" .
//
// Defaults to "dev" for local builds.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "bb",
	Short:   "Bitbucket Cloud CLI",
	Long:    "A CLI for Bitbucket Cloud REST API 2.0. Run 'bb setup' to configure.",
	Version: Version,
	// RunE is called when no subcommand is given — launches TUI.
	// Loads config but skips validation — TUI handles missing config with a setup wizard.
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = loadConfig(cmd)
		if err != nil {
			return err
		}
		cfg.Apply(workspace, repo, username, token)

		// Try to resolve credentials, but don't fail if missing — TUI will handle it.
		if cfg.Token == "" && cfg.Username != "" {
			tok, keyringErr := auth.GetToken(cfg.Username)
			if keyringErr == nil {
				cfg.Token = tok
			}
		}

		// Build client if we have credentials, otherwise pass nil — TUI detects this.
		if cfg.Token != "" {
			client = buildClient(cfg)
		}

		return tui.Run(client, cfg, Version)
	},
	// PersistentPreRunE runs before every subcommand except those that override it.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// --describe short-circuits everything else, including auth validation,
		// so an agent can introspect the CLI before any credentials are set up.
		if describeFlag {
			return runDescribe(cmd.Root())
		}
		var err error
		cfg, err = loadConfig(cmd)
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

		client = buildClient(cfg)
		return nil
	},
}

// Execute runs the root command. Errors are emitted to stderr as a single
// JSON object (see cmd/errors.go) so AI-agent callers can parse them without
// regex-scraping prose.
func Execute() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		emitError(mapError(err))
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.DefaultPath(), "config file path")
	rootCmd.PersistentFlags().StringVarP(&workspace, "workspace", "w", "", "Bitbucket workspace slug (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&repo, "repo", "r", "", "repository slug (overrides config)")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Atlassian account email / username (overrides config/env)")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Bitbucket API token or app password (overrides config/env)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", formatDefault, "output format: json, gcf, or text")
	rootCmd.PersistentFlags().BoolVar(&describeFlag, "describe", false,
		"emit a JSON capability manifest (commands, flags, schemas) and exit")
}

// workspaceAndRepo returns the resolved workspace and repo, or an error if
// either is missing. Errors wrap config.ErrNoWorkspace / config.ErrNoRepo —
// use errors.Is to detect them.
func workspaceAndRepo() (string, string, error) {
	ws, err := workspaceOnly()
	if err != nil {
		return "", "", err
	}
	if cfg.Repo == "" {
		return "", "", fmt.Errorf("%w — run 'bb setup' or pass --repo", config.ErrNoRepo)
	}
	return ws, cfg.Repo, nil
}

// workspaceOnly returns the resolved workspace or an error wrapping
// config.ErrNoWorkspace. Used by commands that operate at workspace scope
// (e.g. repo list, project list).
func workspaceOnly() (string, error) {
	if cfg.Workspace == "" {
		return "", fmt.Errorf("%w — run 'bb setup' or pass --workspace", config.ErrNoWorkspace)
	}
	return cfg.Workspace, nil
}

// printOutput renders v in the active output format (see cmd/output.go).
// textFn supplies the human-readable rendering used by --format text.
func printOutput(v any, textFn func()) error {
	return renderValue(v, textFn)
}

// loadConfig loads the config file and resolves the effective output format.
// Commands that override PersistentPreRunE but emit output via printOutput MUST
// use this (not config.Load directly) so persisted format / BB_FORMAT and the
// non-TTY guard apply consistently — Cobra runs only the nearest
// PersistentPreRunE and does not chain to the parent's.
func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	c, err := config.Load(cfgFile)
	if err != nil {
		return nil, err
	}
	if err := resolveFormat(cmd, c); err != nil {
		return nil, err
	}
	return c, nil
}

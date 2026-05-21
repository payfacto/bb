package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

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
		cfg, err = config.Load(cfgFile)
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
			client = bitbucket.New(cfg)
			if cfg.HasOAuth() {
				client.SetBearerToken(cfg.Token)
			}
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
		// If stdout is not a TTY and the user did not explicitly set --format,
		// force JSON so piped consumers (agents, scripts) never get text output
		// even if the default ever changes.
		if !cmd.Flags().Changed("format") && !term.IsTerminal(int(os.Stdout.Fd())) {
			format = "json"
		}
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
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Bitbucket username (overrides config/env)")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Bitbucket app password (overrides config/env)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "json", "output format: json or text")
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

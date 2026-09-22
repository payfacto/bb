package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/tui"
	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/internal/git"
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
	// RunE is called when no subcommand is given — launches TUI. It runs
	// AFTER PersistentPreRunE below (defined on this same command - Cobra
	// calls both hooks in sequence for a bare invocation, and RunE only runs
	// once PersistentPreRunE has already succeeded), which has a dedicated
	// bare-invocation branch that resolves cfg/client without hard-failing
	// on missing config/credentials (see its "cmd == cmd.Root()" branch).
	// Reload/re-resolve nothing here — cfg/client are already final, and a
	// nil client is intentional (it tells the TUI to show its setup wizard).
	RunE: func(cmd *cobra.Command, args []string) error {
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
		for _, n := range applyFlagsWithGitOrigin(cfg, flagValues{workspace, repo, username, token}) {
			fmt.Fprintln(os.Stderr, n)
		}

		// The bare `bb` invocation (cmd is the root itself - no subcommand was
		// given) falls through to RunE above, which launches the TUI. The TUI
		// has its own setup wizard for a missing/incomplete config (shown
		// whenever client is nil - see cmd/tui/run.go), so this must never
		// hard-fail here: resolve what credentials we can, but let a nil
		// client through rather than erroring before RunE ever gets a chance
		// to hand off to the wizard. Every other command actually calls the
		// Bitbucket API and needs a fully validated cfg and working
		// credentials up front, so it keeps the strict checks below.
		if cmd == cmd.Root() {
			resolveTokenFromKeyring(cfg, false)
			if cfg.Token != "" {
				client = buildClient(cfg)
			}
			return nil
		}

		if err := cfg.Validate(); err != nil {
			return err
		}
		resolveTokenFromKeyring(cfg, true)
		if err := cfg.ValidateCredentials(); err != nil {
			return err
		}

		client = buildClient(cfg)
		return nil
	},
}

// resolveTokenFromKeyring fills cfg.Token from the OS keyring by username
// when it isn't already set (env var/flag already took priority via
// cfg.Apply). warnOnError controls whether an unexpected keyring error (not
// simply "no token stored" or "no keyring available") is reported on
// stderr - suppressed for the bare invocation, where a broken keyring should
// not spam a user on their way into the TUI's own setup flow.
func resolveTokenFromKeyring(cfg *config.Config, warnOnError bool) {
	if cfg.Token != "" || cfg.Username == "" {
		return
	}
	tok, err := auth.GetToken(cfg.Username)
	if err == nil {
		cfg.Token = tok
		return
	}
	if warnOnError && !errors.Is(err, auth.ErrTokenNotFound) && !errors.Is(err, auth.ErrNoKeyring) {
		fmt.Fprintf(os.Stderr, "warning: keyring error (%v) — set BITBUCKET_TOKEN to authenticate\n", err)
	}
}

// exitCode lets a command signal a non-error, non-zero process exit (e.g.
// `pipeline watch` reporting a failed/blocked/timed-out pipeline while still
// printing its normal result to stdout). A RunE sets this and returns nil;
// Execute honors it after a successful run. Zero means a clean exit.
//
// INVARIANT: this package-level mutable state is safe ONLY because Cobra runs
// RunE synchronously on the single calling goroutine, and Execute reads exitCode
// strictly after rootCmd.Execute() returns - a happens-before edge with no
// concurrent access. It is effectively set-once (today only `pipeline watch`
// writes it). Do NOT run commands concurrently and do NOT introduce a second
// concurrent writer: either would turn this into a data race with no
// compiler/test signal. If concurrent execution is ever needed, thread the exit
// code back through the return path (e.g. a typed *ExitCodeError) instead.
var exitCode int

// Execute runs the root command. Errors are emitted to stderr as a single
// JSON object (see cmd/errors.go) so AI-agent callers can parse them without
// regex-scraping prose. A command that completed but wants a non-zero status
// sets exitCode (see `pipeline watch`).
func Execute() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		renderError(mapError(err))
		os.Exit(1)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
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

// flagValues bundles the raw --workspace/--repo/--username/--token flag
// inputs for a single invocation (empty means "not passed"). A parameter
// object in place of four loose strings, since applyFlagsWithGitOrigin and
// its injectable sibling only ever forward them as a group.
type flagValues struct {
	workspace, repo, username, token string
}

// resolveWorkspaceRepoFromGit fills or overrides cfg.Workspace/cfg.Repo from
// the git origin of the current working directory, for any field not
// explicitly supplied via flag (flagWs/flagRepo empty means "not passed").
// A git-derived value wins over a persisted config-file default because the
// config file holds a single global, user-scoped default that cannot know
// which of the user's many repos they are standing in right now, while the
// git origin is a precise, per-invocation signal — but an explicit flag
// still wins over both. getOrigin is injected for testability (mirrors the
// pattern used by the old inferWorkspaceRepo). Nothing is resolved if
// getOrigin errors, returns empty, or returns a non-bitbucket.org remote —
// cfg keeps its existing (config-file) value in that case, exactly like
// before. Returns one note per field that was filled or overridden, for the
// caller to print on stderr; never silent about an override.
func resolveWorkspaceRepoFromGit(cfg *config.Config, flagWs, flagRepo string, getOrigin func() (string, error)) []string {
	if flagWs != "" && flagRepo != "" {
		return nil // both explicit; no need to shell out to git at all
	}
	url, err := getOrigin()
	if err != nil || url == "" {
		return nil
	}
	gw, gr, ok := git.ParseBitbucketRemote(url)
	if !ok {
		return nil
	}
	var notes []string
	if flagWs == "" {
		cfg.Workspace, notes = resolveGitField(notes, "workspace", cfg.Workspace, gw)
	}
	if flagRepo == "" {
		cfg.Repo, notes = resolveGitField(notes, "repo", cfg.Repo, gr)
	}
	return notes
}

// resolveGitField applies the shared workspace/repo resolution rule for one
// field: fill an empty config value from git ("inferred"), override a
// differing one ("using ... config default was ..."), or leave an
// already-matching value alone — appending a note to notes in the first two
// cases. Extracted so resolveWorkspaceRepoFromGit doesn't repeat this
// three-way branch once per field.
func resolveGitField(notes []string, flagName, configValue, gitValue string) (string, []string) {
	switch {
	case configValue == "":
		notes = append(notes, fmt.Sprintf("note: inferred --%s=%s from git origin", flagName, gitValue))
	case configValue != gitValue:
		notes = append(notes, fmt.Sprintf("note: using --%s=%s from git origin (config default was %s)", flagName, gitValue, configValue))
	}
	return gitValue, notes
}

// gitOriginURL is the git-origin lookup applyFlagsWithGitOrigin uses. A
// package-level var (not a direct git.OriginURL call) so tests can stub it —
// otherwise a test run from inside a real bitbucket.org checkout (this repo)
// would have its own origin silently resolve workspace/repo, defeating any
// test that means to simulate a user with nothing configured.
var gitOriginURL = git.OriginURL

// applyFlagsWithGitOrigin is the shared entry point every PersistentPreRunE
// (and the bare-TUI-launch RunE) must call instead of raw cfg.Apply: it lets
// the git origin of the cwd resolve workspace/repo first (see
// resolveWorkspaceRepoFromGit), then applies explicit flags on top (highest
// priority, unchanged), and returns notes for the caller to print on stderr.
func applyFlagsWithGitOrigin(cfg *config.Config, flags flagValues) []string {
	return applyFlagsWithGitOriginUsing(cfg, flags, gitOriginURL)
}

// applyFlagsWithGitOriginUsing is applyFlagsWithGitOrigin with an injectable
// getOrigin, kept unexported so tests can exercise the git-origin resolution
// path without a real git checkout — mirroring the split between
// resolveWorkspaceRepoFromGit (pure/injectable) and its git.OriginURL-bound
// public caller.
func applyFlagsWithGitOriginUsing(cfg *config.Config, flags flagValues, getOrigin func() (string, error)) []string {
	notes := resolveWorkspaceRepoFromGit(cfg, flags.workspace, flags.repo, getOrigin)
	cfg.Apply(flags.workspace, flags.repo, flags.username, flags.token)
	return notes
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

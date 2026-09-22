package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/payfacto/bb/internal/config"
)

func TestResolveWorkspaceRepoFromGit(t *testing.T) {
	const bbURL = "git@bitbucket.org:payfacto/bb.git"

	tests := []struct {
		name             string
		cfgWs, cfgRepo   string
		flagWs, flagRepo string
		getOrigin        func() (string, error)
		wantWs           string
		wantRepo         string
		wantNotes        int
		wantNoteContains []string
	}{
		{
			name:  "both flags set - no lookup, cfg untouched",
			cfgWs: "payfactopay", cfgRepo: "android-terminal-payment",
			flagWs: "acme", flagRepo: "widgets",
			getOrigin: func() (string, error) {
				t.Fatal("getOrigin should not be called when both flags are set")
				return "", nil
			},
			wantWs: "payfactopay", wantRepo: "android-terminal-payment",
			wantNotes: 0,
		},
		{
			name:  "no git repo - origin error - cfg values retained",
			cfgWs: "payfactopay", cfgRepo: "android-terminal-payment",
			getOrigin: errOrigin,
			wantWs:    "payfactopay", wantRepo: "android-terminal-payment",
			wantNotes: 0,
		},
		{
			name:  "origin resolves but empty string - cfg values retained",
			cfgWs: "payfactopay", cfgRepo: "android-terminal-payment",
			getOrigin: okOrigin(""),
			wantWs:    "payfactopay", wantRepo: "android-terminal-payment",
			wantNotes: 0,
		},
		{
			name:  "non-bitbucket.org origin - cfg values retained",
			cfgWs: "payfactopay", cfgRepo: "android-terminal-payment",
			getOrigin: okOrigin("git@github.com:acme/x.git"),
			wantWs:    "payfactopay", wantRepo: "android-terminal-payment",
			wantNotes: 0,
		},
		{
			name:  "cfg empty for both fields - filled from git, inferred notes",
			cfgWs: "", cfgRepo: "",
			getOrigin: okOrigin(bbURL),
			wantWs:    "payfacto", wantRepo: "bb",
			wantNotes:        2,
			wantNoteContains: []string{"inferred --workspace=payfacto", "inferred --repo=bb"},
		},
		{
			name:  "cfg empty for workspace only - only workspace filled",
			cfgWs: "", cfgRepo: "android-terminal-payment",
			getOrigin: okOrigin(bbURL),
			wantWs:    "payfacto", wantRepo: "bb",
			wantNotes:        2,
			wantNoteContains: []string{"inferred --workspace=payfacto", "using --repo=bb"},
		},
		{
			name:  "cfg empty for repo only - only repo filled",
			cfgWs: "payfactopay", cfgRepo: "",
			getOrigin: okOrigin(bbURL),
			wantWs:    "payfacto", wantRepo: "bb",
			wantNotes:        2,
			wantNoteContains: []string{"using --workspace=payfacto", "inferred --repo=bb"},
		},
		{
			// The actual bug scenario: a stale global default in ~/.bbcloud.yaml
			// must be overridden by the git origin of the cwd, not silently win.
			name:  "cfg has different values for both fields - both overridden with notes",
			cfgWs: "payfactopay", cfgRepo: "android-terminal-payment",
			getOrigin: okOrigin(bbURL),
			wantWs:    "payfacto", wantRepo: "bb",
			wantNotes: 2,
			wantNoteContains: []string{
				"--workspace=payfacto", "payfactopay",
				"--repo=bb", "android-terminal-payment",
			},
		},
		{
			name:  "explicit --workspace flag set, --repo empty - only repo resolved from git",
			cfgWs: "payfactopay", cfgRepo: "android-terminal-payment",
			flagWs:    "acme",
			getOrigin: okOrigin(bbURL),
			wantWs:    "payfactopay", wantRepo: "bb",
			wantNotes:        1,
			wantNoteContains: []string{"--repo=bb", "android-terminal-payment"},
		},
		{
			// Regression guard flagged in code review: when the config default
			// already matches what git origin resolves to, the switch in
			// resolveWorkspaceRepoFromGit must fall through both cases silently
			// (neither "inferred" nor "using ... overrides" applies) rather than
			// emitting a spurious note about a no-op "override".
			name:  "cfg already equals git-derived value - no-op, no notes",
			cfgWs: "payfacto", cfgRepo: "bb",
			getOrigin: okOrigin(bbURL),
			wantWs:    "payfacto", wantRepo: "bb",
			wantNotes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Workspace: tt.cfgWs, Repo: tt.cfgRepo}
			notes := resolveWorkspaceRepoFromGit(cfg, tt.flagWs, tt.flagRepo, tt.getOrigin)
			if cfg.Workspace != tt.wantWs || cfg.Repo != tt.wantRepo {
				t.Errorf("got cfg.Workspace=%q cfg.Repo=%q, want ws=%q repo=%q", cfg.Workspace, cfg.Repo, tt.wantWs, tt.wantRepo)
			}
			if len(notes) != tt.wantNotes {
				t.Errorf("got %d notes %v, want %d", len(notes), notes, tt.wantNotes)
			}
			joined := strings.Join(notes, "\n")
			for _, sub := range tt.wantNoteContains {
				if !strings.Contains(joined, sub) {
					t.Errorf("notes %v missing expected substring %q", notes, sub)
				}
			}
		})
	}
}

// TestResolveWorkspaceRepoFromGit_WorkspaceFlagLeavesFieldUntouched covers the
// partial-flag case in more detail than the table above's note assertions
// can: with --workspace supplied, this function must not touch cfg.Workspace
// at all (Apply applies the flag afterward, out of this function's scope) -
// even when the git origin resolves to a different workspace than both the
// flag and the config default.
func TestResolveWorkspaceRepoFromGit_WorkspaceFlagLeavesFieldUntouched(t *testing.T) {
	cfg := &config.Config{Workspace: "payfactopay", Repo: "android-terminal-payment"}
	notes := resolveWorkspaceRepoFromGit(cfg, "acme", "", okOrigin("git@bitbucket.org:payfacto/bb.git"))
	if cfg.Workspace != "payfactopay" {
		t.Errorf("cfg.Workspace should be untouched by this function when --workspace flag is set, got %q", cfg.Workspace)
	}
	if cfg.Repo != "bb" {
		t.Errorf("cfg.Repo should be resolved from git, got %q", cfg.Repo)
	}
	for _, n := range notes {
		if strings.Contains(n, "workspace") {
			t.Errorf("no note about workspace expected since --workspace flag covers it, got %v", notes)
		}
	}
}

func TestApplyFlagsWithGitOrigin(t *testing.T) {
	// Thin wrapper: resolves from git first, then applies flags on top
	// (highest priority, unchanged). Flags win over both the git-derived value
	// and the config default.
	cfg := &config.Config{Workspace: "payfactopay", Repo: "android-terminal-payment"}
	notes := applyFlagsWithGitOriginUsing(cfg, flagValues{workspace: "acme"}, okOrigin("git@bitbucket.org:payfacto/bb.git"))
	if cfg.Workspace != "acme" {
		t.Errorf("explicit --workspace flag should win, got %q", cfg.Workspace)
	}
	if cfg.Repo != "bb" {
		t.Errorf("repo should be resolved from git since --repo flag was empty, got %q", cfg.Repo)
	}
	if len(notes) != 1 {
		t.Errorf("expected exactly one note (for repo), got %v", notes)
	}
}

// resetRootCmdGlobals saves the package-level config/flag globals PersistentPreRunE
// reads and mutates, and returns a func to restore them - the tests below call
// PersistentPreRunE directly (mirroring TestUserMePreRun_resolvesFormat in
// output_test.go) and must not leak state into other tests in this package.
func resetRootCmdGlobals(t *testing.T) {
	t.Helper()
	oldCfgFile, oldWs, oldRepo, oldUser, oldToken, oldFormat, oldDescribe := cfgFile, workspace, repo, username, token, format, describeFlag
	oldCfg, oldClient := cfg, client
	oldGitOriginURL := gitOriginURL
	t.Cleanup(func() {
		cfgFile, workspace, repo, username, token, format, describeFlag = oldCfgFile, oldWs, oldRepo, oldUser, oldToken, oldFormat, oldDescribe
		cfg, client = oldCfg, oldClient
		gitOriginURL = oldGitOriginURL
	})
	cfgFile = filepath.Join(t.TempDir(), "absent.yaml")
	workspace, repo, username, token = "", "", "", ""
	format = formatDefault
	describeFlag = false
	client = nil
	// config.Load overlays these regardless of cfgFile, so a real developer
	// machine with BITBUCKET_USER/BITBUCKET_TOKEN exported (common for
	// headless/CI-style auth) would otherwise leak real credentials into a
	// test meant to simulate a brand-new user with nothing configured.
	t.Setenv("BITBUCKET_USER", "")
	t.Setenv("BITBUCKET_TOKEN", "")
	// Tests run inside this real bitbucket.org checkout, so the real
	// git.OriginURL would silently resolve workspace/repo from bb's own
	// origin and defeat "simulate a brand-new user with nothing configured".
	// Stub it to look like there's no git origin at all.
	gitOriginURL = func() (string, error) { return "", errors.New("no origin (stubbed for test)") }
}

// TestRootPersistentPreRunE_BareInvocation_NoConfig_SucceedsWithNilClient is
// the regression test for a bug found in code review: PersistentPreRunE used
// to hard-fail cfg.Validate()/ValidateCredentials() for EVERY invocation,
// including the bare `bb` launch - and since Cobra never calls RunE once
// PersistentPreRunE returns an error, a brand-new user with no
// ~/.bbcloud.yaml never reached the TUI at all, contradicting both
// tui.Run's own documented "shows the setup wizard when client is nil"
// behavior (cmd/tui/run.go) and the README's "First run" claim. `cmd ==
// cmd.Root()` is how the shared PersistentPreRunE (defined once, on rootCmd)
// tells the bare invocation apart from a subcommand: Cobra always passes it
// the command actually being executed, not the command that owns the hook.
func TestRootPersistentPreRunE_BareInvocation_NoConfig_SucceedsWithNilClient(t *testing.T) {
	resetRootCmdGlobals(t)

	err := rootCmd.PersistentPreRunE(rootCmd, nil)
	if err != nil {
		t.Fatalf("bare invocation should never hard-fail on missing config, got: %v", err)
	}
	if client != nil {
		t.Errorf("client = %v, want nil (no credentials available) so tui.Run shows its setup wizard", client)
	}
	if cfg == nil {
		t.Fatal("cfg should still be loaded (zero-value config), got nil")
	}
}

// TestRootPersistentPreRunE_Subcommand_NoConfig_StillFailsValidation confirms
// the fix above did not weaken validation for real commands: anything other
// than the bare invocation must still hard-fail when config/credentials are
// missing, exactly as before.
func TestRootPersistentPreRunE_Subcommand_NoConfig_StillFailsValidation(t *testing.T) {
	resetRootCmdGlobals(t)

	err := rootCmd.PersistentPreRunE(prListCmd, nil)
	if err == nil {
		t.Fatal("subcommand invocation with no config should fail validation, got nil error")
	}
	if !errors.Is(err, config.ErrNoWorkspace) {
		t.Errorf("expected config.ErrNoWorkspace, got: %v", err)
	}
}

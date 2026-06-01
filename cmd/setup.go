package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
)

var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{"config"},
	Short:   "Configure bb interactively",
	Long:    "Create or update ~/.bbcloud.yaml with your Bitbucket credentials and defaults.",
	// Override parent PersistentPreRunE — setup doesn't need existing credentials.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		existing, _ := config.Load(path) // treat missing/unreadable config as empty
		if existing == nil {
			existing = &config.Config{}
		}

		r := bufio.NewReader(os.Stdin)

		fmt.Println("bb setup — configure Bitbucket Cloud CLI")
		fmt.Println("Press Enter to keep the current value shown in [brackets].")
		fmt.Println()

		ws := promptLine(r, "Workspace", existing.Workspace)
		defaultRepo := promptLine(r, "Default repo (optional)", existing.Repo)
		fmt.Println("Create an API token (with scopes): https://support.atlassian.com/bitbucket-cloud/docs/create-an-api-token/")
		user := promptLine(r, "Atlassian account email", existing.Username)
		tok := promptPassword("API token", existing.Token)

		authType := existing.AuthType
		if tok != "" {
			authType = "apitoken"
		}

		updated := &config.Config{
			Workspace:     ws,
			Repo:          defaultRepo,
			Username:      user,
			AuthType:      authType,
			OAuthClientID: existing.OAuthClientID,
			// Token deliberately not set — stored in keyring below, not in YAML
		}
		if err := updated.Save(path); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		if tok != "" {
			if err := auth.SetToken(user, tok); err != nil {
				fmt.Fprintf(os.Stderr, "\nwarning: could not store API token in OS keyring (%v)\n", err)
				fmt.Fprintf(os.Stderr, "set BITBUCKET_TOKEN=<api-token> in your environment to authenticate\n")
			} else {
				fmt.Println("API token stored in OS keyring.")
			}
		}

		fmt.Printf("\nConfig saved to %s\n", path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

// promptLine prints a prompt with an optional current value and reads a line.
// Pressing Enter without input keeps the current value.
func promptLine(r *bufio.Reader, label, current string) string {
	if current != "" {
		fmt.Printf("%s [%s]: ", label, current)
	} else {
		fmt.Printf("%s: ", label)
	}
	input, err := r.ReadString('\n')
	if err != nil {
		return current
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return current
	}
	return input
}

// maxSecretLen bounds the accepted secret length. Real API tokens, app
// passwords, and OAuth consumer secrets are well under this; the cap only
// guards against a pathological paste.
const maxSecretLen = 4096

// promptPassword prompts for a secret. On a terminal it reads the input hidden
// (no echo) and, once the user presses Enter, prints a row of '*' so they can
// confirm a value was entered without exposing it. Off a terminal (pipes/CI)
// it falls back to a plain line read.
func promptPassword(label, current string) string {
	prompt := label + ": "
	if current != "" {
		prompt = label + " [****]: "
	}
	fmt.Print(prompt)

	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Println() // ReadPassword consumes the newline; restore it
		if err != nil {
			return current
		}
		val := strings.TrimSpace(string(b))
		if val == "" {
			return current
		}
		if len(val) > maxSecretLen {
			val = val[:maxSecretLen]
		}
		fmt.Println(strings.Repeat("*", utf8.RuneCountInString(val)))
		return val
	}

	// Non-terminal fallback (pipes, CI).
	r := bufio.NewReader(os.Stdin)
	input, err := r.ReadString('\n')
	if err != nil {
		return current
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return current
	}
	return input
}

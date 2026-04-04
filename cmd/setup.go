package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/payfactopay/bb/internal/config"
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
		existing, _ := config.Load(path)
		if existing == nil {
			existing = &config.Config{}
		}

		r := bufio.NewReader(os.Stdin)

		fmt.Println("bb setup — configure Bitbucket Cloud CLI")
		fmt.Println("Press Enter to keep the current value shown in [brackets].")
		fmt.Println()

		ws := promptLine(r, "Workspace", existing.Workspace)
		defaultRepo := promptLine(r, "Default repo (optional)", existing.Repo)
		user := promptLine(r, "Username (email)", existing.Username)
		tok := promptPassword("App password (token)", existing.Token)

		updated := &config.Config{
			Workspace: ws,
			Repo:      defaultRepo,
			Username:  user,
			Token:     tok,
		}
		if err := updated.Save(path); err != nil {
			return fmt.Errorf("save config: %w", err)
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
	input, _ := r.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return current
	}
	return input
}

// promptPassword prompts for a password, masking input when in a terminal.
func promptPassword(label, current string) string {
	if current != "" {
		fmt.Printf("%s [****]: ", label)
	} else {
		fmt.Printf("%s: ", label)
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // newline after masked input
		if err != nil || len(b) == 0 {
			return current
		}
		return string(b)
	}

	// Non-terminal fallback (pipes, CI).
	r := bufio.NewReader(os.Stdin)
	input, _ := r.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return current
	}
	return input
}

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Bitbucket Cloud",
	// Override parent PersistentPreRunE — auth commands manage their own credential state.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via OAuth 2.0 (browser-based)",
	Long: `Authenticate with Bitbucket Cloud using OAuth 2.0.

You need an OAuth consumer registered in your Bitbucket account.
Create one at: Bitbucket > Personal Settings > OAuth consumers > Add consumer
Set the callback URL to: http://localhost

Then run: bb auth login`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		existing, _ := config.Load(path) // treat missing/unreadable config as empty
		if existing == nil {
			existing = &config.Config{}
		}

		r := bufio.NewReader(os.Stdin)
		fmt.Println("bb auth login — authenticate with Bitbucket Cloud via OAuth 2.0")
		fmt.Println()

		clientID := promptLine(r, "OAuth Consumer Key (client_id)", existing.OAuthClientID)
		if clientID == "" {
			return fmt.Errorf("oauth consumer key is required")
		}
		clientSecret := promptPassword("OAuth Consumer Secret", "")
		if clientSecret == "" {
			return fmt.Errorf("oauth consumer secret is required")
		}
		fmt.Println()

		tok, err := auth.Login(clientID, clientSecret)
		if err != nil {
			return fmt.Errorf("oauth login: %w", err)
		}

		if existing.Workspace == "" {
			existing.Workspace = promptLine(r, "Workspace slug", "")
		}

		username, err := fetchUsername(tok.AccessToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not fetch username from API (%v) — enter it manually\n", err)
			existing.Username = promptLine(r, "Username", existing.Username)
		} else {
			existing.Username = username
			fmt.Printf("Authenticated as: %s\n", username)
		}

		if err := auth.SetToken(existing.Username, tok.AccessToken); err != nil {
			fmt.Fprintf(os.Stderr, "\nwarning: could not store token in OS keyring (%v)\n", err)
			fmt.Fprintf(os.Stderr, "run 'bb auth token' after setting BITBUCKET_TOKEN manually, or re-run 'bb auth login'\n")
		}

		existing.OAuthClientID = clientID
		existing.AuthType = "oauth"
		existing.Token = "" // never write to YAML
		if err := existing.Save(path); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("\nConfig saved to %s\n", path)
		fmt.Println("Authentication successful!")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored Bitbucket credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		existing, _ := config.Load(path) // treat missing/unreadable config as empty
		if existing == nil || existing.Username == "" {
			return fmt.Errorf("not authenticated (run 'bb auth login' or 'bb setup')")
		}

		if err := auth.DeleteToken(existing.Username); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not remove token from keyring: %v\n", err)
		}

		existing.AuthType = ""
		existing.OAuthClientID = ""
		existing.Token = ""
		if err := existing.Save(path); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("logged out %s\n", existing.Username)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		existing, _ := config.Load(path) // treat missing/unreadable config as empty
		if existing == nil {
			existing = &config.Config{}
		}

		if existing.Username == "" {
			fmt.Println("not authenticated — run 'bb auth login' or 'bb setup'")
			return nil
		}

		fmt.Printf("Username:  %s\n", existing.Username)
		fmt.Printf("Workspace: %s\n", existing.Workspace)

		authMethod := existing.AuthType
		if authMethod == "" {
			authMethod = "apppassword (legacy)"
		}
		fmt.Printf("Auth type: %s\n", authMethod)

		tok, err := auth.GetToken(existing.Username)
		switch {
		case err == nil:
			fmt.Printf("Token:     %s (from OS keyring)\n", maskToken(tok))
		case errors.Is(err, auth.ErrTokenNotFound):
			if envTok := os.Getenv("BITBUCKET_TOKEN"); envTok != "" {
				fmt.Printf("Token:     %s (from BITBUCKET_TOKEN env var)\n", maskToken(envTok))
			} else {
				fmt.Println("Token:     not found — run 'bb auth login' or 'bb setup'")
			}
		default:
			fmt.Printf("Token:     keyring unavailable (%v)\n", err)
		}
		return nil
	},
}

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the stored authentication token",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		existing, _ := config.Load(path) // treat missing/unreadable config as empty
		if existing == nil || existing.Username == "" {
			return fmt.Errorf("not authenticated (run 'bb auth login' or 'bb setup')")
		}

		tok, err := auth.GetToken(existing.Username)
		if errors.Is(err, auth.ErrTokenNotFound) || errors.Is(err, auth.ErrNoKeyring) {
			tok = os.Getenv("BITBUCKET_TOKEN")
		}
		if tok == "" {
			return fmt.Errorf("no token found (run 'bb auth login' or set BITBUCKET_TOKEN)")
		}
		fmt.Println(tok)
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authTokenCmd)
	rootCmd.AddCommand(authCmd)
}

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
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
Set the callback URL to: http://localhost:8765/callback

The port defaults to 8765; override it with oauth_callback_port in
~/.bbcloud.yaml (the callback URL you register must match it).

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

		tok, err := auth.Login(clientID, clientSecret, existing.OAuthPort())
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

		if err := storeOAuthCredentials(existing.Username, clientSecret, tok); err != nil {
			fmt.Fprintf(os.Stderr, "\nwarning: could not store credentials in OS keyring (%v)\n", err)
			fmt.Fprintf(os.Stderr, "run 'bb auth token' after setting BITBUCKET_TOKEN manually, or re-run 'bb auth login'\n")
			fmt.Fprintf(os.Stderr, "note: without keyring storage, automatic token refresh is unavailable\n")
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

		var authMethod string
		switch existing.AuthType {
		case "apitoken":
			authMethod = "API token"
		case "oauth":
			authMethod = "OAuth 2.0"
		default: // "" or "apppassword"
			authMethod = "app password"
		}

		var tokenStatus string
		tok, err := auth.GetToken(existing.Username)
		switch {
		case err == nil:
			tokenStatus = maskToken(tok) + " (from OS keyring)"
		case errors.Is(err, auth.ErrTokenNotFound):
			if envTok := os.Getenv("BITBUCKET_TOKEN"); envTok != "" {
				tokenStatus = maskToken(envTok) + " (from BITBUCKET_TOKEN env var)"
			} else {
				tokenStatus = "not found — run 'bb auth login' or 'bb setup'"
			}
		default:
			tokenStatus = fmt.Sprintf("keyring unavailable (%v)", err)
		}

		var deprecation string
		hasToken := err == nil || os.Getenv("BITBUCKET_TOKEN") != ""
		if existing.IsLegacyAppPassword() && hasToken {
			deprecation = fmt.Sprintf("DEPRECATED — app passwords stop working %s; run 'bb setup' to switch to an API token", config.AppPasswordDeadline)
		}

		render.AuthStatus(render.AuthStatusInfo{
			Username:    existing.Username,
			Workspace:   existing.Workspace,
			AuthType:    authMethod,
			TokenStatus: tokenStatus,
			Deprecation: deprecation,
		})
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

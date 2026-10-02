// Package session holds the credential-handling steps shared by the CLI
// (cmd) and the TUI: building an API client for a config, refreshing OAuth
// tokens, and persisting OAuth credentials.
package session

import (
	"context"
	"fmt"
	"os"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

// BuildClient constructs the API client for cfg, wiring OAuth bearer auth and
// the 401 token refresher when the config uses OAuth. The refresher closure
// captures the cfg pointer passed in, so callers can pass a snapshot.
func BuildClient(cfg *config.Config) *bitbucket.Client {
	c := bitbucket.New(cfg)
	if cfg.HasOAuth() {
		c.SetBearerToken(cfg.Token)
		c.SetTokenRefresher(func() (string, error) { return refreshAccessToken(cfg) })
	}
	return c
}

// refreshAccessToken obtains a fresh OAuth access token using the stored
// refresh token and consumer secret, persists it to the keyring, and returns
// the new token. It is wired into the HTTP client as the 401 refresher for
// OAuth sessions; the client tracks the returned token itself, so cfg is read
// (username, client ID) but not mutated.
func refreshAccessToken(cfg *config.Config) (string, error) {
	secret, err := auth.GetClientSecret(cfg.Username)
	if err != nil {
		return "", fmt.Errorf("consumer secret unavailable (%w) — re-run 'bb auth login'", err)
	}
	refreshTok, err := auth.GetRefreshToken(cfg.Username)
	if err != nil {
		return "", fmt.Errorf("refresh token unavailable (%w) — re-run 'bb auth login'", err)
	}

	tok, err := auth.RefreshLive(cfg.OAuthClientID, secret, refreshTok)
	if err != nil {
		return "", err
	}

	if err := auth.SetToken(cfg.Username, tok.AccessToken); err != nil {
		return "", err
	}
	// Bitbucket may rotate the refresh token; persist the new one so the next
	// refresh doesn't reuse a stale value. A failure here isn't fatal to this
	// refresh, but warn so a future auth failure is debuggable.
	if tok.RefreshToken != "" {
		if err := auth.SetRefreshToken(cfg.Username, tok.RefreshToken); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not persist rotated refresh token (%v) — you may need to re-run 'bb auth login'\n", err)
		}
	}
	return tok.AccessToken, nil
}

// StoreOAuthCredentials saves the OAuth access token, refresh token, and
// consumer secret for username in the OS keyring. The secret and refresh token
// are what make unattended token refresh possible. A new refresh token is only
// stored when Bitbucket returned one (it does not always rotate it).
func StoreOAuthCredentials(username, clientSecret string, tok *auth.Token) error {
	if err := auth.SetToken(username, tok.AccessToken); err != nil {
		return err
	}
	if tok.RefreshToken != "" {
		if err := auth.SetRefreshToken(username, tok.RefreshToken); err != nil {
			return err
		}
	}
	return auth.SetClientSecret(username, clientSecret)
}

// FetchUsername uses a Bearer token to call the Bitbucket /user endpoint
// and return the authenticated account's username.
func FetchUsername(bearerToken string) (string, error) {
	c := bitbucket.NewWithBearerToken(bearerToken)
	u, err := c.User().Me(context.Background())
	if err != nil {
		return "", fmt.Errorf("fetch username: %w", err)
	}
	return u.Nickname, nil
}

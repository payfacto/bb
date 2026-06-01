package cmd

import (
	"context"
	"fmt"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

// refreshOAuthAccessToken obtains a fresh OAuth access token using the stored
// refresh token and consumer secret, persists it to the keyring, updates
// cfg.Token, and returns the new token. It is wired into the HTTP client as the
// 401 refresher for OAuth sessions.
func refreshOAuthAccessToken(cfg *config.Config) (string, error) {
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
	if tok.RefreshToken != "" {
		_ = auth.SetRefreshToken(cfg.Username, tok.RefreshToken)
	}
	cfg.Token = tok.AccessToken
	return tok.AccessToken, nil
}

// storeOAuthCredentials saves the OAuth access token, refresh token, and
// consumer secret for username in the OS keyring. The secret and refresh token
// are what make unattended token refresh possible. A new refresh token is only
// stored when Bitbucket returned one (it does not always rotate it).
func storeOAuthCredentials(username, clientSecret string, tok *auth.Token) error {
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

// maskToken shows first 4 and last 4 characters with asterisks in between.
func maskToken(tok string) string {
	if len(tok) <= 8 {
		return "********"
	}
	stars := make([]byte, len(tok)-8)
	for i := range stars {
		stars[i] = '*'
	}
	return tok[:4] + string(stars) + tok[len(tok)-4:]
}

// fetchUsername uses a Bearer token to call the Bitbucket /user endpoint
// and return the authenticated account's username.
func fetchUsername(bearerToken string) (string, error) {
	c := bitbucket.NewWithBearerToken(bearerToken)
	u, err := c.User().Me(context.Background())
	if err != nil {
		return "", fmt.Errorf("fetch username: %w", err)
	}
	return u.Nickname, nil
}

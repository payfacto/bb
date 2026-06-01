package auth

import (
	"errors"
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

// Keyring service names. The access token keeps the bare "bb" service for
// backward compatibility; OAuth refresh tokens and consumer secrets live under
// dedicated services so they never collide with the access token entry.
const (
	keyringSvc        = "bb"
	keyringSvcRefresh = "bb:oauth-refresh"
	keyringSvcSecret  = "bb:oauth-secret"
)

// ErrTokenNotFound is returned when no token exists in the keyring for the given user.
var ErrTokenNotFound = errors.New("token not found")

// ErrNoKeyring is returned when the OS keyring is unavailable (e.g., headless Linux).
var ErrNoKeyring = errors.New("OS keyring unavailable")

// SetToken stores the access token for username in the OS keyring.
func SetToken(username, token string) error {
	return set(keyringSvc, username, token)
}

// GetToken retrieves the access token for username from the OS keyring.
// Returns ErrTokenNotFound if no entry exists, ErrNoKeyring if keyring is unavailable.
func GetToken(username string) (string, error) {
	return get(keyringSvc, username)
}

// SetRefreshToken stores the OAuth refresh token for username.
func SetRefreshToken(username, token string) error {
	return set(keyringSvcRefresh, username, token)
}

// GetRefreshToken retrieves the OAuth refresh token for username.
func GetRefreshToken(username string) (string, error) {
	return get(keyringSvcRefresh, username)
}

// SetClientSecret stores the OAuth consumer secret for username. It is needed
// to authenticate refresh-token requests to Bitbucket.
func SetClientSecret(username, secret string) error {
	return set(keyringSvcSecret, username, secret)
}

// GetClientSecret retrieves the OAuth consumer secret for username.
func GetClientSecret(username string) (string, error) {
	return get(keyringSvcSecret, username)
}

// DeleteToken removes the access token, OAuth refresh token, and consumer
// secret for username. Missing entries are not an error.
func DeleteToken(username string) error {
	for _, svc := range []string{keyringSvc, keyringSvcRefresh, keyringSvcSecret} {
		if err := keyring.Delete(svc, username); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return fmt.Errorf("delete %s credential: %w", svc, err)
		}
	}
	return nil
}

func set(service, username, value string) error {
	if err := keyring.Set(service, username, value); err != nil {
		return fmt.Errorf("%w: %v", ErrNoKeyring, err)
	}
	return nil
}

func get(service, username string) (string, error) {
	v, err := keyring.Get(service, username)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrTokenNotFound
		}
		return "", fmt.Errorf("%w: %v", ErrNoKeyring, err)
	}
	return v, nil
}

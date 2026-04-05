package auth

import (
	"errors"
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

const keyringSvc = "bb"

// ErrTokenNotFound is returned when no token exists in the keyring for the given user.
var ErrTokenNotFound = errors.New("token not found")

// ErrNoKeyring is returned when the OS keyring is unavailable (e.g., headless Linux).
var ErrNoKeyring = errors.New("OS keyring unavailable")

// SetToken stores token for username in the OS keyring.
func SetToken(username, token string) error {
	if err := keyring.Set(keyringSvc, username, token); err != nil {
		return fmt.Errorf("%w: %v", ErrNoKeyring, err)
	}
	return nil
}

// GetToken retrieves the token for username from the OS keyring.
// Returns ErrTokenNotFound if no entry exists, ErrNoKeyring if keyring is unavailable.
func GetToken(username string) (string, error) {
	tok, err := keyring.Get(keyringSvc, username)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrTokenNotFound
		}
		return "", fmt.Errorf("%w: %v", ErrNoKeyring, err)
	}
	return tok, nil
}

// DeleteToken removes the stored token for username from the OS keyring.
func DeleteToken(username string) error {
	if err := keyring.Delete(keyringSvc, username); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete token: %w", err)
	}
	return nil
}

package cmd

import (
	"context"
	"fmt"

	"github.com/payfacto/bb/pkg/bitbucket"
)

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

package cmd

import (
	"context"
	"strings"
	"sync"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// currentUserRef is the --reviewer / --participant value meaning "the
// authenticated user".
const currentUserRef = "me"

// resolveUserRef turns a --reviewer / --participant value into the account ID
// or curly-braced UUID the client filters on. "me" is looked up through
// fetchMe; anything else is passed through untouched.
func resolveUserRef(ref string, fetchMe func() (bitbucket.User, error)) (string, error) {
	ref = strings.TrimSpace(ref)
	if !strings.EqualFold(ref, currentUserRef) {
		return ref, nil
	}
	me, err := fetchMe()
	if err != nil {
		return "", err
	}
	if me.AccountID != "" {
		return me.AccountID, nil
	}
	if me.UUID != "" {
		return me.UUID, nil
	}
	return "", newCLIError(ErrCodeAPIError,
		"Bitbucket returned no account_id or uuid for the current user; pass an account ID instead of \"me\"", nil)
}

// fetchCurrentUser returns a lookup of the authenticated user that hits the
// API at most once, so --reviewer me --participant me costs one request.
func fetchCurrentUser() func() (bitbucket.User, error) {
	return sync.OnceValues(func() (bitbucket.User, error) {
		return client.User().Me(context.Background())
	})
}

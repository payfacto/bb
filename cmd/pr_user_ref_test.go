package cmd

import (
	"errors"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestResolveUserRef(t *testing.T) {
	me := func() (bitbucket.User, error) {
		return bitbucket.User{AccountID: "acc-me", UUID: "{u-me}"}, nil
	}
	neverCalled := func() (bitbucket.User, error) {
		t.Fatal("current user must not be fetched for an explicit reference")
		return bitbucket.User{}, nil
	}
	tests := []struct {
		name, ref, want string
		fetchMe         func() (bitbucket.User, error)
	}{
		{"empty stays empty", "", "", neverCalled},
		{"account id passes through", "acc-1", "acc-1", neverCalled},
		{"uuid passes through", "{u-1}", "{u-1}", neverCalled},
		{"me resolves to the account id", "me", "acc-me", me},
		{"me is case-insensitive", "ME", "acc-me", me},
		{"surrounding space is trimmed", " acc-1 ", "acc-1", neverCalled},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveUserRef(tc.ref, tc.fetchMe)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveUserRef_MeFallsBackToUUID(t *testing.T) {
	got, err := resolveUserRef("me", func() (bitbucket.User, error) {
		return bitbucket.User{UUID: "{u-me}"}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "{u-me}" {
		t.Errorf("got %q, want {u-me}", got)
	}
}

func TestResolveUserRef_MeErrors(t *testing.T) {
	boom := errors.New("boom")
	if _, err := resolveUserRef("me", func() (bitbucket.User, error) { return bitbucket.User{}, boom }); !errors.Is(err, boom) {
		t.Errorf("want the fetch error, got %v", err)
	}
	if _, err := resolveUserRef("me", func() (bitbucket.User, error) { return bitbucket.User{}, nil }); err == nil {
		t.Error("want an error when the current user has no account id or uuid")
	}
}

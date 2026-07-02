package cmd

import (
	"fmt"

	"github.com/payfacto/bb/internal/git"
)

// inferWorkspaceRepo fills empty workspace/repo values from the git origin
// remote when it is a bitbucket.org URL. Non-empty inputs are never overridden
// (config/flags win). It returns the resolved values plus one note per inferred
// value for the caller to print on stderr. getOrigin is injected so this logic
// is testable without a real git checkout. Nothing is inferred if getOrigin
// errors, returns an empty string, or returns a remote that is not a
// bitbucket.org URL.
func inferWorkspaceRepo(ws, repo string, getOrigin func() (string, error)) (string, string, []string) {
	if ws != "" && repo != "" {
		return ws, repo, nil
	}
	url, err := getOrigin()
	if err != nil || url == "" {
		return ws, repo, nil
	}
	gw, gr, ok := git.ParseBitbucketRemote(url)
	if !ok {
		return ws, repo, nil
	}
	var notes []string
	if ws == "" {
		ws = gw
		notes = append(notes, fmt.Sprintf("note: inferred --workspace=%s from git origin", gw))
	}
	if repo == "" {
		repo = gr
		notes = append(notes, fmt.Sprintf("note: inferred --repo=%s from git origin", gr))
	}
	return ws, repo, notes
}

// inferFromBranch fills an empty source-branch value from the current git
// branch. A detached HEAD (git prints "HEAD"), an error, or empty output yields
// no inference. getBranch is injected for testability. The returned note is
// empty when nothing was inferred.
func inferFromBranch(branch string, getBranch func() (string, error)) (string, string) {
	if branch != "" {
		return branch, ""
	}
	b, err := getBranch()
	if err != nil || b == "" || b == "HEAD" {
		return branch, ""
	}
	return b, fmt.Sprintf("note: inferred --from-branch=%s from current branch", b)
}

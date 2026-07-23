package cmd

import "fmt"

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

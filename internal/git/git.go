// Package git provides minimal, dependency-free helpers for reading the
// local git context: the origin remote URL, the current branch, and a pure
// parser that extracts a Bitbucket workspace/repo from a remote URL.
package git

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// gitTimeout bounds each git invocation so callers can never hang on a wedged
// git process.
const gitTimeout = 3 * time.Second

// OriginURL returns the URL of the "origin" remote. It errors if git is
// unavailable, the working directory is not a repo, or origin is unset.
func OriginURL() (string, error) {
	return runGit("remote", "get-url", "origin")
}

// CurrentBranch returns the current branch via `git rev-parse --abbrev-ref
// HEAD`. In detached-HEAD state git prints "HEAD"; callers treat that as
// "no branch".
func CurrentBranch() (string, error) {
	return runGit("rev-parse", "--abbrev-ref", "HEAD")
}

func runGit(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ParseBitbucketRemote extracts the Bitbucket workspace and repo slug from a
// git remote URL. It recognises scp-style SSH (git@bitbucket.org:ws/repo.git),
// ssh:// URLs, and HTTPS (https://[user@]bitbucket.org/ws/repo[.git]), with or
// without a trailing ".git". ok is false for any non-bitbucket.org host or an
// unparseable URL.
func ParseBitbucketRemote(remoteURL string) (workspace, repo string, ok bool) {
	s := strings.TrimSpace(remoteURL)
	if s == "" {
		return "", "", false
	}
	// Strip scheme (https://, ssh://) if present.
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	// Strip userinfo (e.g. "git@" or "user@").
	if i := strings.Index(s, "@"); i >= 0 {
		s = s[i+1:]
	}
	// s is now "host<sep>path" where sep is ':' (scp-style) or '/' (URL-style).
	i := strings.IndexAny(s, ":/")
	if i < 0 {
		return "", "", false
	}
	host, path := s[:i], s[i+1:]
	if host != "bitbucket.org" {
		return "", "", false
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

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
// ssh:// URLs including an optional non-standard port
// (ssh://git@bitbucket.org:7999/ws/repo.git), and HTTPS
// (https://[user@]bitbucket.org/ws/repo[.git]), with or without a trailing
// ".git". A remote path with more than two segments (e.g.
// bitbucket.org/ws/repo/extra) is rejected. ok is false for any
// non-bitbucket.org host or an unparseable URL.
func ParseBitbucketRemote(remoteURL string) (workspace, repo string, ok bool) {
	s := strings.TrimSpace(remoteURL)
	if s == "" {
		return "", "", false
	}
	// Strip scheme (https://, ssh://) if present. Track it so a ':' after the
	// host is treated as a port (URL form) rather than the scp path separator.
	hasScheme := false
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
		hasScheme = true
	}
	// Strip userinfo (e.g. "git@" or "user@").
	if i := strings.Index(s, "@"); i >= 0 {
		s = s[i+1:]
	}

	var host, path string
	if hasScheme {
		// URL form: "host[:port]/path". The path always starts at the first '/';
		// a ':' before it is a port and is discarded.
		slash := strings.Index(s, "/")
		if slash < 0 {
			return "", "", false
		}
		host, path = s[:slash], s[slash+1:]
		if c := strings.Index(host, ":"); c >= 0 {
			host = host[:c] // strip ":port"
		}
	} else {
		// scp form: "host:path". The ':' separates host from path (not a port).
		i := strings.IndexAny(s, ":/")
		if i < 0 {
			return "", "", false
		}
		host, path = s[:i], s[i+1:]
	}

	if host != "bitbucket.org" {
		return "", "", false
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

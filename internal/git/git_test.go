package git_test

import (
	"testing"

	"github.com/payfacto/bb/internal/git"
)

func TestParseBitbucketRemote(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		wantWs string
		wantRe string
		wantOK bool
	}{
		{"ssh scp with .git", "git@bitbucket.org:payfacto/bb.git", "payfacto", "bb", true},
		{"ssh scp without .git", "git@bitbucket.org:payfacto/bb", "payfacto", "bb", true},
		{"https", "https://bitbucket.org/payfacto/bb.git", "payfacto", "bb", true},
		{"https with userinfo", "https://jmadore@bitbucket.org/payfacto/bb.git", "payfacto", "bb", true},
		{"ssh url form", "ssh://git@bitbucket.org/payfacto/bb.git", "payfacto", "bb", true},
		{"ssh url form with port", "ssh://git@bitbucket.org:7999/payfacto/bb.git", "payfacto", "bb", true},
		{"trailing slash", "https://bitbucket.org/payfacto/bb/", "payfacto", "bb", true},
		{"extra path segment rejected", "https://bitbucket.org/payfacto/bb/extra", "", "", false},
		{"leading and trailing whitespace", "  git@bitbucket.org:payfacto/bb.git\n", "payfacto", "bb", true},
		{"ssh.bitbucket.org scp", "git@ssh.bitbucket.org:payfacto/bb.git", "payfacto", "bb", true},
		{"ssh.bitbucket.org url form", "ssh://git@ssh.bitbucket.org/payfacto/bb.git", "payfacto", "bb", true},
		{"ssh.bitbucket.org url form with port", "ssh://git@ssh.bitbucket.org:22/payfacto/bb.git", "payfacto", "bb", true},
		{"host case-insensitive", "git@SSH.Bitbucket.org:payfacto/bb.git", "payfacto", "bb", true},
		{"lookalike suffix host rejected", "git@ssh.bitbucket.org.evil.com:payfacto/bb.git", "", "", false},
		{"lookalike prefix host rejected", "git@evilbitbucket.org:payfacto/bb.git", "", "", false},
		{"other bitbucket subdomain rejected", "git@api.bitbucket.org:payfacto/bb.git", "", "", false},
		{"github ssh rejected", "git@github.com:payfacto/bb.git", "", "", false},
		{"github https rejected", "https://github.com/payfacto/bb.git", "", "", false},
		{"empty", "", "", "", false},
		{"no separator", "not-a-url", "", "", false},
		{"host only", "https://bitbucket.org/", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, re, ok := git.ParseBitbucketRemote(tt.url)
			if ws != tt.wantWs || re != tt.wantRe || ok != tt.wantOK {
				t.Errorf("ParseBitbucketRemote(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.url, ws, re, ok, tt.wantWs, tt.wantRe, tt.wantOK)
			}
		})
	}
}

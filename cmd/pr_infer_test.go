package cmd

import "testing"

func okOrigin(url string) func() (string, error) {
	return func() (string, error) { return url, nil }
}

func errOrigin() (string, error) { return "", errNotARepo }

// errNotARepo is a sentinel used only by tests to simulate git failing.
var errNotARepo = errTest("not a git repository")

type errTest string

func (e errTest) Error() string { return string(e) }

func TestInferWorkspaceRepo(t *testing.T) {
	const bbURL = "git@bitbucket.org:payfacto/bb.git"

	tests := []struct {
		name       string
		ws, repo   string
		getOrigin  func() (string, error)
		wantWs     string
		wantRepo   string
		wantNotes  int
	}{
		{"both set - no lookup", "acme", "widgets", okOrigin(bbURL), "acme", "widgets", 0},
		{"repo empty - fill repo", "acme", "", okOrigin(bbURL), "acme", "bb", 1},
		{"ws empty - fill ws", "", "widgets", okOrigin(bbURL), "payfacto", "widgets", 1},
		{"both empty - fill both", "", "", okOrigin(bbURL), "payfacto", "bb", 2},
		{"both empty - github origin skips", "", "", okOrigin("git@github.com:payfacto/bb.git"), "", "", 0},
		{"both empty - origin error", "", "", errOrigin, "", "", 0},
		{"both empty - empty origin", "", "", okOrigin(""), "", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, repo, notes := inferWorkspaceRepo(tt.ws, tt.repo, tt.getOrigin)
			if ws != tt.wantWs || repo != tt.wantRepo {
				t.Errorf("got ws=%q repo=%q, want ws=%q repo=%q", ws, repo, tt.wantWs, tt.wantRepo)
			}
			if len(notes) != tt.wantNotes {
				t.Errorf("got %d notes %v, want %d", len(notes), notes, tt.wantNotes)
			}
		})
	}
}

func TestInferFromBranch(t *testing.T) {
	tests := []struct {
		name       string
		branch     string
		getBranch  func() (string, error)
		wantBranch string
		wantNote   bool
	}{
		{"branch set - no lookup", "feature/x", okOrigin("main"), "feature/x", false},
		{"empty - filled", "", okOrigin("feature/x"), "feature/x", true},
		{"empty - detached HEAD", "", okOrigin("HEAD"), "", false},
		{"empty - lookup error", "", errOrigin, "", false},
		{"empty - empty output", "", okOrigin(""), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, note := inferFromBranch(tt.branch, tt.getBranch)
			if branch != tt.wantBranch {
				t.Errorf("got branch=%q, want %q", branch, tt.wantBranch)
			}
			if (note != "") != tt.wantNote {
				t.Errorf("got note=%q, wantNote=%v", note, tt.wantNote)
			}
		})
	}
}

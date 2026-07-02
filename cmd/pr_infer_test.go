package cmd

import (
	"strings"
	"testing"
)

func okOrigin(url string) func() (string, error) {
	return func() (string, error) { return url, nil }
}

// okBranch mirrors okOrigin for branch mocks so TestInferFromBranch reads
// clearly (a branch name flows through a branch-named helper, not okOrigin).
func okBranch(b string) func() (string, error) {
	return func() (string, error) { return b, nil }
}

func errOrigin() (string, error) { return "", errNotARepo }

// errNotARepo is a sentinel used only by tests to simulate git failing.
var errNotARepo = errTest("not a git repository")

type errTest string

func (e errTest) Error() string { return string(e) }

func TestInferWorkspaceRepo(t *testing.T) {
	const bbURL = "git@bitbucket.org:payfacto/bb.git"

	tests := []struct {
		name             string
		ws, repo         string
		getOrigin        func() (string, error)
		wantWs           string
		wantRepo         string
		wantNotes        int
		wantNoteContains []string
	}{
		{"both set - no lookup", "acme", "widgets", okOrigin(bbURL), "acme", "widgets", 0, nil},
		{"repo empty - fill repo", "acme", "", okOrigin(bbURL), "acme", "bb", 1, []string{"--repo=bb"}},
		{"ws empty - fill ws", "", "widgets", okOrigin(bbURL), "payfacto", "widgets", 1, []string{"--workspace=payfacto"}},
		{"both empty - fill both", "", "", okOrigin(bbURL), "payfacto", "bb", 2, []string{"--workspace=payfacto", "--repo=bb"}},
		{"both empty - github origin skips", "", "", okOrigin("git@github.com:payfacto/bb.git"), "", "", 0, nil},
		{"both empty - origin error", "", "", errOrigin, "", "", 0, nil},
		{"both empty - empty origin", "", "", okOrigin(""), "", "", 0, nil},
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
			joined := strings.Join(notes, "\n")
			for _, sub := range tt.wantNoteContains {
				if !strings.Contains(joined, sub) {
					t.Errorf("notes %v missing expected substring %q", notes, sub)
				}
			}
		})
	}
}

func TestInferFromBranch(t *testing.T) {
	tests := []struct {
		name             string
		branch           string
		getBranch        func() (string, error)
		wantBranch       string
		wantNote         bool
		wantNoteContains string
	}{
		{"branch set - no lookup", "feature/x", okBranch("main"), "feature/x", false, ""},
		{"empty - filled", "", okBranch("feature/x"), "feature/x", true, "--from-branch=feature/x"},
		{"empty - detached HEAD", "", okBranch("HEAD"), "", false, ""},
		{"empty - lookup error", "", errOrigin, "", false, ""},
		{"empty - empty output", "", okBranch(""), "", false, ""},
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
			if tt.wantNoteContains != "" && !strings.Contains(note, tt.wantNoteContains) {
				t.Errorf("note %q missing expected substring %q", note, tt.wantNoteContains)
			}
		})
	}
}

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

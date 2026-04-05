package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestBranchListString_empty(t *testing.T) {
	out := render.BranchListString(nil)
	if !strings.Contains(out, "No branches found.") {
		t.Errorf("got: %q", out)
	}
}

func TestBranchListString_row(t *testing.T) {
	branches := []bitbucket.Branch{{Name: "main", Target: bitbucket.BranchTarget{Hash: "abc123def456"}}}
	out := render.BranchListString(branches)
	if !strings.Contains(out, "main") {
		t.Errorf("expected branch name, got: %q", out)
	}
	if !strings.Contains(out, "abc123de") {
		t.Errorf("expected short hash, got: %q", out)
	}
}

func TestTagListString_empty(t *testing.T) {
	out := render.TagListString(nil)
	if !strings.Contains(out, "No tags found.") {
		t.Errorf("got: %q", out)
	}
}

func TestTagListString_row(t *testing.T) {
	tags := []bitbucket.Tag{{Name: "v1.0.0", Target: bitbucket.BranchTarget{Hash: "abc123def456"}}}
	out := render.TagListString(tags)
	if !strings.Contains(out, "v1.0.0") {
		t.Errorf("expected tag name, got: %q", out)
	}
	if !strings.Contains(out, "abc123de") {
		t.Errorf("expected short hash, got: %q", out)
	}
}

func TestRepoListString_empty(t *testing.T) {
	out := render.RepoListString(nil)
	if !strings.Contains(out, "No repositories found.") {
		t.Errorf("got: %q", out)
	}
}

func TestRepoListString_row(t *testing.T) {
	repos := []bitbucket.Repo{{Slug: "my-repo", Name: "My Repository", IsPrivate: true}}
	out := render.RepoListString(repos)
	if !strings.Contains(out, "my-repo") {
		t.Errorf("expected slug, got: %q", out)
	}
	if !strings.Contains(out, "private") {
		t.Errorf("expected privacy, got: %q", out)
	}
}

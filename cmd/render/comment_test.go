package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestCommentListString_empty(t *testing.T) {
	out := render.CommentListString(nil)
	if !strings.Contains(out, "No comments found.") {
		t.Errorf("got: %q", out)
	}
}

func TestCommentListString_row(t *testing.T) {
	comments := []bitbucket.Comment{{ID: 10, User: bitbucket.Actor{DisplayName: "alice"}, Content: bitbucket.Content{Raw: "Looks good!"}}}
	out := render.CommentListString(comments)
	if !strings.Contains(out, "10") {
		t.Errorf("expected ID, got: %q", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("expected author, got: %q", out)
	}
	if !strings.Contains(out, "Looks good") {
		t.Errorf("expected content, got: %q", out)
	}
}

func TestCommentDetailString_inline(t *testing.T) {
	c := bitbucket.Comment{ID: 10, User: bitbucket.Actor{DisplayName: "alice"}, Content: bitbucket.Content{Raw: "Fix this variable name"}, Inline: &bitbucket.Inline{Path: "cmd/root.go", To: 42}}
	out := render.CommentDetailString(c)
	if !strings.Contains(out, "cmd/root.go") {
		t.Errorf("expected file path, got: %q", out)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("expected line number, got: %q", out)
	}
}

func TestTaskListString_empty(t *testing.T) {
	out := render.TaskListString(nil)
	if !strings.Contains(out, "No tasks found.") {
		t.Errorf("got: %q", out)
	}
}

func TestTaskListString_row(t *testing.T) {
	tasks := []bitbucket.Task{{ID: 7, State: "UNRESOLVED", Description: "Fix the typo on line 42"}}
	out := render.TaskListString(tasks)
	if !strings.Contains(out, "7") {
		t.Errorf("expected ID, got: %q", out)
	}
	if !strings.Contains(out, "UNRESOLVED") {
		t.Errorf("expected state, got: %q", out)
	}
	if !strings.Contains(out, "Fix the typo") {
		t.Errorf("expected description, got: %q", out)
	}
}

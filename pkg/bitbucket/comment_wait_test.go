package bitbucket_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func comment(id int, author, created string) bitbucket.Comment {
	return bitbucket.Comment{ID: id, User: bitbucket.Actor{DisplayName: author}, CreatedOn: created}
}

// commentsHandler serves the PR comment list; perPoll returns the comments for
// the nth request (1-based).
func commentsHandler(t *testing.T, perPoll func(n int32) []bitbucket.Comment) http.HandlerFunc {
	var polls atomic.Int32
	return func(w http.ResponseWriter, r *http.Request) {
		mustEncodeJSON(t, w, map[string]any{"values": perPoll(polls.Add(1))})
	}
}

func waitComments(t *testing.T, h http.Handler, opts bitbucket.WaitCommentsOptions) bitbucket.CommentsWaitResult {
	t.Helper()
	if opts.Interval == 0 {
		opts.Interval = time.Millisecond
	}
	res, err := newTestClient(t, h).Comments("ws", "repo", 33).WaitFor(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestComments_WaitFor_NewOnlyIgnoresExisting proves the default mode only
// reports comments posted after the wait started, even if older ones match.
func TestComments_WaitFor_NewOnlyIgnoresExisting(t *testing.T) {
	old := comment(10, "SYNAPSE-PPP-AI-AGENT-PETRA-2026", "2026-11-10T10:00:00Z")
	h := commentsHandler(t, func(n int32) []bitbucket.Comment {
		if n < 3 {
			return []bitbucket.Comment{old}
		}
		return []bitbucket.Comment{old, comment(11, "Jay", ""), comment(12, "SYNAPSE-PPP-AI-AGENT-PETRA-2026", "")}
	})
	got := waitComments(t, h, bitbucket.WaitCommentsOptions{
		Filter: bitbucket.CommentFilter{AuthorContains: "petra"}, NewOnly: true,
	})
	if got.Status != bitbucket.CommentsFound {
		t.Fatalf("expected found, got %s", got.Status)
	}
	if len(got.Comments) != 1 || got.Comments[0].ID != 12 {
		t.Errorf("expected only comment 12, got %+v", got.Comments)
	}
}

func TestComments_WaitFor_AfterIDMatchesExisting(t *testing.T) {
	h := commentsHandler(t, func(int32) []bitbucket.Comment {
		return []bitbucket.Comment{comment(5, "Petra", ""), comment(9, "Petra", "")}
	})
	got := waitComments(t, h, bitbucket.WaitCommentsOptions{Filter: bitbucket.CommentFilter{AfterID: 5}})
	if len(got.Comments) != 1 || got.Comments[0].ID != 9 {
		t.Errorf("expected comment 9, got %+v", got.Comments)
	}
}

func TestComments_WaitFor_AfterTime(t *testing.T) {
	h := commentsHandler(t, func(int32) []bitbucket.Comment {
		return []bitbucket.Comment{
			comment(1, "a", "2026-11-10T09:00:00.123456+00:00"),
			comment(2, "b", "2026-11-10T11:00:00.5+00:00"),
		}
	})
	after := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC)
	got := waitComments(t, h, bitbucket.WaitCommentsOptions{Filter: bitbucket.CommentFilter{After: after}})
	if len(got.Comments) != 1 || got.Comments[0].ID != 2 {
		t.Errorf("expected comment 2, got %+v", got.Comments)
	}
}

// TestComments_WaitFor_SettleCollectsBurst proves --settle keeps polling after
// the first match so a reviewer's burst of comments is returned together.
func TestComments_WaitFor_SettleCollectsBurst(t *testing.T) {
	h := commentsHandler(t, func(n int32) []bitbucket.Comment {
		switch {
		case n < 2:
			return nil
		case n < 4:
			return []bitbucket.Comment{comment(1, "Petra", "")}
		default:
			return []bitbucket.Comment{comment(1, "Petra", ""), comment(2, "Petra", "")}
		}
	})
	got := waitComments(t, h, bitbucket.WaitCommentsOptions{Settle: 50 * time.Millisecond})
	if len(got.Comments) != 2 {
		t.Errorf("expected both burst comments, got %+v", got.Comments)
	}
}

func TestComments_WaitFor_Timeout(t *testing.T) {
	h := commentsHandler(t, func(int32) []bitbucket.Comment { return nil })
	got := waitComments(t, h, bitbucket.WaitCommentsOptions{Timeout: 15 * time.Millisecond})
	if got.Status != bitbucket.CommentsTimeout {
		t.Errorf("expected timeout, got %s", got.Status)
	}
}

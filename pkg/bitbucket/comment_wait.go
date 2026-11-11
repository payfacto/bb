package bitbucket

import (
	"context"
	"errors"
	"strings"
	"time"
)

// CommentsWaitStatus is the outcome of waiting for pull request comments.
type CommentsWaitStatus string

const (
	CommentsFound   CommentsWaitStatus = "found"
	CommentsTimeout CommentsWaitStatus = "timeout"
)

// CommentFilter selects the comments WaitFor reports. Zero fields don't filter.
type CommentFilter struct {
	// AuthorID matches the author's account ID exactly. Prefer it over
	// AuthorContains for automation: display names are user-editable.
	AuthorID string
	// AuthorContains matches a case-insensitive substring of the author's
	// display name or nickname.
	AuthorContains string
	// AfterID keeps comments with a larger ID (IDs grow with posting order).
	AfterID int
	// After keeps comments created strictly after this time.
	After time.Time
}

// matches reports whether c passes every set filter field. Deleted comments
// never match.
func (f CommentFilter) matches(c Comment) bool {
	if c.Deleted || c.ID <= f.AfterID {
		return false
	}
	if f.AuthorID != "" && c.User.AccountID != f.AuthorID {
		return false
	}
	if !f.After.IsZero() {
		created, err := time.Parse(time.RFC3339Nano, c.CreatedOn)
		if err != nil || !created.After(f.After) {
			return false
		}
	}
	if f.AuthorContains != "" {
		needle := strings.ToLower(f.AuthorContains)
		if !strings.Contains(strings.ToLower(c.User.DisplayName), needle) &&
			!strings.Contains(strings.ToLower(c.User.Nickname), needle) {
			return false
		}
	}
	return true
}

// WaitCommentsOptions configures WaitFor. Timeout <= 0 waits indefinitely.
// NewOnly ignores every comment created before the wait starts (it
// raises Filter.AfterID to the newest such comment). Settle > 0 keeps
// polling after the first match until no new match has arrived for Settle, so
// a reviewer's burst of comments comes back together.
type WaitCommentsOptions struct {
	Interval time.Duration
	Timeout  time.Duration
	Filter   CommentFilter
	NewOnly  bool
	Settle   time.Duration
}

// CommentsWaitResult is what `pr wait-comments` reports when it stops waiting.
type CommentsWaitResult struct {
	Status   CommentsWaitStatus `json:"status"`
	Comments []Comment          `json:"comments"`
}

// WaitFor polls the pull request's comments until at least one matches the
// filter (and, with Settle, until matches stop arriving) or Timeout elapses.
// A timeout with no match is an outcome (CommentsTimeout, nil error); a timeout
// after a match still reports what was found. An external cancellation such as
// Ctrl-C is returned as the error.
func (r *CommentResource) WaitFor(ctx context.Context, opts WaitCommentsOptions) (CommentsWaitResult, error) {
	ctx, cancel := withOptionalTimeout(ctx, opts.Timeout)
	defer cancel()
	filter := opts.Filter
	needBaseline := opts.NewOnly
	waitStart := time.Now() // captured before the first request (see baselineID)
	var matched []Comment
	seen := make(map[int]bool) // IDs of comments already matched
	var lastNewMatch time.Time
	err := pollUntil(ctx, opts.Interval, func(ctx context.Context) (bool, error) {
		comments, err := r.List(ctx)
		if err != nil {
			return false, err
		}
		if needBaseline {
			filter.AfterID = max(filter.AfterID, baselineID(comments, waitStart))
			needBaseline = false
		}
		matched = filterComments(comments, filter)
		for _, c := range matched {
			if !seen[c.ID] {
				seen[c.ID] = true
				lastNewMatch = time.Now()
			}
		}
		return len(matched) > 0 && time.Since(lastNewMatch) >= opts.Settle, nil
	})
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return CommentsWaitResult{}, err
	}
	if len(matched) == 0 {
		return CommentsWaitResult{Status: CommentsTimeout, Comments: []Comment{}}, nil
	}
	return CommentsWaitResult{Status: CommentsFound, Comments: matched}, nil
}

// baselineID is the newest ID among comments created before the wait started.
// Comments created after waitStart stay eligible even though they arrived in
// the first (baseline) response. A comment whose time cannot be parsed counts
// as pre-existing.
func baselineID(comments []Comment, waitStart time.Time) int {
	newest := 0
	for _, c := range comments {
		created, err := time.Parse(time.RFC3339Nano, c.CreatedOn)
		if err == nil && !created.Before(waitStart) {
			continue
		}
		newest = max(newest, c.ID)
	}
	return newest
}

func filterComments(comments []Comment, f CommentFilter) []Comment {
	out := []Comment{}
	for _, c := range comments {
		if f.matches(c) {
			out = append(out, c)
		}
	}
	return out
}

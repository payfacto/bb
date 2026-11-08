package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// CommentListString returns formatted text for a list of PR comments.
func CommentListString(comments []bitbucket.Comment) string {
	if len(comments) == 0 {
		return "No comments found.\n"
	}
	var sb strings.Builder
	for _, c := range comments {
		id := IDStyle.Render(fmt.Sprintf("[%d]", c.ID))
		if c.Resolution != nil {
			id += " " + StateBadge(commentResolvedState)
		}
		sb.WriteString(fmt.Sprintf("  %s  %s: %s\n", id, c.User.DisplayName, truncate(c.Content.Raw, 100)))
	}
	return sb.String()
}

// CommentList prints the formatted comment list to stdout.
func CommentList(comments []bitbucket.Comment) {
	fmt.Print(CommentListString(comments))
}

// CommentDetailString returns formatted text for a single PR comment.
func CommentDetailString(c bitbucket.Comment) string {
	const labelW = 7
	label := func(s string) string {
		return LabelStyle.Render(fmt.Sprintf("  %-*s", labelW, s))
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("ID"), IDStyle.Render(fmt.Sprintf("%d", c.ID))))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Author"), c.User.DisplayName))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Status"), commentStatus(c.Resolution)))
	if c.Inline != nil {
		sb.WriteString(fmt.Sprintf("%s  %s:%d\n", label("File"), c.Inline.Path, c.Inline.To))
	}
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Text"), c.Content.Raw))
	return sb.String()
}

const (
	commentResolvedState = "RESOLVED"
	commentOpenState     = "OPEN"
)

// commentStatus describes a comment thread's state: OPEN, or RESOLVED with
// who resolved it and when.
func commentStatus(res *bitbucket.CommentResolution) string {
	if res == nil {
		return StateBadge(commentOpenState)
	}
	return fmt.Sprintf("%s by %s at %s", StateBadge(commentResolvedState), res.User.DisplayName, res.CreatedOn)
}

// CommentDetail prints the formatted comment detail to stdout.
func CommentDetail(c bitbucket.Comment) {
	fmt.Print(CommentDetailString(c))
}

// CommentResolvedString returns a one-line confirmation that a comment thread
// was resolved, including who resolved it and when.
func CommentResolvedString(commentID int, res bitbucket.CommentResolution) string {
	return fmt.Sprintf("Resolved comment %d (by %s at %s).\n",
		commentID, res.User.DisplayName, res.CreatedOn)
}

// CommentResolved prints the resolve confirmation to stdout.
func CommentResolved(commentID int, res bitbucket.CommentResolution) {
	fmt.Print(CommentResolvedString(commentID, res))
}

// TaskListString returns formatted text for a list of PR tasks.
func TaskListString(tasks []bitbucket.Task) string {
	if len(tasks) == 0 {
		return "No tasks found.\n"
	}
	var sb strings.Builder
	for _, t := range tasks {
		id := IDStyle.Render(fmt.Sprintf("[%d]", t.ID))
		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n", id, StateBadge(t.State), truncate(t.Description, 80)))
	}
	return sb.String()
}

// TaskList prints the formatted task list to stdout.
func TaskList(tasks []bitbucket.Task) {
	fmt.Print(TaskListString(tasks))
}

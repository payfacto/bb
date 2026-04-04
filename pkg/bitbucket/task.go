package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// TaskResource provides operations on pull request tasks.
type TaskResource struct {
	client    *Client
	workspace string
	repo      string
	prID      int
}

func (r *TaskResource) basePath() string {
	return fmt.Sprintf("%s/pullrequests/%d/tasks",
		repoPath(r.workspace, r.repo), r.prID)
}

// List returns all tasks on the pull request.
func (r *TaskResource) List(ctx context.Context) ([]Task, error) {
	q := url.Values{"pagelen": {"100"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Task]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// SetState marks a task as resolved (true) or unresolved (false).
func (r *TaskResource) SetState(ctx context.Context, taskID int, resolved bool) error {
	state := "UNRESOLVED"
	if resolved {
		state = "RESOLVED"
	}
	path := fmt.Sprintf("%s/%d", r.basePath(), taskID)
	_, err := r.client.do(ctx, "PUT", path, map[string]string{"state": state}, nil)
	return err
}

package bitbucket

import (
	"context"
	"errors"
	"net/url"
)

// PipelineScheduleResource provides operations on a repository's scheduled
// (cron-triggered) pipelines.
type PipelineScheduleResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PipelineScheduleResource) basePath() string {
	return repoPath(r.workspace, r.repo) + "/pipelines_config/schedules"
}

func (r *PipelineScheduleResource) schedulePath(uuid string) string {
	return r.basePath() + "/" + url.PathEscape(uuid)
}

// List returns every schedule configured for the repository.
func (r *PipelineScheduleResource) List(ctx context.Context) ([]PipelineSchedule, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	return fetchAllPages[PipelineSchedule](ctx, r.client, r.basePath(), q)
}

// Get returns a single schedule by UUID.
func (r *PipelineScheduleResource) Get(ctx context.Context, uuid string) (PipelineSchedule, error) {
	data, err := r.client.do(ctx, "GET", r.schedulePath(uuid), nil, nil)
	if err != nil {
		return PipelineSchedule{}, err
	}
	return decode[PipelineSchedule](data)
}

// Create adds a schedule. An empty Target.Type defaults to
// PipelineRefTargetType, the only target kind schedules support.
func (r *PipelineScheduleResource) Create(ctx context.Context, input CreatePipelineScheduleInput) (PipelineSchedule, error) {
	if input.Target.Type == "" {
		input.Target.Type = PipelineRefTargetType
	}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return PipelineSchedule{}, err
	}
	return decode[PipelineSchedule](data)
}

// ErrScheduleEnabledRequired is returned by Update when the input does not
// say whether the schedule should be enabled or disabled.
var ErrScheduleEnabledRequired = errors.New("schedule update requires \"enabled\" (true or false)")

// Update changes a schedule. The Bitbucket PUT only accepts the enabled flag;
// target and cron pattern are immutable (delete and re-create to change them).
func (r *PipelineScheduleResource) Update(ctx context.Context, uuid string, input UpdatePipelineScheduleInput) (PipelineSchedule, error) {
	if input.Enabled == nil {
		return PipelineSchedule{}, ErrScheduleEnabledRequired
	}
	data, err := r.client.do(ctx, "PUT", r.schedulePath(uuid), input, nil)
	if err != nil {
		return PipelineSchedule{}, err
	}
	return decode[PipelineSchedule](data)
}

// Delete removes a schedule by UUID.
func (r *PipelineScheduleResource) Delete(ctx context.Context, uuid string) error {
	_, err := r.client.do(ctx, "DELETE", r.schedulePath(uuid), nil, nil)
	return err
}

// Executions returns the recorded runs of a schedule.
func (r *PipelineScheduleResource) Executions(ctx context.Context, uuid string) ([]PipelineScheduleExecution, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	return fetchAllPages[PipelineScheduleExecution](ctx, r.client, r.schedulePath(uuid)+"/executions", q)
}

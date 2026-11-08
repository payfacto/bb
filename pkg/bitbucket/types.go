package bitbucket

import "encoding/json"

// PR represents a Bitbucket pull request.
type PR struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	State        string   `json:"state"`
	Draft        bool     `json:"draft"`
	Description  string   `json:"description"`
	Source       Endpoint `json:"source"`
	Destination  Endpoint `json:"destination"`
	Author       Actor    `json:"author"`
	Reviewers    []Actor  `json:"reviewers"`
	CommentCount int      `json:"comment_count,omitempty"`
	TaskCount    int      `json:"task_count,omitempty"`
	CreatedOn    string   `json:"created_on,omitempty"`
	UpdatedOn    string   `json:"updated_on,omitempty"`
	Links        Links    `json:"links"`
}

// PRListOptions holds the filters for listing pull requests. All fields are
// optional; the zero value lists open PRs in the endpoint's default order.
type PRListOptions struct {
	State        string // OPEN, MERGED, DECLINED, SUPERSEDED (empty = API default)
	SourceBranch string // exact source branch name to filter by
	Sort         string // Bitbucket sort field, "-" prefix for descending (e.g. -updated_on)
	Since        string // lower bound on created_on (ISO-8601); empty = no lower bound
	Until        string // upper bound on created_on (ISO-8601); empty = no upper bound
	Query        string // text matched against title/description via BBQL "~"; empty = no text filter
	Limit        int    // cap on results; <= 0 returns all pages
}

// UpdatePRReviewersInput is the minimal body for adding a reviewer to a PR.
type UpdatePRReviewersInput struct {
	Title     string  `json:"title"`
	Reviewers []Actor `json:"reviewers"`
}

// Endpoint is a branch reference used in PR source/destination.
type Endpoint struct {
	Branch struct {
		Name string `json:"name"`
	} `json:"branch"`
}

// NewEndpoint builds an Endpoint from a branch name.
func NewEndpoint(branchName string) Endpoint {
	var e Endpoint
	e.Branch.Name = branchName
	return e
}

// Actor is a Bitbucket user reference.
type Actor struct {
	AccountID   string `json:"account_id,omitempty"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname,omitempty"`
}

// CloneLink holds a single clone URL (SSH or HTTPS).
type CloneLink struct {
	Href string `json:"href"`
	Name string `json:"name"` // "ssh" or "https"
}

// Links holds href references returned by the API.
type Links struct {
	HTML struct {
		Href string `json:"href"`
	} `json:"html"`
	Clone []CloneLink `json:"clone,omitempty"`
}

// Comment represents a PR comment.
type Comment struct {
	ID        int     `json:"id"`
	Content   Content `json:"content"`
	User      Actor   `json:"user"`
	Inline    *Inline `json:"inline,omitempty"`
	Parent    *Parent `json:"parent,omitempty"`
	CreatedOn string  `json:"created_on,omitempty"`
	UpdatedOn string  `json:"updated_on,omitempty"`
}

// CommentResolution records who resolved a comment thread and when.
type CommentResolution struct {
	Type      string `json:"type"`
	User      Actor  `json:"user"`
	CreatedOn string `json:"created_on,omitempty"`
}

// Content holds the raw text of a comment.
type Content struct {
	Raw string `json:"raw"`
}

// Inline identifies the file and line for an inline comment.
type Inline struct {
	Path string `json:"path"`
	To   int    `json:"to"`
}

// Parent is a reference to a parent comment for replies.
type Parent struct {
	ID int `json:"id"`
}

// Task represents a PR task.
type Task struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	State       string `json:"state"` // RESOLVED or UNRESOLVED
}

// CreatePRInput holds the request body for creating a PR.
type CreatePRInput struct {
	Title             string   `json:"title"`
	Description       string   `json:"description,omitempty"`
	Source            Endpoint `json:"source"`
	Destination       Endpoint `json:"destination"`
	CloseSourceBranch bool     `json:"close_source_branch"`
	Draft             bool     `json:"draft,omitempty"`
}

// UpdatePRInput holds the editable fields for updating a PR. Each field is a
// pointer so callers can distinguish "not provided" from an explicit value: a
// nil pointer keeps the PR's current value in Update, while a non-nil pointer
// sets the field - including an empty string, which CLEARS the description.
// Title cannot be cleared (Bitbucket requires a non-empty title); the cmd layer
// rejects an empty title. Draft follows the same rule: nil leaves the draft
// state unchanged, true marks the PR as draft, false marks it ready.
type UpdatePRInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Draft       *bool   `json:"draft,omitempty"`
}

// AddCommentInput holds the request body for adding a comment.
type AddCommentInput struct {
	Content Content `json:"content"`
	Inline  *Inline `json:"inline,omitempty"`
	Parent  *Parent `json:"parent,omitempty"`
}

// paged is a generic Bitbucket paged response container.
type paged[T any] struct {
	Values []T    `json:"values"`
	Next   string `json:"next"`
}

// Pipeline types

type Pipeline struct {
	UUID        string         `json:"uuid"`
	BuildNumber int            `json:"build_number"`
	State       PipelineState  `json:"state"`
	Target      PipelineTarget `json:"target"`
	// Branch and PRID are normalized from Target on decode: Branch is the PR
	// source branch for pull-request runs and Target.RefName otherwise; PRID is
	// the pull request id (0, omitted, for non-PR runs).
	Branch      string `json:"branch"`
	PRID        int    `json:"pr_id,omitempty"`
	CreatedOn   string `json:"created_on"`
	CompletedOn string `json:"completed_on"`
}

// UnmarshalJSON decodes a pipeline and fills the normalized Branch and PRID.
func (p *Pipeline) UnmarshalJSON(data []byte) error {
	type plain Pipeline
	var raw plain
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = Pipeline(raw)
	p.Branch = p.Target.BranchName()
	p.PRID = p.Target.PullRequestID()
	return nil
}

type PipelineState struct {
	Name   string          `json:"name"`
	Result *PipelineResult `json:"result,omitempty"`
	Stage  *PipelineStage  `json:"stage,omitempty"`
}

type PipelineResult struct {
	Name string `json:"name"`
}

type PipelineStage struct {
	Name string `json:"name"`
}

type PipelineTarget struct {
	RefType string          `json:"ref_type"`
	RefName string          `json:"ref_name"`
	Commit  *PipelineCommit `json:"commit,omitempty"`
	// Pull-request runs (type pipeline_pullrequest_target) carry these instead
	// of ref_type/ref_name; Source and Destination are plain branch names.
	Source      string                     `json:"source,omitempty"`
	Destination string                     `json:"destination,omitempty"`
	PullRequest *PipelineTargetPullRequest `json:"pullrequest,omitempty"`
}

// PipelineTargetPullRequest identifies the pull request that triggered a run.
type PipelineTargetPullRequest struct {
	ID int `json:"id"`
}

// BranchName returns the PR source branch for pull-request runs, else RefName.
func (t PipelineTarget) BranchName() string {
	if t.Source != "" {
		return t.Source
	}
	return t.RefName
}

// PullRequestID returns the triggering pull request id, or 0 for non-PR runs.
func (t PipelineTarget) PullRequestID() int {
	if t.PullRequest == nil {
		return 0
	}
	return t.PullRequest.ID
}

type PipelineCommit struct {
	Hash string `json:"hash"`
}

type PipelineStep struct {
	UUID        string        `json:"uuid"`
	Name        string        `json:"name"`
	State       PipelineState `json:"state"`
	StartedOn   string        `json:"started_on"`
	CompletedOn string        `json:"completed_on"`
}

// PipelineWatchStatus is the terminal outcome reported by `pipeline watch`.
type PipelineWatchStatus string

const (
	WatchSuccess PipelineWatchStatus = "success"
	WatchFailed  PipelineWatchStatus = "failed"
	WatchBlocked PipelineWatchStatus = "blocked"
	WatchTimeout PipelineWatchStatus = "timeout"
)

// ManualGate identifies the step a pipeline is paused on and the Bitbucket web
// URL where it can be resumed. The public API cannot resume a manual step (see
// feature request BCLOUD-20050), so watch surfaces the gate instead.
type ManualGate struct {
	Step string `json:"step"`
	URL  string `json:"url"`
}

// PipelineWatchResult is the single object `pipeline watch` emits when a
// pipeline reaches a terminal state.
type PipelineWatchResult struct {
	Pipeline   Pipeline            `json:"pipeline"`
	Steps      []PipelineStep      `json:"steps"`
	Status     PipelineWatchStatus `json:"status"`
	ManualGate *ManualGate         `json:"manual_gate,omitempty"`
}

// TestReportSummary holds the per-step test counts Bitbucket parses from the
// JUnit-style reports a pipeline step publishes.
type TestReportSummary struct {
	UUID            string `json:"uuid"`
	StepUUID        string `json:"step_uuid"`
	TotalCases      int    `json:"number_of_test_cases"`
	SuccessfulCases int    `json:"number_of_successful_test_cases"`
	FailedCases     int    `json:"number_of_failed_test_cases"`
	ErrorCases      int    `json:"number_of_error_test_cases"`
	SkippedCases    int    `json:"number_of_skipped_test_cases"`
}

// StepTestReport pairs a pipeline step with its test report summary. Report is
// nil when the step published no test report (or did not run).
type StepTestReport struct {
	StepUUID string             `json:"step_uuid"`
	StepName string             `json:"step_name"`
	Report   *TestReportSummary `json:"report"`
}

// FlexString decodes a JSON string or number into a string. The test report
// endpoints are untyped in Bitbucket's OpenAPI spec, so fields whose encoding
// is not documented (e.g. a test case duration) accept either form.
type FlexString string

// UnmarshalJSON accepts a JSON string, number, or null.
func (f *FlexString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*f = FlexString(n.String())
	return nil
}

// TestCase is a single test case from a step's test report.
type TestCase struct {
	UUID               string     `json:"uuid"`
	Name               string     `json:"name"`
	ClassName          string     `json:"class_name,omitempty"`
	PackageName        string     `json:"package_name,omitempty"`
	FullyQualifiedName string     `json:"fully_qualified_name,omitempty"`
	Status             string     `json:"status"`
	Duration           FlexString `json:"duration,omitempty"`
}

// TestCaseReason is the failure output Bitbucket recorded for a test case.
type TestCaseReason struct {
	Message string `json:"message,omitempty"`
	Output  string `json:"output,omitempty"`
}

// StepTestCase is a test case annotated with the step it ran in and, when
// requested, its failure reasons.
type StepTestCase struct {
	StepUUID string `json:"step_uuid"`
	StepName string `json:"step_name"`
	TestCase
	Reasons []TestCaseReason `json:"reasons,omitempty"`
}

// PipelineVariable represents a repository-level pipeline variable.
type PipelineVariable struct {
	UUID    string `json:"uuid"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

// CreatePipelineVariableInput is the request body for creating a pipeline variable.
type CreatePipelineVariableInput struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

// PipelineRefTargetType is the Bitbucket object type of a branch/tag pipeline target.
const PipelineRefTargetType = "pipeline_ref_target"

// PipelineSchedule is a cron-triggered pipeline run. CronPattern is a 7-field
// expression with second precision, evaluated in UTC (e.g. "0 0 12 * * ? *"
// runs at 12:00 UTC every day).
type PipelineSchedule struct {
	UUID        string                 `json:"uuid"`
	Enabled     bool                   `json:"enabled"`
	CronPattern string                 `json:"cron_pattern"`
	Target      PipelineScheduleTarget `json:"target"`
	CreatedOn   string                 `json:"created_on,omitempty"`
	UpdatedOn   string                 `json:"updated_on,omitempty"`
}

// PipelineScheduleTarget is the branch and pipeline definition a schedule runs.
type PipelineScheduleTarget struct {
	Type     string           `json:"type,omitempty"` // pipeline_ref_target
	RefType  string           `json:"ref_type"`       // branch (the only type schedules accept)
	RefName  string           `json:"ref_name"`
	Selector PipelineSelector `json:"selector"`
}

// PipelineSelector picks the bitbucket-pipelines.yml definition to run.
type PipelineSelector struct {
	Type    string `json:"type"` // branches | tags | bookmarks | default | custom
	Pattern string `json:"pattern,omitempty"`
}

// CreatePipelineScheduleInput is the request body for creating a schedule.
// Enabled is a pointer so an unset value is omitted and the API default applies.
type CreatePipelineScheduleInput struct {
	Target      PipelineScheduleTarget `json:"target"`
	CronPattern string                 `json:"cron_pattern"`
	Enabled     *bool                  `json:"enabled,omitempty"`
}

// UpdatePipelineScheduleInput is the request body for updating a schedule;
// enabled is the only mutable field.
type UpdatePipelineScheduleInput struct {
	Enabled bool `json:"enabled"`
}

// PipelineScheduleExecution is one recorded run of a schedule. The API
// publishes no fixed schema for it, so the raw object is passed through.
type PipelineScheduleExecution map[string]any

// PipelineConfig represents the repository's pipelines_config resource. The
// real API response carries more fields (repository, build_image, etc.); this
// type only models Enabled, the one bb exposes, since decode[T] ignores JSON
// fields it doesn't know about.
type PipelineConfig struct {
	Enabled bool `json:"enabled"`
}

type TriggerPipelineInput struct {
	Target    TriggerTarget     `json:"target"`
	Variables []TriggerVariable `json:"variables,omitempty"`
}

type TriggerTarget struct {
	Type     string           `json:"type"`               // pipeline_ref_target | pipeline_commit_target
	RefType  string           `json:"ref_type,omitempty"` // branch | tag (ref targets only)
	RefName  string           `json:"ref_name,omitempty"` // ref targets only
	Commit   *TriggerCommit   `json:"commit,omitempty"`   // commit targets only
	Selector *TriggerSelector `json:"selector,omitempty"` // custom pipeline
}

type TriggerCommit struct {
	Type string `json:"type"` // always "commit"
	Hash string `json:"hash"`
}

type TriggerSelector struct {
	Type    string `json:"type"` // always "custom"
	Pattern string `json:"pattern"`
}

type TriggerVariable struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured,omitempty"`
}

// TriggerRef selects what a pipeline runs against: exactly one of Branch, Tag,
// or Commit. Validation of "exactly one" lives in the cmd layer.
type TriggerRef struct {
	Branch string
	Tag    string
	Commit string
}

// TriggerOptions is the ergonomic input to PipelineResource.Trigger. Custom, when
// set, runs the named custom pipeline; Variables are per-run pipeline variables.
type TriggerOptions struct {
	Ref       TriggerRef
	Custom    string
	Variables []TriggerVariable
}

// Branch types

type Branch struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
	Links  Links        `json:"links"`
}

type BranchTarget struct {
	Hash string `json:"hash"`
}

type CreateBranchInput struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
}

// Commit types

type Commit struct {
	Hash    string         `json:"hash"`
	Date    string         `json:"date"`
	Message string         `json:"message"`
	Author  CommitAuthor   `json:"author"`
	Parents []CommitParent `json:"parents"`
}

type CommitAuthor struct {
	Raw  string `json:"raw"`
	User *Actor `json:"user,omitempty"`
}

type CommitParent struct {
	Hash string `json:"hash"`
}

// CommitStatus is a build/CI status attached to a commit (e.g. a pipeline result).
type CommitStatus struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"` // SUCCESSFUL | FAILED | INPROGRESS | STOPPED
	URL         string `json:"url"`
	RefName     string `json:"refname"`
}

// Report is a Code Insights report attached to a commit by an external tool.
type Report struct {
	UUID              string       `json:"uuid"`
	ExternalID        string       `json:"external_id,omitempty"`
	Title             string       `json:"title"`
	Details           string       `json:"details,omitempty"`
	Reporter          string       `json:"reporter,omitempty"`
	Link              string       `json:"link,omitempty"`
	LogoURL           string       `json:"logo_url,omitempty"`
	RemoteLinkEnabled bool         `json:"remote_link_enabled,omitempty"`
	ReportType        string       `json:"report_type,omitempty"` // SECURITY | COVERAGE | TEST | BUG
	Result            string       `json:"result,omitempty"`      // PASSED | FAILED | PENDING
	Data              []ReportData `json:"data,omitempty"`
	CreatedOn         string       `json:"created_on,omitempty"`
	UpdatedOn         string       `json:"updated_on,omitempty"`
}

// ReportData is one key/value metric displayed on a report. Value's JSON type
// depends on Type (number, boolean, string, or link object).
type ReportData struct {
	Type  string `json:"type,omitempty"` // BOOLEAN | DATE | DURATION | LINK | NUMBER | PERCENTAGE | TEXT
	Title string `json:"title"`
	Value any    `json:"value"`
}

// ReportAnnotation is an individual finding on a report, optionally anchored
// to a file and line.
type ReportAnnotation struct {
	UUID           string `json:"uuid"`
	ExternalID     string `json:"external_id,omitempty"`
	AnnotationType string `json:"annotation_type,omitempty"` // VULNERABILITY | CODE_SMELL | BUG
	Path           string `json:"path,omitempty"`
	Line           int    `json:"line,omitempty"`
	Summary        string `json:"summary"`
	Details        string `json:"details,omitempty"`
	Result         string `json:"result,omitempty"`   // PASSED | FAILED | SKIPPED | IGNORED
	Severity       string `json:"severity,omitempty"` // CRITICAL | HIGH | MEDIUM | LOW
	Link           string `json:"link,omitempty"`
	CreatedOn      string `json:"created_on,omitempty"`
	UpdatedOn      string `json:"updated_on,omitempty"`
}

// User type

type User struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	Links       Links  `json:"links"`
}

// Workspace types

// Workspace represents a Bitbucket Cloud workspace.
type Workspace struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	UUID  string `json:"uuid"`
	Links Links  `json:"links"`
	// Administrator is true when the caller administers the workspace.
	Administrator bool `json:"administrator"`
}

// Project types

// Project represents a Bitbucket workspace project.
type Project struct {
	UUID                    string `json:"uuid"`
	Key                     string `json:"key"`
	Name                    string `json:"name"`
	Description             string `json:"description"`
	IsPrivate               bool   `json:"is_private"`
	HasPubliclyVisibleRepos bool   `json:"has_publicly_visible_repos"`
	CreatedOn               string `json:"created_on"`
	UpdatedOn               string `json:"updated_on"`
	Links                   Links  `json:"links"`
}

// ProjectRef is the minimal project reference embedded in a Repo.
type ProjectRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Repo type

type Repo struct {
	Slug        string      `json:"slug"`
	UUID        string      `json:"uuid"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	IsPrivate   bool        `json:"is_private"`
	FullName    string      `json:"full_name"`
	Links       Links       `json:"links"`
	Project     *ProjectRef `json:"project,omitempty"`
}

// CreateRepoInput is the request body for creating a new repository.
type CreateRepoInput struct {
	Scm         string      `json:"scm"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	IsPrivate   bool        `json:"is_private"`
	Project     *ProjectRef `json:"project,omitempty"`
}

// MainbranchRef is the minimal branch reference used when setting a repo's default branch.
type MainbranchRef struct {
	Name string `json:"name"`
	Type string `json:"type"` // always "branch"
}

// UpdateRepoInput is the request body for updating repository metadata.
type UpdateRepoInput struct {
	Description string         `json:"description,omitempty"`
	IsPrivate   *bool          `json:"is_private,omitempty"`
	Mainbranch  *MainbranchRef `json:"mainbranch,omitempty"`
}

// ForkRepoInput is the request body for forking a repository.
type ForkRepoInput struct {
	Name      string        `json:"name,omitempty"`
	Workspace *WorkspaceRef `json:"workspace,omitempty"`
}

// WorkspaceRef is a minimal workspace reference used in fork requests.
type WorkspaceRef struct {
	Slug string `json:"slug"`
}

// Snippet types

// Snippet represents a Bitbucket snippet (code gist).
type Snippet struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	IsPrivate bool                   `json:"is_private"`
	CreatedOn string                 `json:"created_on"`
	UpdatedOn string                 `json:"updated_on"`
	Owner     Actor                  `json:"owner"`
	Creator   Actor                  `json:"creator"`
	Files     map[string]SnippetFile `json:"files"`
	Links     Links                  `json:"links"`
}

// SnippetFile is a single file within a snippet.
type SnippetFile struct {
	Links SnippetFileLinks `json:"links"`
}

// SnippetFileLinks holds the raw download link for a snippet file.
type SnippetFileLinks struct {
	Raw struct {
		Href string `json:"href"`
	} `json:"raw"`
}

// Webhook types

type Webhook struct {
	UUID        string   `json:"uuid"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
	CreatedAt   string   `json:"created_at"`
	Links       Links    `json:"links"`
}

type CreateWebhookInput struct {
	Description string   `json:"description,omitempty"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
}

// PR Activity types

type Activity struct {
	Comment  *Comment  `json:"comment,omitempty"`
	Approval *Approval `json:"approval,omitempty"`
	Update   *PRUpdate `json:"update,omitempty"`
}

type Approval struct {
	User Actor  `json:"user"`
	Date string `json:"date"`
}

type PRUpdate struct {
	State  string `json:"state"`
	Author Actor  `json:"author"`
	Date   string `json:"date"`
}

// PRStatus type

type PRStatus struct {
	State       string `json:"state"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	CreatedOn   string `json:"created_on"`
}

// PR mergeability types

// PRMergeability summarises whether a pull request can be merged right now.
// Mergeable is derived client-side: true when no check is blocking.
type PRMergeability struct {
	Mergeable bool                `json:"mergeable"`
	Checks    []MergeabilityCheck `json:"checks"`
}

// MergeabilityCheck is one check from the mergeability/checks endpoint. Type
// selects which optional fields are populated: State (pullrequest_state_check),
// Reason (git_mergeability_check), Check/Requirement/Observed
// (standard_merge_check), Check/Message (custom_pre_merge_check,
// custom_on_merge_check), Queued/MergeQueue (merge_queue_check).
type MergeabilityCheck struct {
	Type        string                `json:"type"`
	Status      string                `json:"status"`
	Required    bool                  `json:"required"`
	Blocking    bool                  `json:"blocking"`
	State       string                `json:"state,omitempty"`
	Reason      string                `json:"reason,omitempty"`
	Check       *MergeCheckDefinition `json:"check,omitempty"`
	Requirement map[string]any        `json:"requirement,omitempty"`
	Observed    map[string]any        `json:"observed,omitempty"`
	Message     string                `json:"message,omitempty"`
	Queued      bool                  `json:"queued,omitempty"`
	MergeQueue  *MergeQueue           `json:"merge_queue,omitempty"`
}

// MergeCheckDefinition identifies a standard (Kind) or custom (ID, Name)
// merge check.
type MergeCheckDefinition struct {
	Type string `json:"type"`
	Kind string `json:"kind,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// MergeQueue is the merge queue configured for a pull request's destination.
type MergeQueue struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// FileConflict is a file that conflicts between a pull request's source and
// destination.
type FileConflict struct {
	Path     string `json:"path"`
	Scenario string `json:"scenario"`
	Message  string `json:"message,omitempty"`
}

// Tag types

type Tag struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
	Links  Links        `json:"links"`
}

type CreateTagInput struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
}

// Environment types

type Environment struct {
	UUID            string          `json:"uuid"`
	Name            string          `json:"name"`
	EnvironmentType EnvironmentType `json:"environment_type"`
	Lock            EnvironmentLock `json:"lock"`
}

type EnvironmentType struct {
	Name string `json:"name"` // "Production", "Staging", "Test"
}

type EnvironmentLock struct {
	Name string `json:"name"` // "UNLOCKED", "LOCKED"
}

// CreateEnvironmentInput is the request body for creating a deployment
// environment. EnvironmentType.Name must be one of "Test", "Staging",
// "Production" (case-sensitive).
type CreateEnvironmentInput struct {
	Name            string          `json:"name"`
	EnvironmentType EnvironmentType `json:"environment_type"`
}

// Deployment types

type Deployment struct {
	UUID           string           `json:"uuid"`
	State          DeploymentState  `json:"state"`
	Environment    DeploymentEnvRef `json:"environment"`
	Deployable     Deployable       `json:"deployable"`
	LastUpdateTime string           `json:"last_update_time"`
}

type DeploymentState struct {
	Name   string            `json:"name"`
	Status *DeploymentStatus `json:"status,omitempty"`
}

type DeploymentStatus struct {
	Name string `json:"name"` // "SUCCESSFUL", "FAILED"
}

type DeploymentEnvRef struct {
	UUID string `json:"uuid"`
}

type Deployable struct {
	Commit   *DeployableCommit   `json:"commit,omitempty"`
	Pipeline *DeployablePipeline `json:"pipeline,omitempty"`
}

type DeployableCommit struct {
	Hash string `json:"hash"`
}

type DeployablePipeline struct {
	UUID string `json:"uuid"`
}

// WorkspaceMember type

type WorkspaceMember struct {
	User  User  `json:"user"`
	Links Links `json:"links"`
}

// Download type

type Download struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Links Links  `json:"links"`
}

// DeployKey types

type DeployKey struct {
	ID        int    `json:"id"`
	Label     string `json:"label"`
	Key       string `json:"key"`
	CreatedOn string `json:"created_on"`
	Links     Links  `json:"links"`
}

type AddDeployKeyInput struct {
	Label string `json:"label"`
	Key   string `json:"key"`
}

// BranchRestriction types

type BranchRestriction struct {
	ID              int    `json:"id"`
	Kind            string `json:"kind"`
	BranchMatchKind string `json:"branch_match_kind"`
	Pattern         string `json:"pattern"`
	Value           *int   `json:"value,omitempty"`
	Links           Links  `json:"links"`
}

type CreateBranchRestrictionInput struct {
	Kind            string `json:"kind"`
	BranchMatchKind string `json:"branch_match_kind"`
	Pattern         string `json:"pattern"`
	Value           *int   `json:"value,omitempty"`
}

// Code search types

// CodeSearchResult is one match returned by the code search API.
type CodeSearchResult struct {
	Type              string                   `json:"type"`
	ContentMatchCount int                      `json:"content_match_count"`
	ContentMatches    []CodeSearchContentMatch `json:"content_matches,omitempty"`
	PathMatches       []CodeSearchSegment      `json:"path_matches,omitempty"`
	File              CodeSearchFile           `json:"file"`
}

// CodeSearchContentMatch groups consecutive matched lines within a file.
type CodeSearchContentMatch struct {
	Lines []CodeSearchLine `json:"lines"`
}

// CodeSearchLine is a single line with its segments.
type CodeSearchLine struct {
	Line     int                 `json:"line"`
	Segments []CodeSearchSegment `json:"segments"`
}

// CodeSearchSegment is a run of text; Match is true when it is part of a hit.
type CodeSearchSegment struct {
	Text  string `json:"text"`
	Match bool   `json:"match,omitempty"`
}

// CodeSearchFile identifies the matched file and its origin commit/repo.
type CodeSearchFile struct {
	Path   string            `json:"path"`
	Type   string            `json:"type"`
	Commit *CodeSearchCommit `json:"commit,omitempty"`
}

// CodeSearchCommit is the commit a matched file belongs to.
type CodeSearchCommit struct {
	Hash       string             `json:"hash"`
	Repository *CodeSearchRepoRef `json:"repository,omitempty"`
}

// CodeSearchRepoRef is the minimal repository reference in a code search hit.
type CodeSearchRepoRef struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// CodeSearchOptions configures a code search. Query holds the raw search terms
// (passed through verbatim); Ext/Lang/Repo/Project are folded into Bitbucket
// search modifiers (comma-separated values produce repeated modifiers). Limit
// caps the result count; <= 0 returns all matches.
type CodeSearchOptions struct {
	Query   string
	Ext     string
	Lang    string
	Repo    string
	Project string
	Limit   int
}

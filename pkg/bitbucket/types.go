package bitbucket

// PR represents a Bitbucket pull request.
type PR struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	State        string   `json:"state"`
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
	CreatedOn   string         `json:"created_on"`
	CompletedOn string         `json:"completed_on"`
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

// Issue types

type Issue struct {
	ID        int     `json:"id"`
	Title     string  `json:"title"`
	State     string  `json:"state"`
	Priority  string  `json:"priority"`
	Kind      string  `json:"kind"`
	Content   Content `json:"content"`
	Reporter  Actor   `json:"reporter"`
	Assignee  *Actor  `json:"assignee,omitempty"`
	CreatedOn string  `json:"created_on"`
	UpdatedOn string  `json:"updated_on"`
	Links     Links   `json:"links"`
}

type CreateIssueInput struct {
	Title    string   `json:"title"`
	Content  *Content `json:"content,omitempty"`
	Kind     string   `json:"kind,omitempty"`
	Priority string   `json:"priority,omitempty"`
}

// UpdateIssueInput is the request body for updating an issue's status.
type UpdateIssueInput struct {
	Status string `json:"status"` // "new", "open", "resolved", "on hold", "invalid", "duplicate", "wontfix", "closed"
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

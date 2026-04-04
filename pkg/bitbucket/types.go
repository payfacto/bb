package bitbucket

// PR represents a Bitbucket pull request.
type PR struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	State       string   `json:"state"`
	Description string   `json:"description"`
	Source      Endpoint `json:"source"`
	Destination Endpoint `json:"destination"`
	Author      Actor    `json:"author"`
	Links       Links    `json:"links"`
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
	DisplayName string `json:"display_name"`
}

// Links holds href references returned by the API.
type Links struct {
	HTML struct {
		Href string `json:"href"`
	} `json:"html"`
}

// Comment represents a PR comment.
type Comment struct {
	ID      int     `json:"id"`
	Content Content `json:"content"`
	User    Actor   `json:"user"`
	Inline  *Inline `json:"inline,omitempty"`
	Parent  *Parent `json:"parent,omitempty"`
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
}

// AddCommentInput holds the request body for adding a comment.
type AddCommentInput struct {
	Content Content `json:"content"`
	Inline  *Inline `json:"inline,omitempty"`
	Parent  *Parent `json:"parent,omitempty"`
}

// paged is a generic Bitbucket paged response container.
type paged[T any] struct {
	Values []T `json:"values"`
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
}

type PipelineResult struct {
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

type TriggerPipelineInput struct {
	Target TriggerTarget `json:"target"`
}

type TriggerTarget struct {
	RefType string `json:"ref_type"`
	Type    string `json:"type"`
	RefName string `json:"ref_name"`
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

// Repo type

type Repo struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	FullName    string `json:"full_name"`
	Links       Links  `json:"links"`
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

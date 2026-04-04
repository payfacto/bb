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
	ID      int      `json:"id"`
	Content Content  `json:"content"`
	User    Actor    `json:"user"`
	Inline  *Inline  `json:"inline,omitempty"`
	Parent  *Parent  `json:"parent,omitempty"`
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

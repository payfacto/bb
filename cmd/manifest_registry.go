package cmd

import "github.com/payfacto/bb/pkg/bitbucket"

// This file is data, not logic. It declares every leaf command in the Cobra
// tree alongside the metadata that cannot be inferred by walking Cobra alone:
// action class, output Go type, optional stdin type, example invocation, and
// ordering hints for endpoints whose API has no sort parameter.
//
// The walker, manifest types, and emission live in cmd/describe.go.
//
// Invariants enforced by tests in cmd/describe_test.go:
//   - Every leaf in the Cobra tree appears in commandRegistry.
//   - Every entry in commandRegistry corresponds to a real Cobra leaf (no stale rows).
//   - Every OutputType / StdinType resolves against typeRegistry.

// ResultMap is the synthetic name we expose for the small ad-hoc
// {"result": "...", ...} payloads several mutating commands return.
type ResultMap = map[string]any

// commandRegistry is keyed by space-separated command path
// (e.g. "pr list", "auth status"). MUST contain every leaf in the Cobra tree.
var commandRegistry = map[string]commandSpec{
	// pr ---------------------------------------------------------------
	"pr list":         {Action: actionRead, OutputType: "[]PR", Example: "bb pr list --state OPEN"},
	"pr get":          {Action: actionRead, OutputType: "PR", Example: "bb pr get --pr-id 42"},
	"pr create":       {Action: actionWrite, OutputType: "PR", StdinType: "CreatePRInput", Example: "bb pr create --title 'fix' --from-branch feat --to-branch main"},
	"pr diff":         {Action: actionRead, OutputType: "string", Example: "bb pr diff --pr-id 42"},
	"pr approve":      {Action: actionWrite, OutputType: "ResultMap", Example: "bb pr approve --pr-id 42"},
	"pr merge":        {Action: actionDestructive, OutputType: "ResultMap", Example: "bb pr merge --pr-id 42 --strategy squash"},
	"pr decline":      {Action: actionDestructive, OutputType: "ResultMap", Example: "bb pr decline --pr-id 42"},
	"pr add-reviewer": {Action: actionWrite, OutputType: "ResultMap", Example: "bb pr add-reviewer --id 42 --account-id {uuid}"},
	"pr activity":     {Action: actionRead, OutputType: "[]Activity", Example: "bb pr activity --pr-id 42"},
	"pr statuses":     {Action: actionRead, OutputType: "[]PRStatus", Example: "bb pr statuses --pr-id 42"},

	// pr comment / pr task --------------------------------------------
	"pr comment list":  {Action: actionRead, OutputType: "[]Comment", Example: "bb pr comment list --pr-id 42"},
	"pr comment get":   {Action: actionRead, OutputType: "Comment", Example: "bb pr comment get --pr-id 42 --comment-id 7"},
	"pr comment add":   {Action: actionWrite, OutputType: "Comment", StdinType: "AddCommentInput", Example: "bb pr comment add --pr-id 42 --text 'LGTM'"},
	"pr comment reply": {Action: actionWrite, OutputType: "Comment", Example: "bb pr comment reply --pr-id 42 --parent-id 7 --text 'thanks'"},
	"pr task list":     {Action: actionRead, OutputType: "[]Task", Example: "bb pr task list --pr-id 42"},
	"pr task complete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb pr task complete --pr-id 42 --task-ids 1,2"},
	"pr task reopen":   {Action: actionDestructive, OutputType: "ResultMap", Example: "bb pr task reopen --pr-id 42 --task-ids 1,2"},

	// pipeline ---------------------------------------------------------
	"pipeline list":    {Action: actionRead, OutputType: "[]Pipeline", Example: "bb pipeline list"},
	"pipeline get":     {Action: actionRead, OutputType: "Pipeline", Example: "bb pipeline get --uuid '{uuid}'"},
	"pipeline trigger": {Action: actionWrite, OutputType: "Pipeline", Example: "bb pipeline trigger --branch main"},
	"pipeline stop":    {Action: actionDestructive, OutputType: "ResultMap", Example: "bb pipeline stop --uuid '{uuid}'"},
	"pipeline steps":   {Action: actionRead, OutputType: "[]PipelineStep", Example: "bb pipeline steps --uuid '{uuid}'"},
	"pipeline log":     {Action: actionRead, OutputType: "string", Example: "bb pipeline log --uuid '{uuid}' --step-uuid '{step}'"},

	// pipeline-var -----------------------------------------------------
	"pipeline-var list":   {Action: actionRead, OutputType: "[]PipelineVariable", Ordering: "unspecified", Example: "bb pipeline-var list"},
	"pipeline-var create": {Action: actionWrite, OutputType: "PipelineVariable", StdinType: "CreatePipelineVariableInput", Example: "bb pipeline-var create --key API_KEY --value secret --secured"},
	"pipeline-var delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb pipeline-var delete --uuid '{uuid}'"},

	// branch -----------------------------------------------------------
	"branch list":   {Action: actionRead, OutputType: "[]Branch", Example: "bb branch list"},
	"branch create": {Action: actionWrite, OutputType: "Branch", StdinType: "CreateBranchInput", Example: "bb branch create --name feat/x --target main"},
	"branch delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb branch delete --name feat/x"},

	// tag --------------------------------------------------------------
	"tag list":   {Action: actionRead, OutputType: "[]Tag", Example: "bb tag list"},
	"tag create": {Action: actionWrite, OutputType: "Tag", StdinType: "CreateTagInput", Example: "bb tag create --name v1.0.0 --target {hash}"},
	"tag delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb tag delete --name v1.0.0"},

	// commit -----------------------------------------------------------
	"commit list": {Action: actionRead, OutputType: "[]Commit", Example: "bb commit list --branch main"},
	"commit get":  {Action: actionRead, OutputType: "Commit", Example: "bb commit get --hash {sha}"},

	// file (top-level, sibling of commit) ------------------------------
	"file get": {Action: actionRead, OutputType: "string", Example: "bb file get --ref main --path README.md"},

	// repo -------------------------------------------------------------
	"repo list":   {Action: actionRead, OutputType: "[]Repo", Example: "bb repo list"},
	"repo get":    {Action: actionRead, OutputType: "Repo", Example: "bb repo get my-repo"},
	"repo create": {Action: actionWrite, OutputType: "Repo", StdinType: "CreateRepoInput", Example: "bb repo create new-repo --project KEY"},
	"repo update": {Action: actionWrite, OutputType: "Repo", StdinType: "UpdateRepoInput", Example: "bb repo update my-repo --description 'new desc'"},
	"repo fork":   {Action: actionWrite, OutputType: "Repo", StdinType: "ForkRepoInput", Example: "bb repo fork source-repo --target-workspace ws"},
	"repo delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb repo delete my-repo"},

	// issue ------------------------------------------------------------
	"issue list":   {Action: actionRead, OutputType: "[]Issue", Example: "bb issue list"},
	"issue get":    {Action: actionRead, OutputType: "Issue", Example: "bb issue get --id 7"},
	"issue create": {Action: actionWrite, OutputType: "Issue", StdinType: "CreateIssueInput", Example: "bb issue create --title 'bug' --kind bug"},
	"issue close":  {Action: actionDestructive, OutputType: "Issue", Example: "bb issue close --id 7"},
	"issue reopen": {Action: actionWrite, OutputType: "Issue", Example: "bb issue reopen --id 7"},

	// deployment -------------------------------------------------------
	"deployment list": {Action: actionRead, OutputType: "[]Deployment", Ordering: "unspecified", Example: "bb deployment list"},

	// env --------------------------------------------------------------
	"env list": {Action: actionRead, OutputType: "[]Environment", Ordering: "unspecified", Example: "bb env list"},

	// member -----------------------------------------------------------
	"member list": {Action: actionRead, OutputType: "[]WorkspaceMember", Ordering: "unspecified", Example: "bb member list"},

	// user -------------------------------------------------------------
	"user me": {Action: actionRead, OutputType: "User", Example: "bb user me"},

	// webhook ----------------------------------------------------------
	"webhook list":   {Action: actionRead, OutputType: "[]Webhook", Ordering: "unspecified", Example: "bb webhook list"},
	"webhook create": {Action: actionWrite, OutputType: "Webhook", StdinType: "CreateWebhookInput", Example: "bb webhook create --url https://example.com --events repo:push"},
	"webhook delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb webhook delete --uuid '{uuid}'"},

	// deploy-key -------------------------------------------------------
	"deploy-key list":   {Action: actionRead, OutputType: "[]DeployKey", Ordering: "unspecified", Example: "bb deploy-key list"},
	"deploy-key add":    {Action: actionWrite, OutputType: "DeployKey", StdinType: "AddDeployKeyInput", Example: "bb deploy-key add --label ci --key 'ssh-rsa ...'"},
	"deploy-key delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb deploy-key delete --id 123"},

	// restriction ------------------------------------------------------
	"restriction list":   {Action: actionRead, OutputType: "[]BranchRestriction", Ordering: "unspecified", Example: "bb restriction list"},
	"restriction create": {Action: actionWrite, OutputType: "BranchRestriction", StdinType: "CreateBranchRestrictionInput", Example: "bb restriction create --kind push --pattern main"},
	"restriction delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb restriction delete --id 123"},

	// download ---------------------------------------------------------
	"download list":   {Action: actionRead, OutputType: "[]Download", Ordering: "unspecified", Example: "bb download list"},
	"download upload": {Action: actionWrite, OutputType: "ResultMap", Example: "bb download upload --file ./artifact.zip"},
	"download delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb download delete --name artifact.zip"},

	// project ----------------------------------------------------------
	"project list": {Action: actionRead, OutputType: "[]Project", Example: "bb project list"},
	"project get":  {Action: actionRead, OutputType: "Project", Example: "bb project get KEY"},

	// snippet ----------------------------------------------------------
	"snippet list":   {Action: actionRead, OutputType: "[]Snippet", Ordering: "unspecified", Example: "bb snippet list"},
	"snippet get":    {Action: actionRead, OutputType: "Snippet", Example: "bb snippet get {id}"},
	"snippet create": {Action: actionWrite, OutputType: "Snippet", Example: "bb snippet create --title 'demo' --file demo.txt"},
	"snippet delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb snippet delete {id}"},

	// workspace --------------------------------------------------------
	"workspace list": {Action: actionRead, OutputType: "[]Workspace", Ordering: "unspecified", Example: "bb workspace list"},

	// auth (status/token are machine-callable; login/logout are not) ----
	"auth status": {Action: actionRead, OutputType: "string", Example: "bb auth status"},
	"auth token":  {Action: actionRead, OutputType: "string", Example: "bb auth token"},
	"auth login":  {Skip: true},
	"auth logout": {Skip: true},

	// not agent-callable ----------------------------------------------
	"setup":      {Skip: true},
	"completion": {Skip: true},
}

// typeRegistry maps OutputType / StdinType strings to a zero value used to
// reflect a JSON Schema. Each key must be unique; missing keys are detected
// by TestEveryRegisteredTypeResolves.
var typeRegistry = map[string]any{
	"string":    "",
	"ResultMap": map[string]any{},

	"PR":            bitbucket.PR{},
	"[]PR":          []bitbucket.PR{},
	"CreatePRInput": bitbucket.CreatePRInput{},
	"Activity":      bitbucket.Activity{},
	"[]Activity":    []bitbucket.Activity{},
	"PRStatus":      bitbucket.PRStatus{},
	"[]PRStatus":    []bitbucket.PRStatus{},

	"Comment":         bitbucket.Comment{},
	"[]Comment":       []bitbucket.Comment{},
	"AddCommentInput": bitbucket.AddCommentInput{},
	"Task":            bitbucket.Task{},
	"[]Task":          []bitbucket.Task{},

	"Pipeline":                    bitbucket.Pipeline{},
	"[]Pipeline":                  []bitbucket.Pipeline{},
	"PipelineStep":                bitbucket.PipelineStep{},
	"[]PipelineStep":              []bitbucket.PipelineStep{},
	"PipelineVariable":            bitbucket.PipelineVariable{},
	"[]PipelineVariable":          []bitbucket.PipelineVariable{},
	"CreatePipelineVariableInput": bitbucket.CreatePipelineVariableInput{},

	"Branch":            bitbucket.Branch{},
	"[]Branch":          []bitbucket.Branch{},
	"CreateBranchInput": bitbucket.CreateBranchInput{},

	"Tag":            bitbucket.Tag{},
	"[]Tag":          []bitbucket.Tag{},
	"CreateTagInput": bitbucket.CreateTagInput{},

	"Commit":   bitbucket.Commit{},
	"[]Commit": []bitbucket.Commit{},

	"Repo":            bitbucket.Repo{},
	"[]Repo":          []bitbucket.Repo{},
	"CreateRepoInput": bitbucket.CreateRepoInput{},
	"UpdateRepoInput": bitbucket.UpdateRepoInput{},
	"ForkRepoInput":   bitbucket.ForkRepoInput{},

	"Issue":            bitbucket.Issue{},
	"[]Issue":          []bitbucket.Issue{},
	"CreateIssueInput": bitbucket.CreateIssueInput{},

	"Deployment":   bitbucket.Deployment{},
	"[]Deployment": []bitbucket.Deployment{},

	"Environment":   bitbucket.Environment{},
	"[]Environment": []bitbucket.Environment{},

	"WorkspaceMember":   bitbucket.WorkspaceMember{},
	"[]WorkspaceMember": []bitbucket.WorkspaceMember{},

	"User": bitbucket.User{},

	"Webhook":            bitbucket.Webhook{},
	"[]Webhook":          []bitbucket.Webhook{},
	"CreateWebhookInput": bitbucket.CreateWebhookInput{},

	"DeployKey":         bitbucket.DeployKey{},
	"[]DeployKey":       []bitbucket.DeployKey{},
	"AddDeployKeyInput": bitbucket.AddDeployKeyInput{},

	"BranchRestriction":            bitbucket.BranchRestriction{},
	"[]BranchRestriction":          []bitbucket.BranchRestriction{},
	"CreateBranchRestrictionInput": bitbucket.CreateBranchRestrictionInput{},

	"Download":   bitbucket.Download{},
	"[]Download": []bitbucket.Download{},

	"Project":   bitbucket.Project{},
	"[]Project": []bitbucket.Project{},

	"Snippet":   bitbucket.Snippet{},
	"[]Snippet": []bitbucket.Snippet{},

	"Workspace":   bitbucket.Workspace{},
	"[]Workspace": []bitbucket.Workspace{},
}

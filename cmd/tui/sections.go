package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func buildMenuItems(client *bitbucket.Client, cfg *config.Config) []menuItem {
	ws := cfg.Workspace
	repo := cfg.Repo
	return []menuItem{
		{label: "Pull Requests", description: "List, review, approve, merge PRs", onSelect: func() View {
			return newPRListView(client, ws, repo)
		}},
		{label: "Pipelines", description: "View builds, trigger, check steps", onSelect: func() View {
			return newPipelineListView(client, ws, repo)
		}},
		{label: "Branches", description: "List and manage branches", onSelect: func() View {
			return newSimpleListView("Branches", func(ctx context.Context, _ string) ([]listItem, error) {
				branches, err := client.Branches(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(branches))
				for i, b := range branches {
					hash := b.Target.Hash
					if len(hash) > 8 {
						hash = hash[:8]
					}
					items[i] = listItem{id: hash, title: b.Name, data: b}
				}
				return items, nil
			}, func(item listItem) tea.Cmd {
				b := item.data.(bitbucket.Branch)
				content := render.BranchListString([]bitbucket.Branch{b})
				return pushViewCmd(newTextView("Branch: "+b.Name, content))
			})
		}},
		{label: "Commits", description: "Browse commit history", onSelect: func() View {
			return newCommitListView(client, ws, repo)
		}},
		{label: "Tags", description: "List and manage tags", onSelect: func() View {
			return newSimpleListView("Tags", func(ctx context.Context, _ string) ([]listItem, error) {
				tags, err := client.Tags(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(tags))
				for i, t := range tags {
					hash := t.Target.Hash
					if len(hash) > 8 {
						hash = hash[:8]
					}
					items[i] = listItem{id: hash, title: t.Name, data: t}
				}
				return items, nil
			}, nil)
		}},
		{label: "Issues", description: "Track and manage issues", onSelect: func() View {
			return newIssueListView(client, ws, repo)
		}},
		{label: "Repositories", description: "List workspace repos", onSelect: func() View {
			return newSimpleListView("Repositories", func(ctx context.Context, _ string) ([]listItem, error) {
				repos, err := client.Repos(ws).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(repos))
				for i, r := range repos {
					privacy := "public"
					if r.IsPrivate {
						privacy = "private"
					}
					items[i] = listItem{id: r.Slug, title: r.Name, subtitle: privacy, data: r}
				}
				return items, nil
			}, nil)
		}},
		{label: "Deployments", description: "View deployments", onSelect: func() View {
			return newSimpleListView("Deployments", func(ctx context.Context, _ string) ([]listItem, error) {
				deps, err := client.Deployments(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(deps))
				for i, d := range deps {
					status := d.State.Name
					if d.State.Status != nil {
						status += "/" + d.State.Status.Name
					}
					items[i] = listItem{id: d.UUID, title: status, data: d}
				}
				return items, nil
			}, nil)
		}},
		{label: "Settings", description: "Webhooks, deploy keys, environments, restrictions", onSelect: func() View {
			return newMenuModel(ws, repo, buildSettingsItems(client, ws, repo))
		}},
		{label: "Members", description: "Workspace members", onSelect: func() View {
			return newSimpleListView("Members", func(ctx context.Context, _ string) ([]listItem, error) {
				members, err := client.Members(ws).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(members))
				for i, m := range members {
					items[i] = listItem{id: m.User.Nickname, title: m.User.DisplayName, subtitle: m.User.AccountID, data: m}
				}
				return items, nil
			}, nil)
		}},
	}
}

func buildSettingsItems(client *bitbucket.Client, ws, repo string) []menuItem {
	return []menuItem{
		{label: "Webhooks", description: "Manage repository webhooks", onSelect: func() View {
			return newSimpleListView("Webhooks", func(ctx context.Context, _ string) ([]listItem, error) {
				hooks, err := client.Webhooks(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(hooks))
				for i, h := range hooks {
					active := "inactive"
					if h.Active {
						active = "active"
					}
					items[i] = listItem{id: h.UUID, title: h.URL, subtitle: active, data: h}
				}
				return items, nil
			}, nil)
		}},
		{label: "Deploy Keys", description: "Manage deploy keys", onSelect: func() View {
			return newSimpleListView("Deploy Keys", func(ctx context.Context, _ string) ([]listItem, error) {
				keys, err := client.DeployKeys(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(keys))
				for i, k := range keys {
					items[i] = listItem{id: fmt.Sprintf("%d", k.ID), title: k.Label, data: k}
				}
				return items, nil
			}, nil)
		}},
		{label: "Environments", description: "Deployment environments", onSelect: func() View {
			return newSimpleListView("Environments", func(ctx context.Context, _ string) ([]listItem, error) {
				envs, err := client.Environments(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(envs))
				for i, e := range envs {
					lock := ""
					if e.Lock.Name == "LOCKED" {
						lock = " [LOCKED]"
					}
					items[i] = listItem{id: e.UUID, title: e.Name + lock, subtitle: e.EnvironmentType.Name, data: e}
				}
				return items, nil
			}, nil)
		}},
		{label: "Restrictions", description: "Branch restrictions", onSelect: func() View {
			return newSimpleListView("Restrictions", func(ctx context.Context, _ string) ([]listItem, error) {
				restrictions, err := client.Restrictions(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(restrictions))
				for i, r := range restrictions {
					items[i] = listItem{id: fmt.Sprintf("%d", r.ID), title: r.Kind, subtitle: r.Pattern, data: r}
				}
				return items, nil
			}, nil)
		}},
		{label: "Downloads", description: "Download artifacts", onSelect: func() View {
			return newSimpleListView("Downloads", func(ctx context.Context, _ string) ([]listItem, error) {
				downloads, err := client.Downloads(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(downloads))
				for i, d := range downloads {
					items[i] = listItem{id: fmt.Sprintf("%d bytes", d.Size), title: d.Name, data: d}
				}
				return items, nil
			}, nil)
		}},
	}
}

func newSimpleListView(title string, fetch func(ctx context.Context, filter string) ([]listItem, error), onSelect func(listItem) tea.Cmd) *listModel {
	return newListView(ListConfig{Title: title, Fetch: fetch, OnSelect: onSelect})
}

// --- PR Section ---

func newPRListView(client *bitbucket.Client, ws, repo string) *listModel {
	return newListView(ListConfig{
		Title:   "Pull Requests",
		Filters: []string{"OPEN", "MERGED", "DECLINED", "SUPERSEDED"},
		Fetch: func(ctx context.Context, filter string) ([]listItem, error) {
			prs, err := client.PRs(ws, repo).List(ctx, filter)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(prs))
			for i, pr := range prs {
				items[i] = listItem{id: fmt.Sprintf("#%d", pr.ID), title: pr.Title, subtitle: pr.Author.DisplayName, data: pr}
			}
			return items, nil
		},
		OnSelect: func(item listItem) tea.Cmd {
			pr := item.data.(bitbucket.PR)
			return pushViewCmd(newPRDetailView(client, ws, repo, pr))
		},
	})
}

func newPRDetailView(client *bitbucket.Client, ws, repo string, pr bitbucket.PR) *detailModel {
	approveKey := key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "approve"))
	mergeKey := key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "merge"))
	commentsKey := key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "comments"))
	diffKey := key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "diff"))

	return newDetailView(DetailConfig{
		Title:   fmt.Sprintf("#%d", pr.ID),
		Content: render.PRDetailString(pr),
		Actions: []ActionItem{
			{Label: "Comments", Key: &commentsKey, OnSelect: func() tea.Cmd {
				return pushViewCmd(newPRCommentListView(client, ws, repo, pr.ID))
			}},
			{Label: "Activity", OnSelect: func() tea.Cmd {
				return fetchAndPushText(fmt.Sprintf("Activity (#%d)", pr.ID), func() (string, error) {
					activities, err := client.PRs(ws, repo).Activity(context.Background(), pr.ID)
					if err != nil {
						return "", err
					}
					return render.PRActivityString(activities), nil
				})
			}},
			{Label: "Statuses", OnSelect: func() tea.Cmd {
				return fetchAndPushText(fmt.Sprintf("Statuses (#%d)", pr.ID), func() (string, error) {
					statuses, err := client.PRs(ws, repo).Statuses(context.Background(), pr.ID)
					if err != nil {
						return "", err
					}
					return render.PRStatusesString(statuses), nil
				})
			}},
			{Label: "Diff", Key: &diffKey, OnSelect: func() tea.Cmd {
				return fetchAndPushText(fmt.Sprintf("Diff (#%d)", pr.ID), func() (string, error) {
					return client.PRs(ws, repo).Diff(context.Background(), pr.ID)
				})
			}},
			{Label: "Tasks", OnSelect: func() tea.Cmd {
				return pushViewCmd(newPRTaskListView(client, ws, repo, pr.ID))
			}},
			{Label: "Approve", Key: &approveKey, OnSelect: func() tea.Cmd {
				return executeAction(func() error {
					return client.PRs(ws, repo).Approve(context.Background(), pr.ID)
				}, fmt.Sprintf("PR #%d approved", pr.ID))
			}},
			{Label: "Merge", Key: &mergeKey, Confirm: &ConfirmConfig{
				Message: fmt.Sprintf("Merge PR #%d into %s?", pr.ID, pr.Destination.Branch.Name),
				OnYes: func() tea.Cmd {
					return tea.Sequence(
						popView,
						executeAction(func() error {
							return client.PRs(ws, repo).Merge(context.Background(), pr.ID, "merge_commit")
						}, fmt.Sprintf("PR #%d merged", pr.ID)),
					)
				},
			}},
			{Label: "Decline", Confirm: &ConfirmConfig{
				Message: fmt.Sprintf("Decline PR #%d?", pr.ID),
				OnYes: func() tea.Cmd {
					return tea.Sequence(
						popView,
						executeAction(func() error {
							return client.PRs(ws, repo).Decline(context.Background(), pr.ID)
						}, fmt.Sprintf("PR #%d declined", pr.ID)),
					)
				},
			}},
		},
	})
}

func newPRCommentListView(client *bitbucket.Client, ws, repo string, prID int) *listModel {
	return newListView(ListConfig{
		Title: "Comments",
		Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
			comments, err := client.Comments(ws, repo, prID).List(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(comments))
			for i, c := range comments {
				text := c.Content.Raw
				if len(text) > 60 {
					text = text[:60] + "..."
				}
				items[i] = listItem{id: fmt.Sprintf("[%d]", c.ID), title: c.User.DisplayName + ": " + text, data: c}
			}
			return items, nil
		},
		OnSelect: func(item listItem) tea.Cmd {
			c := item.data.(bitbucket.Comment)
			return pushViewCmd(newTextView(fmt.Sprintf("Comment [%d]", c.ID), render.CommentDetailString(c)))
		},
	})
}

func newPRTaskListView(client *bitbucket.Client, ws, repo string, prID int) *listModel {
	return newListView(ListConfig{
		Title: "Tasks",
		Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
			tasks, err := client.Tasks(ws, repo, prID).List(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(tasks))
			for i, t := range tasks {
				items[i] = listItem{id: fmt.Sprintf("[%d]", t.ID), title: t.Description, subtitle: t.State, data: t}
			}
			return items, nil
		},
	})
}

// --- Pipeline Section ---

func newPipelineListView(client *bitbucket.Client, ws, repo string) *listModel {
	return newListView(ListConfig{
		Title: "Pipelines",
		Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
			pipelines, err := client.Pipelines(ws, repo).List(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(pipelines))
			for i, p := range pipelines {
				state := p.State.Name
				if p.State.Result != nil {
					state += "/" + p.State.Result.Name
				}
				items[i] = listItem{id: fmt.Sprintf("#%d", p.BuildNumber), title: state, subtitle: p.Target.RefName, data: p}
			}
			return items, nil
		},
		OnSelect: func(item listItem) tea.Cmd {
			p := item.data.(bitbucket.Pipeline)
			return pushViewCmd(newPipelineDetailView(client, ws, repo, p))
		},
	})
}

func newPipelineDetailView(client *bitbucket.Client, ws, repo string, p bitbucket.Pipeline) *detailModel {
	return newDetailView(DetailConfig{
		Title:   fmt.Sprintf("#%d", p.BuildNumber),
		Content: render.PipelineDetailString(p),
		Actions: []ActionItem{
			{Label: "Steps", OnSelect: func() tea.Cmd {
				return pushViewCmd(newSimpleListView("Steps", func(ctx context.Context, _ string) ([]listItem, error) {
					steps, err := client.Pipelines(ws, repo).Steps(ctx, p.UUID)
					if err != nil {
						return nil, err
					}
					items := make([]listItem, len(steps))
					for i, s := range steps {
						state := s.State.Name
						if s.State.Result != nil {
							state += "/" + s.State.Result.Name
						}
						items[i] = listItem{id: s.UUID, title: s.Name, subtitle: state, data: s}
					}
					return items, nil
				}, func(item listItem) tea.Cmd {
					step := item.data.(bitbucket.PipelineStep)
					return fetchAndPushText("Log: "+step.Name, func() (string, error) {
						return client.Pipelines(ws, repo).Log(context.Background(), p.UUID, step.UUID)
					})
				}))
			}},
		},
	})
}

// --- Commit Section ---

func newCommitListView(client *bitbucket.Client, ws, repo string) *listModel {
	return newListView(ListConfig{
		Title: "Commits",
		Fetch: func(ctx context.Context, filter string) ([]listItem, error) {
			branch := filter
			if branch == "" {
				branch = "main"
			}
			commits, err := client.Commits(ws, repo).List(ctx, branch)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(commits))
			for i, c := range commits {
				hash := c.Hash
				if len(hash) > 8 {
					hash = hash[:8]
				}
				msg := c.Message
				if len(msg) > 60 {
					msg = msg[:60] + "..."
				}
				items[i] = listItem{id: hash, title: msg, subtitle: c.Author.Raw, data: c}
			}
			return items, nil
		},
		OnSelect: func(item listItem) tea.Cmd {
			c := item.data.(bitbucket.Commit)
			hash := c.Hash
			if len(hash) > 8 {
				hash = hash[:8]
			}
			return pushViewCmd(newTextView("Commit: "+hash, render.CommitDetailString(c)))
		},
	})
}

// --- Issue Section ---

func newIssueListView(client *bitbucket.Client, ws, repo string) *listModel {
	return newListView(ListConfig{
		Title: "Issues",
		Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
			issues, err := client.Issues(ws, repo).List(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(issues))
			for i, issue := range issues {
				items[i] = listItem{id: fmt.Sprintf("#%d", issue.ID), title: issue.Title, subtitle: issue.State + " / " + issue.Kind, data: issue}
			}
			return items, nil
		},
		OnSelect: func(item listItem) tea.Cmd {
			issue := item.data.(bitbucket.Issue)
			return pushViewCmd(newTextView(fmt.Sprintf("Issue #%d", issue.ID), render.IssueDetailString(issue)))
		},
	})
}

// --- Helpers ---

func fetchAndPushText(title string, fetch func() (string, error)) tea.Cmd {
	return func() tea.Msg {
		content, err := fetch()
		if err != nil {
			return actionResultMsg{success: false, message: err.Error()}
		}
		return pushViewMsg{view: newTextView(title, content)}
	}
}

func executeAction(action func() error, successMsg string) tea.Cmd {
	return func() tea.Msg {
		if err := action(); err != nil {
			return actionResultMsg{success: false, message: err.Error()}
		}
		return actionResultMsg{success: true, message: successMsg}
	}
}

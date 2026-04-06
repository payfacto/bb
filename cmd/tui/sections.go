package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/internal/history"
	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	repoFavMarker = "★ "
	repoMRUMarker = "↺ "
	projNoMarker  = "  " // aligns plain items with fav/MRU markers

	shortHashLen  = 8  // characters to show for a commit/tag hash abbreviation
	isoDateLen    = 10 // characters to keep from an ISO-8601 timestamp (YYYY-MM-DD)
	msgPreviewLen = 60 // max characters shown for a commit message preview

	// Deploy key display: show a short prefix and suffix, mask the middle.
	deployKeyPrefixLen = 6
	deployKeySuffixLen = 4
	deployKeyMaskLen   = 12
	deployKeyMinLen    = deployKeyPrefixLen + deployKeySuffixLen // minimum length before masking
)

// abbrevHash returns the first shortHashLen characters of h, or h unchanged if shorter.
func abbrevHash(h string) string {
	if len(h) > shortHashLen {
		return h[:shortHashLen]
	}
	return h
}

func buildMenuItems(client *bitbucket.Client, cfg *config.Config, hist *history.History, cache *listCache) []menuItem {
	ws := cfg.Workspace
	repo := cfg.Repo
	ps := cfg.PageSize
	if ps <= 0 {
		ps = defaultPageSize
	}
	needRepo := func(make func() View) func() View {
		return guardRepo(repo, make)
	}

	favKey := key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "toggle favourite"))
	cacheKey := "repos:" + ws
	histPath := history.HistoryPath(config.DefaultPath())

	items := []menuItem{
		{label: "Projects", description: "Workspace projects", onSelect: func() View {
			projFavKey := key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "toggle favourite"))
			projCacheKey := "projects:" + ws
			return newListView(ListConfig{
				Title:     "Projects",
				PageSize:  ps,
				Shortcuts: []key.Binding{projFavKey},
				Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
					// 1. In-memory hit — instant.
					if cached, ok := cache.Get(projCacheKey); ok {
						return cached, nil
					}
					// 2. Disk hit — no API call needed.
					if diskProjects, ok := hist.Projects(ws); ok {
						items := projectItemsFromCache(diskProjects, hist, ws)
						cache.Pin(projCacheKey, items)
						return items, nil
					}
					// 3. First run or explicit refresh — fetch from API and persist.
					projects, err := client.Projects(ws).List(ctx)
					if err != nil {
						return nil, err
					}
					hist.SetProjects(ws, toCachedProjects(projects))
					// Save is best-effort; a failure here means the cache won't
					// persist but projects are still displayed correctly.
					_ = hist.Save(histPath)
					items := projectListItems(projects, hist, ws)
					cache.Pin(projCacheKey, items)
					return items, nil
				},
				OnRefresh: func() tea.Cmd {
					cache.Invalidate(projCacheKey)
					hist.ClearProjects(ws)
					return saveHistoryCmd(hist, histPath)
				},
				OnKey: func(msg tea.KeyMsg, selected listItem, items []listItem) ([]listItem, tea.Cmd) {
					if !key.Matches(msg, projFavKey) {
						return nil, nil
					}
					p := selected.data.(bitbucket.Project)
					hist.ToggleProjectFavourite(ws, p.Key)
					cache.Invalidate(projCacheKey)
					updated := sortProjectItems(items, hist, ws)
					return updated, saveHistoryCmd(hist, histPath)
				},
				OnSelect: func(item listItem) tea.Cmd {
					p := item.data.(bitbucket.Project)
					hist.AddProjectMRU(ws, p.Key, p.Name)
					cache.Invalidate(projCacheKey)
					navCmd := pushViewCmd(newDetailView(DetailConfig{
						Title: "Project: " + p.Key,
						ContentFetch: func() string {
							return render.ProjectDetailString(p)
						},
						Actions: []ActionItem{
							{Label: "Open in browser", OnSelect: func() tea.Cmd {
								return openURLCmd(p.Links.HTML.Href)
							}},
							{Label: "Repos in this project", OnSelect: func() tea.Cmd {
								return pushViewCmd(newListView(ListConfig{
									Title:     "Repos: " + p.Key,
									PageSize:  ps,
									Shortcuts: []key.Binding{favKey},
									Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
										repos, err := client.Repos(ws).ListByProject(ctx, p.Key)
										if err != nil {
											return nil, err
										}
										return repoListItems(repos, hist, ws), nil
									},
									OnKey: func(msg tea.KeyMsg, selected listItem, items []listItem) ([]listItem, tea.Cmd) {
										if !key.Matches(msg, favKey) {
											return nil, nil
										}
										r := selected.data.(bitbucket.Repo)
										hist.ToggleFavourite(ws, r.Slug)
										cache.Invalidate(cacheKey)
										updated := sortRepoItems(items, hist, ws)
										return updated, saveHistoryCmd(hist, histPath)
									},
									OnSelect: func(item listItem) tea.Cmd {
										r := item.data.(bitbucket.Repo)
										hist.AddMRU(ws, r.Slug, r.Name)
										cache.Invalidate(cacheKey)
										repoCfg := *cfg
										repoCfg.Repo = r.Slug
										navCmd := pushViewCmd(newMenuModel(ws, r.Slug, buildMenuItems(client, &repoCfg, hist, cache)))
										return tea.Batch(navCmd, saveHistoryCmd(hist, histPath))
									},
								}))
							}},
						},
					}))
					return tea.Batch(navCmd, saveHistoryCmd(hist, histPath))
				},
			})
		}},
		{label: "Repositories", description: "List workspace repos", onSelect: func() View {
			return newListView(ListConfig{
				Title:     "Repositories",
				PageSize:  ps,
				Shortcuts: []key.Binding{favKey},
				Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
					// 1. In-memory hit — instant.
					if cached, ok := cache.Get(cacheKey); ok {
						return cached, nil
					}
					// 2. Disk hit — no API call needed.
					if diskRepos, ok := hist.Repos(ws); ok {
						items := repoItemsFromCache(diskRepos, hist, ws)
						cache.Pin(cacheKey, items)
						return items, nil
					}
					// 3. First run or explicit refresh — fetch from API and persist.
					repos, err := client.Repos(ws).List(ctx)
					if err != nil {
						return nil, err
					}
					hist.SetRepos(ws, toCachedRepos(repos))
					// Save is best-effort; a failure here means the cache won't
					// persist but the repos are still displayed correctly.
					_ = hist.Save(histPath)
					items := repoListItems(repos, hist, ws)
					cache.Pin(cacheKey, items)
					return items, nil
				},
				OnRefresh: func() tea.Cmd {
					cache.Invalidate(cacheKey)
					hist.ClearRepos(ws)
					return saveHistoryCmd(hist, histPath)
				},
				OnKey: func(msg tea.KeyMsg, selected listItem, items []listItem) ([]listItem, tea.Cmd) {
					if !key.Matches(msg, favKey) {
						return nil, nil
					}
					r := selected.data.(bitbucket.Repo)
					hist.ToggleFavourite(ws, r.Slug)
					cache.Invalidate(cacheKey)
					updated := sortRepoItems(items, hist, ws)
					return updated, saveHistoryCmd(hist, histPath)
				},
				OnSelect: func(item listItem) tea.Cmd {
					r := item.data.(bitbucket.Repo)
					hist.AddMRU(ws, r.Slug, r.Name)
					cache.Invalidate(cacheKey)
					repoCfg := *cfg
					repoCfg.Repo = r.Slug
					navCmd := pushViewCmd(newMenuModel(ws, r.Slug, buildMenuItems(client, &repoCfg, hist, cache)))
					return tea.Batch(navCmd, saveHistoryCmd(hist, histPath))
				},
			})
		}},
		{label: "Pull Requests", description: "List, review, approve, merge PRs", onSelect: needRepo(func() View {
			return newPRListView(client, ws, repo, ps)
		})},
		{label: "Pipelines", description: "View builds, trigger, check steps", onSelect: needRepo(func() View {
			return newPipelineListView(client, ws, repo, ps)
		})},
		{label: "Branches", description: "List and manage branches", onSelect: needRepo(func() View {
			newBranchKey := key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new branch"))
			return newListView(ListConfig{
				Title:     "Branches",
				PageSize:  ps,
				Shortcuts: []key.Binding{newBranchKey},
				Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
					branches, err := client.Branches(ws, repo).List(ctx)
					if err != nil {
						return nil, err
					}
					items := make([]listItem, len(branches))
					for i, b := range branches {
						items[i] = listItem{id: abbrevHash(b.Target.Hash), title: b.Name, data: b}
					}
					return items, nil
				},
				OnKey: func(msg tea.KeyMsg, _ listItem, items []listItem) ([]listItem, tea.Cmd) {
					if !key.Matches(msg, newBranchKey) {
						return nil, nil
					}
					return nil, pushViewCmd(newInputView("New Branch", "branch-name", func(name string) tea.Cmd {
						return executeAction(func() error {
							_, err := client.Branches(ws, repo).Create(context.Background(), name, "HEAD")
							return err
						}, fmt.Sprintf("Branch %q created", name))
					}))
				},
				OnSelect: func(item listItem) tea.Cmd {
					b := item.data.(bitbucket.Branch)
					return pushViewCmd(newBranchDetailView(client, ws, repo, b, ps))
				},
			})
		})},
		{label: "Commits", description: "Browse commit history", onSelect: needRepo(func() View {
			return newCommitListView(client, ws, repo, ps)
		})},
		{label: "Tags", description: "List and manage tags", onSelect: needRepo(func() View {
			newTagKey := key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new tag"))
			return newListView(ListConfig{
				Title:     "Tags",
				PageSize:  ps,
				Shortcuts: []key.Binding{newTagKey},
				Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
					tags, err := client.Tags(ws, repo).List(ctx)
					if err != nil {
						return nil, err
					}
					items := make([]listItem, len(tags))
					for i, t := range tags {
						items[i] = listItem{id: abbrevHash(t.Target.Hash), title: t.Name, data: t}
					}
					return items, nil
				},
				OnKey: func(msg tea.KeyMsg, _ listItem, items []listItem) ([]listItem, tea.Cmd) {
					if !key.Matches(msg, newTagKey) {
						return nil, nil
					}
					return nil, pushViewCmd(newInputView("New Tag — Name", "tag-name", func(name string) tea.Cmd {
						return pushViewCmd(newInputView("New Tag — Target (commit hash or branch)", "HEAD", func(target string) tea.Cmd {
							if target == "" {
								target = "HEAD"
							}
							return executeAction(func() error {
								_, err := client.Tags(ws, repo).Create(context.Background(), name, target)
								return err
							}, fmt.Sprintf("Tag %q created", name))
						}))
					}))
				},
				OnSelect: func(item listItem) tea.Cmd {
					t := item.data.(bitbucket.Tag)
					return pushViewCmd(newDetailView(DetailConfig{
						Title:   "Tag: " + t.Name,
						Content: render.TagListString([]bitbucket.Tag{t}),
						Actions: []ActionItem{
							{Label: "Open in browser", OnSelect: func() tea.Cmd {
								return openURLCmd(t.Links.HTML.Href)
							}},
						},
					}))
				},
			})
		})},
		{label: "Issues", description: "Track and manage issues", onSelect: needRepo(func() View {
			return newIssueListView(client, ws, repo, ps)
		})},
		{label: "Deployments", description: "View deployments", onSelect: needRepo(func() View {
			return newSimpleListView("Deployments", ps, func(ctx context.Context, _ string) ([]listItem, error) {
				deps, err := client.Deployments(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				// build environment UUID → name lookup (best-effort, ignore error)
				envNames := map[string]string{}
				if envs, err := client.Environments(ws, repo).List(ctx); err == nil {
					for _, e := range envs {
						envNames[e.UUID] = e.Name
					}
				}
				items := make([]listItem, len(deps))
				for i, d := range deps {
					status := d.State.Name
					if d.State.Status != nil && d.State.Status.Name != "" {
						status = d.State.Status.Name
					}
					date := d.LastUpdateTime
					if len(date) > isoDateLen {
						date = date[:isoDateLen]
					}
					title := render.StateBadge(status)
					if date != "" {
						title += "  " + date
					}
					if name := envNames[d.Environment.UUID]; name != "" {
						title += "  " + name
					}
					if d.Deployable.Commit != nil {
						title += "  " + abbrevHash(d.Deployable.Commit.Hash)
					}
					items[i] = listItem{title: title, data: d}
				}
				return items, nil
			}, func(item listItem) tea.Cmd {
				d := item.data.(bitbucket.Deployment)
				state := d.State.Name
				if d.State.Status != nil {
					state += "/" + d.State.Status.Name
				}
				// build actions for commit and pipeline navigation
				var actions []ActionItem
				if d.Deployable.Commit != nil {
					commitHash := d.Deployable.Commit.Hash
					actions = append(actions, ActionItem{
						Label: "Commit: " + abbrevHash(commitHash),
						OnSelect: func() tea.Cmd {
							return fetchAndPushCommitDetail(client, ws, repo, commitHash)
						},
					})
				}
				if d.Deployable.Pipeline != nil {
					pipelineUUID := d.Deployable.Pipeline.UUID
					actions = append(actions, ActionItem{
						Label: "Pipeline: " + pipelineUUID,
						OnSelect: func() tea.Cmd {
							return pushViewCmd(newLoadingTextView("Pipeline", func() (string, error) {
								p, err := client.Pipelines(ws, repo).Get(context.Background(), pipelineUUID)
								if err != nil {
									return "", err
								}
								return render.PipelineDetailString(p), nil
							}))
						},
					})
				}
				return pushViewCmd(newDetailView(DetailConfig{
					Title: "Deployment",
					ContentFetch: func() string {
						envName := d.Environment.UUID
						if envs, err := client.Environments(ws, repo).List(context.Background()); err == nil {
							for _, e := range envs {
								if e.UUID == d.Environment.UUID {
									envName = e.Name
									if e.EnvironmentType.Name != "" {
										envName += " (" + e.EnvironmentType.Name + ")"
									}
									break
								}
							}
						}
						var sb strings.Builder
						sb.WriteString(fmt.Sprintf("State:       %s\n", state))
						sb.WriteString(fmt.Sprintf("Environment: %s\n", envName))
						if d.Deployable.Commit != nil {
							sb.WriteString(fmt.Sprintf("Commit:      %s\n", d.Deployable.Commit.Hash))
						}
						if d.Deployable.Pipeline != nil {
							sb.WriteString(fmt.Sprintf("Pipeline:    %s\n", d.Deployable.Pipeline.UUID))
						}
						sb.WriteString(fmt.Sprintf("Updated:     %s\n", d.LastUpdateTime))
						return sb.String()
					},
					Actions: actions,
				}))
			})
		})},
		{label: "Members", description: "Workspace members", onSelect: func() View {
			return newSimpleListView("Members", ps, func(ctx context.Context, _ string) ([]listItem, error) {
				members, err := client.Members(ws).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(members))
				for i, m := range members {
					title := m.User.DisplayName
					if m.User.Nickname != "" && m.User.Nickname != m.User.DisplayName {
						title += "  (" + m.User.Nickname + ")"
					}
					items[i] = listItem{title: title, data: m}
				}
				return items, nil
			}, func(item listItem) tea.Cmd {
				m := item.data.(bitbucket.WorkspaceMember)
				nickname := m.User.Nickname
				displayName := m.User.DisplayName

				var actions []ActionItem
				if ws != "" && repo != "" {
					actions = append(actions, ActionItem{
						Label: "Their Pull Requests",
						OnSelect: func() tea.Cmd {
							return pushViewCmd(newListView(ListConfig{
								Title:    displayName + "'s PRs",
								PageSize: ps,
								Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
									prs, err := client.PRs(ws, repo).ListByAuthor(ctx, nickname)
									if err != nil {
										return nil, err
									}
									items := make([]listItem, len(prs))
									for i, pr := range prs {
										items[i] = listItem{
											id:    fmt.Sprintf("#%d", pr.ID),
											title: pr.Title,
											data:  pr,
										}
									}
									return items, nil
								},
								OnSelect: func(item listItem) tea.Cmd {
									pr := item.data.(bitbucket.PR)
									return pushViewCmd(newLoadingTextView(fmt.Sprintf("PR #%d", pr.ID), func() (string, error) {
										full, err := client.PRs(ws, repo).Get(context.Background(), pr.ID)
										if err != nil {
											return "", err
										}
										return render.PRDetailString(full), nil
									}))
								},
							}))
						},
					})
					actions = append(actions, ActionItem{
						Label: "Their Commits",
						OnSelect: func() tea.Cmd {
							return pushViewCmd(newListView(ListConfig{
								Title:    displayName + "'s Commits",
								PageSize: ps,
								Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
									commits, err := client.Commits(ws, repo).List(ctx, "")
									if err != nil {
										return nil, err
									}
									var items []listItem
									for _, c := range commits {
										// match by display name (Actor only has DisplayName)
										if c.Author.User == nil || c.Author.User.DisplayName != displayName {
											continue
										}
										msg := c.Message
										if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
											msg = msg[:idx]
										}
										if len(msg) > msgPreviewLen {
											msg = msg[:msgPreviewLen] + "..."
										}
										date := c.Date
										if len(date) > isoDateLen {
											date = date[:isoDateLen]
										}
										items = append(items, listItem{id: abbrevHash(c.Hash), title: msg, subtitle: date, data: c})
									}
									return items, nil
								},
								OnSelect: func(item listItem) tea.Cmd {
									c := item.data.(bitbucket.Commit)
									return fetchAndPushCommitDetail(client, ws, repo, c.Hash)
								},
							}))
						},
					})
				}

				content := fmt.Sprintf("Display Name: %s\nNickname:     %s\nAccount ID:   %s\n",
					displayName, nickname, m.User.AccountID)
				return pushViewCmd(newDetailView(DetailConfig{
					Title:   displayName,
					Content: content,
					Actions: actions,
				}))
			})
		}},
		{label: "Snippets", description: "Workspace snippets", onSelect: func() View {
			return newSimpleListView("Snippets", ps, func(ctx context.Context, _ string) ([]listItem, error) {
				snippets, err := client.Snippets(ws).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(snippets))
				for i, s := range snippets {
					visibility := "public"
					if s.IsPrivate {
						visibility = "private"
					}
					title := fmt.Sprintf("[%s] %s", visibility, s.Title)
					items[i] = listItem{id: s.ID, title: title, data: s}
				}
				return items, nil
			}, func(item listItem) tea.Cmd {
				s := item.data.(bitbucket.Snippet)
				return pushViewCmd(newDetailView(DetailConfig{
					Title: "Snippet: " + s.ID,
					ContentFetch: func() string {
						return render.SnippetDetailString(s)
					},
					Actions: []ActionItem{
						{Label: "Open in browser", OnSelect: func() tea.Cmd {
							return openURLCmd(s.Links.HTML.Href)
						}},
					},
				}))
			})
		}},
		{label: "Settings", description: "Webhooks, deploy keys, environments, restrictions", onSelect: needRepo(func() View {
			return newMenuModel(ws, repo, buildSettingsItems(client, ws, repo, ps))
		})},
		{label: "bb Setup", description: "Reconfigure workspace, repo, credentials", onSelect: func() View {
			return newSetupView(config.DefaultPath(), cfg)
		}},
	}

	if repo != "" {
		repoInfo := menuItem{
			label:       "Repo Info",
			description: "Details, clone URLs, open in browser",
			onSelect: func() View {
				return newDetailView(DetailConfig{
					Title: repo,
					ContentFetch: func() string {
						r, err := client.Repos(ws).Get(context.Background(), repo)
						if err != nil {
							return "Error: " + err.Error()
						}
						var sb strings.Builder
						sb.WriteString(fmt.Sprintf("Slug:        %s\n", r.Slug))
						sb.WriteString(fmt.Sprintf("Full Name:   %s\n", r.FullName))
						if r.Description != "" {
							sb.WriteString(fmt.Sprintf("Description: %s\n", r.Description))
						}
						visibility := "private"
						if !r.IsPrivate {
							visibility = "public"
						}
						sb.WriteString(fmt.Sprintf("Visibility:  %s\n", visibility))
						if ssh := cloneURL(r, "ssh"); ssh != "" {
							sb.WriteString(fmt.Sprintf("Clone SSH:   %s\n", ssh))
						}
						if https := cloneURL(r, "https"); https != "" {
							sb.WriteString(fmt.Sprintf("Clone HTTPS: %s\n", https))
						}
						return sb.String()
					},
					Actions: []ActionItem{
						{Label: "Open in browser", OnSelect: func() tea.Cmd {
							return func() tea.Msg {
								r, err := client.Repos(ws).Get(context.Background(), repo)
								if err != nil {
									return actionResultMsg{success: false, message: "get repo: " + err.Error()}
								}
								return openURLCmd(r.Links.HTML.Href)()
							}
						}},
						{Label: "Update Description", OnSelect: func() tea.Cmd {
							return pushViewCmd(newInputView("New Description", "repo description", func(desc string) tea.Cmd {
								return executeAction(func() error {
									_, err := client.Repos(ws).Update(context.Background(), repo, bitbucket.UpdateRepoInput{Description: desc})
									return err
								}, "Description updated")
							}))
						}},
						{Label: "✗ Delete Repo", Style: &actionDangerStyle, Confirm: &ConfirmConfig{
							Message: fmt.Sprintf("Permanently delete %q? This cannot be undone.", repo),
							OnYes: func() tea.Cmd {
								return tea.Sequence(
									popView,
									popView,
									popView,
									executeAction(func() error {
										return client.Repos(ws).Delete(context.Background(), repo)
									}, fmt.Sprintf("Repository %q deleted", repo)),
								)
							},
						}},
					},
				})
			},
		}
		items = append([]menuItem{repoInfo}, items...)
	}

	return items
}

func buildSettingsItems(client *bitbucket.Client, ws, repo string, pageSize int) []menuItem {
	return []menuItem{
		{label: "Webhooks", description: "Manage repository webhooks", onSelect: func() View {
			return newSimpleListView("Webhooks", pageSize, func(ctx context.Context, _ string) ([]listItem, error) {
				hooks, err := client.Webhooks(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(hooks))
				for i, h := range hooks {
					activeBadge := render.StateBadge("INACTIVE")
					if h.Active {
						activeBadge = render.StateBadge("ACTIVE")
					}
					items[i] = listItem{title: activeBadge + " " + h.URL, data: h}
				}
				return items, nil
			}, func(item listItem) tea.Cmd {
				h := item.data.(bitbucket.Webhook)
				active := "INACTIVE"
				if h.Active {
					active = "ACTIVE"
				}
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("UUID:        %s\n", h.UUID))
				sb.WriteString(fmt.Sprintf("URL:         %s\n", h.URL))
				sb.WriteString(fmt.Sprintf("Description: %s\n", h.Description))
				sb.WriteString(fmt.Sprintf("Status:      %s\n", render.StateBadge(active)))
				sb.WriteString("Events:\n")
				for _, e := range h.Events {
					sb.WriteString(fmt.Sprintf("             %s\n", e))
				}
				sb.WriteString(fmt.Sprintf("Created:     %s\n", h.CreatedAt))
				return pushViewCmd(newDetailView(DetailConfig{
					Title:   "Webhook: " + h.URL,
					Content: sb.String(),
					Actions: []ActionItem{
						{Label: "Open in browser", OnSelect: func() tea.Cmd {
							return openURLCmd(h.URL)
						}},
						{Label: "✗ Delete", Style: &actionDangerStyle, Confirm: &ConfirmConfig{
							Message: fmt.Sprintf("Delete webhook %q?", h.URL),
							OnYes: func() tea.Cmd {
								return tea.Sequence(
									popView,
									popView,
									executeAction(func() error {
										return client.Webhooks(ws, repo).Delete(context.Background(), h.UUID)
									}, "Webhook deleted"),
								)
							},
						}},
					},
				}))
			})
		}},
		{label: "Deploy Keys", description: "Manage deploy keys", onSelect: func() View {
			return newSimpleListView("Deploy Keys", pageSize, func(ctx context.Context, _ string) ([]listItem, error) {
				keys, err := client.DeployKeys(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(keys))
				for i, k := range keys {
					items[i] = listItem{id: fmt.Sprintf("%d", k.ID), title: k.Label, data: k}
				}
				return items, nil
			}, func(item listItem) tea.Cmd {
				k := item.data.(bitbucket.DeployKey)
				maskedKey := k.Key
				if len(k.Key) > deployKeyMinLen {
					maskedKey = k.Key[:deployKeyPrefixLen] + strings.Repeat("•", deployKeyMaskLen) + k.Key[len(k.Key)-deployKeySuffixLen:]
				}
				return pushViewCmd(newDetailView(DetailConfig{
					Title: "Deploy Key: " + k.Label,
					Content: fmt.Sprintf("ID:      %d\nLabel:   %s\nCreated: %s\n\nKey:\n%s\n",
						k.ID, k.Label, k.CreatedOn, maskedKey),
					Actions: []ActionItem{
						{Label: "Reveal Key", OnSelect: func() tea.Cmd {
							return pushViewCmd(newTextView("Key: "+k.Label, k.Key+"\n"))
						}},
						{Label: "✗ Delete", Style: &actionDangerStyle, Confirm: &ConfirmConfig{
							Message: fmt.Sprintf("Delete deploy key %q?", k.Label),
							OnYes: func() tea.Cmd {
								return tea.Sequence(
									popView,
									popView,
									executeAction(func() error {
										return client.DeployKeys(ws, repo).Delete(context.Background(), k.ID)
									}, fmt.Sprintf("Deploy key %q deleted", k.Label)),
								)
							},
						}},
					},
				}))
			})
		}},
		{label: "Environments", description: "Deployment environments", onSelect: func() View {
			return newSimpleListView("Environments", pageSize, func(ctx context.Context, _ string) ([]listItem, error) {
				envs, err := client.Environments(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(envs))
				for i, e := range envs {
					title := e.Name
					if e.EnvironmentType.Name != "" {
						title += "  " + e.EnvironmentType.Name
					}
					if e.Lock.Name == "LOCKED" {
						title += "  " + actionDangerStyle.Render("[LOCKED]")
					}
					items[i] = listItem{title: title, data: e}
				}
				return items, nil
			}, func(item listItem) tea.Cmd {
				e := item.data.(bitbucket.Environment)
				content := fmt.Sprintf("UUID:  %s\nName:  %s\nType:  %s\nLock:  %s\n",
					e.UUID, e.Name, e.EnvironmentType.Name, render.StateBadge(e.Lock.Name))
				return pushViewCmd(newTextView("Environment: "+e.Name, content))
			})
		}},
		{label: "Restrictions", description: "Branch restrictions", onSelect: func() View {
			return newSimpleListView("Restrictions", pageSize, func(ctx context.Context, _ string) ([]listItem, error) {
				restrictions, err := client.Restrictions(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(restrictions))
				for i, r := range restrictions {
					items[i] = listItem{id: fmt.Sprintf("%d", r.ID), title: r.Kind, subtitle: r.Pattern, data: r}
				}
				return items, nil
			}, func(item listItem) tea.Cmd {
				r := item.data.(bitbucket.BranchRestriction)
				var valueStr string
				if r.Value != nil && *r.Value > 0 {
					switch r.Kind {
					case "require_passing_builds_to_merge":
						valueStr = fmt.Sprintf("%d (passing builds required)", *r.Value)
					case "require_approvals_to_merge":
						valueStr = fmt.Sprintf("%d (approvals required)", *r.Value)
					case "require_default_reviewer_approvals_to_merge":
						valueStr = fmt.Sprintf("%d (default reviewer approvals required)", *r.Value)
					default:
						valueStr = fmt.Sprintf("%d", *r.Value)
					}
				}
				content := fmt.Sprintf("ID:      %d\nKind:    %s\nMatch:   %s\nPattern: %s\n",
					r.ID, r.Kind, r.BranchMatchKind, r.Pattern)
				if valueStr != "" {
					content += fmt.Sprintf("Value:   %s\n", valueStr)
				}
				return pushViewCmd(newDetailView(DetailConfig{
					Title:   fmt.Sprintf("Restriction #%d", r.ID),
					Content: content,
					Actions: []ActionItem{
						{Label: "✗ Delete", Style: &actionDangerStyle, Confirm: &ConfirmConfig{
							Message: fmt.Sprintf("Delete restriction #%d (%s)?", r.ID, r.Kind),
							OnYes: func() tea.Cmd {
								return tea.Sequence(
									popView,
									popView,
									executeAction(func() error {
										return client.Restrictions(ws, repo).Delete(context.Background(), r.ID)
									}, fmt.Sprintf("Restriction #%d deleted", r.ID)),
								)
							},
						}},
					},
				}))
			})
		}},
		{label: "Downloads", description: "Download artifacts", onSelect: func() View {
			return newSimpleListView("Downloads", pageSize, func(ctx context.Context, _ string) ([]listItem, error) {
				downloads, err := client.Downloads(ws, repo).List(ctx)
				if err != nil {
					return nil, err
				}
				items := make([]listItem, len(downloads))
				for i, d := range downloads {
					items[i] = listItem{id: fmt.Sprintf("%d bytes", d.Size), title: d.Name, data: d}
				}
				return items, nil
			}, func(item listItem) tea.Cmd {
				d := item.data.(bitbucket.Download)
				content := fmt.Sprintf("Name: %s\nSize: %d bytes\n", d.Name, d.Size)
				return pushViewCmd(newTextView("Download: "+d.Name, content))
			})
		}},
	}
}

func newSimpleListView(title string, pageSize int, fetch func(ctx context.Context, filter string) ([]listItem, error), onSelect func(listItem) tea.Cmd) *listModel {
	return newListView(ListConfig{Title: title, PageSize: pageSize, Fetch: fetch, OnSelect: onSelect})
}

// saveHistoryCmd returns a tea.Cmd that persists hist to path and reports any
// error via actionResultMsg so it surfaces in the TUI status bar.
func saveHistoryCmd(hist *history.History, path string) tea.Cmd {
	return func() tea.Msg {
		if err := hist.Save(path); err != nil {
			return actionResultMsg{success: false, message: fmt.Sprintf("save history: %v", err)}
		}
		return nil
	}
}

// openURLCmd opens url in the system default browser.
func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if url == "" {
			return nil
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		default:
			return actionResultMsg{success: false, message: "open URL: unsupported platform"}
		}
		if err := cmd.Start(); err != nil {
			return actionResultMsg{success: false, message: fmt.Sprintf("open URL: %v", err)}
		}
		return nil
	}
}

// repoBaseTitle returns the display title for a repo without any sort marker.
func repoBaseTitle(r bitbucket.Repo) string {
	if r.IsPrivate {
		return r.Name
	}
	return r.Name + "  " + errorStyle.Render("[public]")
}

// cloneURL returns the clone href for the given protocol ("ssh" or "https") from a Repo.
func cloneURL(r bitbucket.Repo, protocol string) string {
	for _, cl := range r.Links.Clone {
		if cl.Name == protocol {
			return cl.Href
		}
	}
	return ""
}

// toCachedRepos converts API repos to the minimal form persisted in history.
func toCachedRepos(repos []bitbucket.Repo) []history.CachedRepo {
	out := make([]history.CachedRepo, len(repos))
	for i, r := range repos {
		out[i] = history.CachedRepo{
			Slug:       r.Slug,
			Name:       r.Name,
			IsPrivate:  r.IsPrivate,
			CloneSSH:   cloneURL(r, "ssh"),
			CloneHTTPS: cloneURL(r, "https"),
		}
	}
	return out
}

// repoItemsFromCache reconstructs listItems from disk-cached repo data.
func repoItemsFromCache(cached []history.CachedRepo, hist *history.History, ws string) []listItem {
	repos := make([]bitbucket.Repo, len(cached))
	for i, c := range cached {
		links := bitbucket.Links{Clone: []bitbucket.CloneLink{
			{Href: c.CloneSSH, Name: "ssh"},
			{Href: c.CloneHTTPS, Name: "https"},
		}}
		repos[i] = bitbucket.Repo{Slug: c.Slug, Name: c.Name, IsPrivate: c.IsPrivate, Links: links}
	}
	return repoListItems(repos, hist, ws)
}

// repoListItems converts API repos into listItems with favourite/MRU markers, sorted.
func repoListItems(repos []bitbucket.Repo, hist *history.History, ws string) []listItem {
	items := make([]listItem, len(repos))
	for i, r := range repos {
		items[i] = listItem{title: repoBaseTitle(r), data: r}
	}
	return sortRepoItems(items, hist, ws)
}

// sortRepoItems re-orders items: favourites (A-Z) → MRU (newest first, non-fav only) → rest (A-Z).
// Favourite items are prefixed with repoFavMarker and MRU-only items with repoMRUMarker.
// The function is idempotent — existing markers are stripped before re-applying.
func sortRepoItems(items []listItem, hist *history.History, ws string) []listItem {
	for i, item := range items {
		items[i].title = repoBaseTitle(item.data.(bitbucket.Repo))
	}

	mruSlugs := hist.RecentSlugs(ws)
	mruRank := make(map[string]int, len(mruSlugs))
	for i, s := range mruSlugs {
		mruRank[s] = i
	}

	var favs, mruOnly, rest []listItem
	for _, item := range items {
		r := item.data.(bitbucket.Repo)
		if hist.IsFavourite(ws, r.Slug) {
			item.title = repoFavMarker + item.title
			favs = append(favs, item)
		} else if _, inMRU := mruRank[r.Slug]; inMRU {
			item.title = repoMRUMarker + item.title
			mruOnly = append(mruOnly, item)
		} else {
			item.title = projNoMarker + item.title
			rest = append(rest, item)
		}
	}

	// Favourites A-Z by repo name.
	sort.Slice(favs, func(i, j int) bool {
		return favs[i].data.(bitbucket.Repo).Name < favs[j].data.(bitbucket.Repo).Name
	})
	// MRU in recency order (lower rank = more recent).
	sort.Slice(mruOnly, func(i, j int) bool {
		ri := mruRank[mruOnly[i].data.(bitbucket.Repo).Slug]
		rj := mruRank[mruOnly[j].data.(bitbucket.Repo).Slug]
		return ri < rj
	})
	// Rest A-Z by repo name.
	sort.Slice(rest, func(i, j int) bool {
		return rest[i].data.(bitbucket.Repo).Name < rest[j].data.(bitbucket.Repo).Name
	})

	result := make([]listItem, 0, len(items))
	result = append(result, favs...)
	result = append(result, mruOnly...)
	result = append(result, rest...)
	return result
}

// --- Project helpers ---

// projectBaseTitle returns the display title for a project without any sort marker.
// The key is shown first so it is always visible without scrolling.
func projectBaseTitle(p bitbucket.Project) string {
	title := headerStyle.Render(p.Key) + "  " + p.Name
	if !p.IsPrivate {
		title += "  " + errorStyle.Render("[public]")
	}
	return title
}

// toCachedProjects converts API projects to the minimal form persisted in history.
func toCachedProjects(projects []bitbucket.Project) []history.CachedProject {
	out := make([]history.CachedProject, len(projects))
	for i, p := range projects {
		out[i] = history.CachedProject{Key: p.Key, Name: p.Name, IsPrivate: p.IsPrivate, URL: p.Links.HTML.Href}
	}
	return out
}

// projectItemsFromCache reconstructs listItems from disk-cached project data.
func projectItemsFromCache(cached []history.CachedProject, hist *history.History, ws string) []listItem {
	projects := make([]bitbucket.Project, len(cached))
	for i, c := range cached {
		var links bitbucket.Links
		links.HTML.Href = c.URL
		projects[i] = bitbucket.Project{Key: c.Key, Name: c.Name, IsPrivate: c.IsPrivate, Links: links}
	}
	return projectListItems(projects, hist, ws)
}

// projectListItems converts API projects into listItems with favourite/MRU markers, sorted.
func projectListItems(projects []bitbucket.Project, hist *history.History, ws string) []listItem {
	items := make([]listItem, len(projects))
	for i, p := range projects {
		items[i] = listItem{title: projectBaseTitle(p), data: p}
	}
	return sortProjectItems(items, hist, ws)
}

// sortProjectItems re-orders items: favourites (A-Z) → MRU (newest first, non-fav only) → rest (A-Z).
// Favourite items are prefixed with repoFavMarker and MRU-only items with repoMRUMarker.
// The function is idempotent — existing markers are stripped before re-applying.
func sortProjectItems(items []listItem, hist *history.History, ws string) []listItem {
	for i, item := range items {
		items[i].title = projectBaseTitle(item.data.(bitbucket.Project))
	}

	mruKeys := hist.RecentProjectKeys(ws)
	mruRank := make(map[string]int, len(mruKeys))
	for i, k := range mruKeys {
		mruRank[k] = i
	}

	var favs, mruOnly, rest []listItem
	for _, item := range items {
		p := item.data.(bitbucket.Project)
		if hist.IsProjectFavourite(ws, p.Key) {
			item.title = repoFavMarker + item.title
			favs = append(favs, item)
		} else if _, inMRU := mruRank[p.Key]; inMRU {
			item.title = repoMRUMarker + item.title
			mruOnly = append(mruOnly, item)
		} else {
			item.title = projNoMarker + item.title
			rest = append(rest, item)
		}
	}

	// Favourites A-Z by project name.
	sort.Slice(favs, func(i, j int) bool {
		return favs[i].data.(bitbucket.Project).Name < favs[j].data.(bitbucket.Project).Name
	})
	// MRU in recency order (lower rank = more recent).
	sort.Slice(mruOnly, func(i, j int) bool {
		ri := mruRank[mruOnly[i].data.(bitbucket.Project).Key]
		rj := mruRank[mruOnly[j].data.(bitbucket.Project).Key]
		return ri < rj
	})
	// Rest A-Z by project name.
	sort.Slice(rest, func(i, j int) bool {
		return rest[i].data.(bitbucket.Project).Name < rest[j].data.(bitbucket.Project).Name
	})

	result := make([]listItem, 0, len(items))
	result = append(result, favs...)
	result = append(result, mruOnly...)
	result = append(result, rest...)
	return result
}

// --- PR Section ---

func newBranchDetailView(client *bitbucket.Client, ws, repo string, b bitbucket.Branch, pageSize int) *detailModel {
	deleteKey := key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete"))
	commitsKey := key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "commits"))

	return newDetailView(DetailConfig{
		Title: "Branch: " + b.Name,
		ContentFetch: func() string {
			return render.BranchDetailString(b)
		},
		Actions: []ActionItem{
			{Label: "Open in browser", OnSelect: func() tea.Cmd {
				return openURLCmd(b.Links.HTML.Href)
			}},
			{Label: "Commits", Key: &commitsKey, OnSelect: func() tea.Cmd {
				return pushViewCmd(newListView(ListConfig{
					Title:    "Commits: " + b.Name,
					PageSize: pageSize,
					Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
						commits, err := client.Commits(ws, repo).List(ctx, b.Name)
						if err != nil {
							return nil, err
						}
						items := make([]listItem, len(commits))
						for i, c := range commits {
							msg := c.Message
							if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
								msg = msg[:idx]
							}
							if len(msg) > msgPreviewLen {
								msg = msg[:msgPreviewLen] + "..."
							}
							items[i] = listItem{id: abbrevHash(c.Hash), title: msg, subtitle: c.Author.Raw, data: c}
						}
						return items, nil
					},
					OnSelect: func(item listItem) tea.Cmd {
						c := item.data.(bitbucket.Commit)
						return pushViewCmd(newCommitDetailView(client, ws, repo, c))
					},
				}))
			}},
			{Label: "✗ Delete", Key: &deleteKey, Style: &actionDangerStyle, Confirm: &ConfirmConfig{
				Message: fmt.Sprintf("Delete branch %q?", b.Name),
				OnYes: func() tea.Cmd {
					return tea.Sequence(
						popView,
						popView,
						executeAction(func() error {
							return client.Branches(ws, repo).Delete(context.Background(), b.Name)
						}, fmt.Sprintf("Branch %q deleted", b.Name)),
					)
				},
			}},
		},
	})
}

func newPRListView(client *bitbucket.Client, ws, repo string, pageSize int) *listModel {
	return newListView(ListConfig{
		Title:    "Pull Requests",
		Filters:  []string{"OPEN", "MERGED", "DECLINED", "SUPERSEDED"},
		PageSize: pageSize,
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
			return pushViewCmd(newPRDetailView(client, ws, repo, pr, pageSize))
		},
	})
}

func newPRDetailView(client *bitbucket.Client, ws, repo string, pr bitbucket.PR, pageSize int) *detailModel {
	commentsKey := key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "comments"))
	diffKey := key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "diff"))

	return newDetailView(DetailConfig{
		Title: fmt.Sprintf("#%d", pr.ID),
		ContentFetch: func() string {
			return render.PRDetailString(pr)
		},
		Actions: []ActionItem{
			{Label: "Open in browser", OnSelect: func() tea.Cmd {
				return openURLCmd(pr.Links.HTML.Href)
			}},
			{Label: "Comments", Key: &commentsKey, OnSelect: func() tea.Cmd {
				return pushViewCmd(newPRCommentListView(client, ws, repo, pr.ID, pageSize))
			}},
			{Label: "Activity", OnSelect: func() tea.Cmd {
				return pushViewCmd(newPRActivityListView(client, ws, repo, pr.ID, pageSize))
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
				return pushViewCmd(newPRTaskListView(client, ws, repo, pr.ID, pageSize))
			}},
			{Label: "Add Reviewer", OnSelect: func() tea.Cmd {
				return pushViewCmd(newInputView("Add Reviewer — Account ID", "account_id", func(accountID string) tea.Cmd {
					return executeAction(func() error {
						return client.PRs(ws, repo).AddReviewer(context.Background(), pr.ID, accountID)
					}, "Reviewer added")
				}))
			}},
			{Label: "✓ Approve", Style: &actionSuccessStyle, OnSelect: func() tea.Cmd {
				return executeAction(func() error {
					return client.PRs(ws, repo).Approve(context.Background(), pr.ID)
				}, fmt.Sprintf("PR #%d approved", pr.ID))
			}},
			{Label: "⤓ Merge", Style: &actionWarnStyle, Confirm: &ConfirmConfig{
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
			{Label: "✗ Decline", Style: &actionDangerStyle, Confirm: &ConfirmConfig{
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

func newPRCommentListView(client *bitbucket.Client, ws, repo string, prID, pageSize int) *listModel {
	return newListView(ListConfig{
		Title:    "Comments",
		PageSize: pageSize,
		Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
			comments, err := client.Comments(ws, repo, prID).List(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(comments))
			for i, c := range comments {
				text := c.Content.Raw
				if idx := strings.IndexByte(text, '\n'); idx >= 0 {
					text = text[:idx]
				}
				if len(text) > msgPreviewLen {
					text = text[:msgPreviewLen] + "..."
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

func newPRTaskListView(client *bitbucket.Client, ws, repo string, prID, pageSize int) *listModel {
	return newListView(ListConfig{
		Title:    "Tasks",
		PageSize: pageSize,
		Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
			tasks, err := client.Tasks(ws, repo, prID).List(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(tasks))
			for i, t := range tasks {
				desc := t.Description
				if idx := strings.IndexByte(desc, '\n'); idx >= 0 {
					desc = desc[:idx]
				}
				items[i] = listItem{id: fmt.Sprintf("[%d]", t.ID), title: render.StateBadge(t.State) + " " + desc, data: t}
			}
			return items, nil
		},
		OnSelect: func(item listItem) tea.Cmd {
			t := item.data.(bitbucket.Task)
			content := fmt.Sprintf("ID:    %d\nState: %s\n\n%s\n", t.ID, t.State, t.Description)
			return pushViewCmd(newTextView(fmt.Sprintf("Task #%d", t.ID), content))
		},
	})
}

func newPRActivityListView(client *bitbucket.Client, ws, repo string, prID, pageSize int) *listModel {
	return newListView(ListConfig{
		Title:    fmt.Sprintf("Activity (#%d)", prID),
		PageSize: pageSize,
		Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
			activities, err := client.PRs(ws, repo).Activity(ctx, prID)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, 0, len(activities))
			for _, a := range activities {
				switch {
				case a.Approval != nil:
					date := a.Approval.Date
					if len(date) > isoDateLen {
						date = date[:isoDateLen]
					}
					tag := actionSuccessStyle.Render("[approval]")
					items = append(items, listItem{
						title:    tag + " " + a.Approval.User.DisplayName + " approved",
						subtitle: date,
						data:     a,
					})
				case a.Update != nil:
					date := a.Update.Date
					if len(date) > isoDateLen {
						date = date[:isoDateLen]
					}
					items = append(items, listItem{
						title:    subtitleStyle.Render("[update]") + " " + a.Update.Author.DisplayName + " → " + a.Update.State,
						subtitle: date,
						data:     a,
					})
				case a.Comment != nil:
					items = append(items, listItem{
						title: helpKeyStyle.Render(fmt.Sprintf("[%d]", a.Comment.ID)) + " " +
							a.Comment.User.DisplayName + ": " + truncateStr(a.Comment.Content.Raw, msgPreviewLen),
						data: a,
					})
				}
			}
			return items, nil
		},
		OnSelect: func(item listItem) tea.Cmd {
			a := item.data.(bitbucket.Activity)
			switch {
			case a.Comment != nil:
				return pushViewCmd(newTextView(
					fmt.Sprintf("Comment [%d]", a.Comment.ID),
					render.CommentDetailString(*a.Comment),
				))
			case a.Approval != nil:
				return pushViewCmd(newTextView("Approval", fmt.Sprintf(
					"User:  %s\nDate:  %s\n", a.Approval.User.DisplayName, a.Approval.Date,
				)))
			case a.Update != nil:
				return pushViewCmd(newTextView("Update", fmt.Sprintf(
					"Author: %s\nState:  %s\nDate:   %s\n",
					a.Update.Author.DisplayName, a.Update.State, a.Update.Date,
				)))
			}
			return nil
		},
	})
}

// truncateStr shortens s to max runes with an ellipsis — mirrors render.truncate locally
// so sections.go does not import the render package's unexported helper.
func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// --- Pipeline Section ---

func newPipelineListView(client *bitbucket.Client, ws, repo string, pageSize int) *listModel {
	triggerKey := key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "trigger pipeline"))
	return newListView(ListConfig{
		Title:     "Pipelines",
		PageSize:  pageSize,
		Shortcuts: []key.Binding{triggerKey},
		Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
			pipelines, err := client.Pipelines(ws, repo).List(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(pipelines))
			for i, p := range pipelines {
				// Show result badge when available (FAILED/SUCCESSFUL/STOPPED),
				// otherwise show the state badge (IN_PROGRESS/PENDING).
				var badge string
				if p.State.Result != nil && p.State.Result.Name != "" {
					badge = render.StateBadge(p.State.Result.Name)
				} else {
					badge = render.StateBadge(p.State.Name)
				}
				items[i] = listItem{id: fmt.Sprintf("#%d", p.BuildNumber), title: badge, subtitle: p.Target.RefName, data: p}
			}
			return items, nil
		},
		OnKey: func(msg tea.KeyMsg, _ listItem, items []listItem) ([]listItem, tea.Cmd) {
			if !key.Matches(msg, triggerKey) {
				return nil, nil
			}
			return nil, pushViewCmd(newInputView("Trigger Pipeline — Branch", "main", func(branch string) tea.Cmd {
				return executeAction(func() error {
					_, err := client.Pipelines(ws, repo).Trigger(context.Background(), branch)
					return err
				}, fmt.Sprintf("Pipeline triggered on %q", branch))
			}))
		},
		OnSelect: func(item listItem) tea.Cmd {
			p := item.data.(bitbucket.Pipeline)
			return pushViewCmd(newPipelineDetailView(client, ws, repo, p, pageSize))
		},
	})
}

func newPipelineDetailView(client *bitbucket.Client, ws, repo string, p bitbucket.Pipeline, pageSize int) *detailModel {
	actions := []ActionItem{
		{Label: "Steps", OnSelect: func() tea.Cmd {
			return pushViewCmd(newSimpleListView("Steps", pageSize, func(ctx context.Context, _ string) ([]listItem, error) {
					steps, err := client.Pipelines(ws, repo).Steps(ctx, p.UUID)
					if err != nil {
						return nil, err
					}
					items := make([]listItem, len(steps))
					for i, s := range steps {
						var badge string
						if s.State.Result != nil && s.State.Result.Name != "" {
							badge = render.StateBadge(s.State.Result.Name)
						} else {
							badge = render.StateBadge(s.State.Name)
						}
						items[i] = listItem{id: s.UUID, title: s.Name, subtitle: badge, data: s}
					}
					return items, nil
				}, func(item listItem) tea.Cmd {
					step := item.data.(bitbucket.PipelineStep)
					result := ""
					if step.State.Result != nil {
						result = step.State.Result.Name
					}
					if result == "NOT_RUN" || step.State.Name == "PENDING" || step.StartedOn == "" {
						return pushMessage("Log: "+step.Name, "No log available — this step did not run.")
					}
					return fetchAndPushText("Log: "+step.Name, func() (string, error) {
						log, err := client.Pipelines(ws, repo).Log(context.Background(), p.UUID, step.UUID)
						if err != nil {
							if strings.Contains(err.Error(), "404") {
								return "No log available for this step.", nil
							}
							return "", err
						}
						return log, nil
					})
				}))
			}},
	}

	if p.State.Name == "IN_PROGRESS" || p.State.Name == "PENDING" {
		actions = append(actions, ActionItem{
			Label: "✗ Stop",
			Style: &actionDangerStyle,
			Confirm: &ConfirmConfig{
				Message: fmt.Sprintf("Stop pipeline #%d?", p.BuildNumber),
				OnYes: func() tea.Cmd {
					return executeAction(func() error {
						return client.Pipelines(ws, repo).Stop(context.Background(), p.UUID)
					}, fmt.Sprintf("Pipeline #%d stopped", p.BuildNumber))
				},
			},
		})
	}

	return newDetailView(DetailConfig{
		Title:   fmt.Sprintf("#%d", p.BuildNumber),
		Content: render.PipelineDetailString(p),
		Actions: actions,
	})
}

// --- Commit Section ---

func newCommitListView(client *bitbucket.Client, ws, repo string, pageSize int) *listModel {
	return newListView(ListConfig{
		Title:    "Commits",
		PageSize: pageSize,
		Fetch: func(ctx context.Context, filter string) ([]listItem, error) {
			commits, err := client.Commits(ws, repo).List(ctx, filter)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(commits))
			for i, c := range commits {
				msg := c.Message
				if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
					msg = msg[:idx]
				}
				if len(msg) > msgPreviewLen {
					msg = msg[:msgPreviewLen] + "..."
				}
				date := c.Date
				if len(date) >= isoDateLen {
					date = date[:isoDateLen]
				}
				subtitle := date + "  " + c.Author.Raw
				items[i] = listItem{id: abbrevHash(c.Hash), title: msg, subtitle: subtitle, data: c}
			}
			return items, nil
		},
		OnSelect: func(item listItem) tea.Cmd {
			c := item.data.(bitbucket.Commit)
			return pushViewCmd(newCommitDetailView(client, ws, repo, c))
		},
	})
}

// --- Issue Section ---

// fetchAndPushCommitDetail fetches a commit by hash then pushes a full commit detail view with actionable parents.
func fetchAndPushCommitDetail(client *bitbucket.Client, ws, repo, hash string) tea.Cmd {
	short := abbrevHash(hash)
	return func() tea.Msg {
		c, err := client.Commits(ws, repo).Get(context.Background(), hash)
		if err != nil {
			return pushViewCmd(newTextView("Commit: "+short, "Error: "+err.Error()))()
		}
		return pushViewCmd(newCommitDetailView(client, ws, repo, c))()
	}
}

func newCommitDetailView(client *bitbucket.Client, ws, repo string, c bitbucket.Commit) *detailModel {
	hash := abbrevHash(c.Hash)

	actions := make([]ActionItem, 0, len(c.Parents))
	for _, p := range c.Parents {
		p := p // capture
		actions = append(actions, ActionItem{
			Label: "Parent: " + abbrevHash(p.Hash),
			OnSelect: func() tea.Cmd {
				return fetchAndPushCommitDetail(client, ws, repo, p.Hash)
			},
		})
	}

	return newDetailView(DetailConfig{
		Title:        "Commit: " + hash,
		ContentFetch: func() string { return render.CommitDetailString(c) },
		Actions:      actions,
	})
}

func newIssueListView(client *bitbucket.Client, ws, repo string, pageSize int) *listModel {
	newIssueKey := key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new issue"))
	return newListView(ListConfig{
		Title:     "Issues",
		PageSize:  pageSize,
		EmptyMsg:  "No issues found.",
		Shortcuts: []key.Binding{newIssueKey},
		Fetch: func(ctx context.Context, _ string) ([]listItem, error) {
			issues, err := client.Issues(ws, repo).List(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]listItem, len(issues))
			for i, issue := range issues {
				items[i] = listItem{id: fmt.Sprintf("#%d", issue.ID), title: render.StateBadge(issue.State) + " " + issue.Title, subtitle: issue.Kind, data: issue}
			}
			return items, nil
		},
		OnKey: func(msg tea.KeyMsg, _ listItem, items []listItem) ([]listItem, tea.Cmd) {
			if !key.Matches(msg, newIssueKey) {
				return nil, nil
			}
			return nil, pushViewCmd(newInputView("New Issue — Title", "issue title", func(title string) tea.Cmd {
				return pushViewCmd(newInputView("Kind (bug/enhancement/proposal/task)", "bug", func(kind string) tea.Cmd {
					if kind == "" {
						kind = "bug"
					}
					return executeAction(func() error {
						_, err := client.Issues(ws, repo).Create(context.Background(), bitbucket.CreateIssueInput{
							Title: title,
							Kind:  kind,
						})
						return err
					}, fmt.Sprintf("Issue %q created", title))
				}))
			}))
		},
		OnSelect: func(item listItem) tea.Cmd {
			issue := item.data.(bitbucket.Issue)
			actions := []ActionItem{
				{Label: "Open in browser", OnSelect: func() tea.Cmd {
					return openURLCmd(issue.Links.HTML.Href)
				}},
			}
			if issue.State == "new" || issue.State == "open" {
				actions = append(actions, ActionItem{
					Label: "✓ Close",
					Style: &actionWarnStyle,
					Confirm: &ConfirmConfig{
						Message: fmt.Sprintf("Close issue #%d?", issue.ID),
						OnYes: func() tea.Cmd {
							return executeAction(func() error {
								_, err := client.Issues(ws, repo).Update(context.Background(), issue.ID, bitbucket.UpdateIssueInput{Status: "resolved"})
								return err
							}, fmt.Sprintf("Issue #%d closed", issue.ID))
						},
					},
				})
			} else {
				actions = append(actions, ActionItem{
					Label: "↺ Reopen",
					Style: &actionSuccessStyle,
					Confirm: &ConfirmConfig{
						Message: fmt.Sprintf("Reopen issue #%d?", issue.ID),
						OnYes: func() tea.Cmd {
							return executeAction(func() error {
								_, err := client.Issues(ws, repo).Update(context.Background(), issue.ID, bitbucket.UpdateIssueInput{Status: "open"})
								return err
							}, fmt.Sprintf("Issue #%d reopened", issue.ID))
						},
					},
				})
			}
			return pushViewCmd(newDetailView(DetailConfig{
				Title:   fmt.Sprintf("Issue #%d", issue.ID),
				Content: render.IssueDetailString(issue),
				Actions: actions,
			}))
		},
	})
}

// --- Helpers ---

// fetchAndPushText pushes an immediate spinner/loading view that transitions
// to a scrollable text view once the fetch completes.
func fetchAndPushText(title string, fetch func() (string, error)) tea.Cmd {
	return pushViewCmd(newLoadingTextView(title, fetch))
}

func pushMessage(title, body string) tea.Cmd {
	return fetchAndPushText(title, func() (string, error) { return body, nil })
}

func executeAction(action func() error, successMsg string) tea.Cmd {
	return func() tea.Msg {
		if err := action(); err != nil {
			return actionResultMsg{success: false, message: err.Error()}
		}
		return actionResultMsg{success: true, message: successMsg}
	}
}

// guardRepo returns a menu onSelect func that shows a friendly error view when
// no repository is configured, instead of attempting an API call that will fail.
func guardRepo(repo string, make func() View) func() View {
	if repo != "" {
		return make
	}
	return func() View {
		return newTextView("No Repository Selected",
			"This section requires a repository to be configured.\n\n"+
				"  • Open 'Repositories' from the home menu and press Enter\n"+
				"    on a repository to scope the menu to it, or\n\n"+
				"  • Run 'bb setup' to save a default repository.\n")
	}
}

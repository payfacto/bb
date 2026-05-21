package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
}

var repoListSort string

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories in the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		repos, err := client.Repos(ws).List(context.Background(), repoListSort)
		if err != nil {
			return err
		}
		return printOutput(repos, func() { render.RepoList(repos) })
	},
}

var (
	repoCreateName        string
	repoCreateDescription string
	repoCreatePrivate     bool
	repoCreateProject     string
)

var repoCreateCmd = &cobra.Command{
	Use:   "create <slug>",
	Short: "Create a new repository in the workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		slug := args[0]
		var input bitbucket.CreateRepoInput
		if _, err := stdinInputOr(&input, func() bitbucket.CreateRepoInput {
			name := repoCreateName
			if name == "" {
				name = slug
			}
			body := bitbucket.CreateRepoInput{
				Scm:         "git",
				Name:        name,
				Description: repoCreateDescription,
				IsPrivate:   repoCreatePrivate,
			}
			if repoCreateProject != "" {
				body.Project = &bitbucket.ProjectRef{Key: repoCreateProject}
			}
			return body
		}); err != nil {
			return err
		}
		repo, err := client.Repos(ws).Create(context.Background(), slug, input)
		if err != nil {
			return err
		}
		return printOutput(repo, func() { render.RepoDetail(repo) })
	},
}

var (
	repoForkName      string
	repoForkWorkspace string
)

var repoForkCmd = &cobra.Command{
	Use:   "fork <slug>",
	Short: "Fork a repository into the workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		var input bitbucket.ForkRepoInput
		if _, err := stdinInputOr(&input, func() bitbucket.ForkRepoInput {
			body := bitbucket.ForkRepoInput{Name: repoForkName}
			if repoForkWorkspace != "" {
				body.Workspace = &bitbucket.WorkspaceRef{Slug: repoForkWorkspace}
			}
			return body
		}); err != nil {
			return err
		}
		repo, err := client.Repos(ws).Fork(context.Background(), args[0], input)
		if err != nil {
			return err
		}
		return printOutput(repo, func() { render.RepoDetail(repo) })
	},
}

var repoGetCmd = &cobra.Command{
	Use:   "get <slug>",
	Short: "Get repository details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		repo, err := client.Repos(ws).Get(context.Background(), args[0])
		if err != nil {
			return err
		}
		return printOutput(repo, func() { render.RepoDetail(repo) })
	},
}

var (
	repoUpdateDescription string
	repoUpdateBranch      string
)

var repoUpdateCmd = &cobra.Command{
	Use:   "update <slug>",
	Short: "Update repository metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		var input bitbucket.UpdateRepoInput
		if _, err := stdinInputOr(&input, func() bitbucket.UpdateRepoInput {
			body := bitbucket.UpdateRepoInput{}
			if cmd.Flags().Changed("description") {
				body.Description = repoUpdateDescription
			}
			if cmd.Flags().Changed("default-branch") {
				body.Mainbranch = &bitbucket.MainbranchRef{Name: repoUpdateBranch, Type: "branch"}
			}
			return body
		}); err != nil {
			return err
		}
		repo, err := client.Repos(ws).Update(context.Background(), args[0], input)
		if err != nil {
			return err
		}
		return printOutput(repo, func() { render.RepoDetail(repo) })
	},
}

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <slug>",
	Short: "Delete a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		slug := args[0]
		if err := client.Repos(ws).Delete(context.Background(), slug); err != nil {
			return err
		}
		return printOutput(map[string]any{"deleted": true, "slug": slug}, func() {
			fmt.Printf("Repository %q deleted\n", slug)
		})
	},
}

func init() {
	repoListCmd.Flags().StringVar(&repoListSort, "sort", "",
		"sort by Bitbucket field, prefix with - for descending (e.g. -updated_on); empty preserves API default")

	repoCreateCmd.Flags().StringVar(&repoCreateName, "name", "", "display name (defaults to slug)")
	repoCreateCmd.Flags().StringVar(&repoCreateDescription, "description", "", "repository description")
	repoCreateCmd.Flags().BoolVar(&repoCreatePrivate, "private", true, "make the repository private")
	repoCreateCmd.Flags().StringVar(&repoCreateProject, "project", "", "project key to assign to")

	repoForkCmd.Flags().StringVar(&repoForkName, "name", "", "override the repository name for the fork")
	repoForkCmd.Flags().StringVar(&repoForkWorkspace, "workspace", "", "target workspace slug (defaults to current workspace)")

	repoUpdateCmd.Flags().StringVar(&repoUpdateDescription, "description", "", "new repository description")
	repoUpdateCmd.Flags().StringVar(&repoUpdateBranch, "default-branch", "", "new default branch name")

	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoForkCmd)
	repoCmd.AddCommand(repoGetCmd)
	repoCmd.AddCommand(repoUpdateCmd)
	repoCmd.AddCommand(repoDeleteCmd)
	rootCmd.AddCommand(repoCmd)
}

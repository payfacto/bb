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

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories in the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		repos, err := client.Repos(ws).List(context.Background())
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
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		slug := args[0]
		name := repoCreateName
		if name == "" {
			name = slug
		}
		input := bitbucket.CreateRepoInput{
			Scm:         "git",
			Name:        name,
			Description: repoCreateDescription,
			IsPrivate:   repoCreatePrivate,
		}
		if repoCreateProject != "" {
			input.Project = &bitbucket.ProjectRef{Key: repoCreateProject}
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
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		input := bitbucket.ForkRepoInput{
			Name: repoForkName,
		}
		if repoForkWorkspace != "" {
			input.Workspace = &bitbucket.WorkspaceRef{Slug: repoForkWorkspace}
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
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
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
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		input := bitbucket.UpdateRepoInput{}
		if cmd.Flags().Changed("description") {
			input.Description = repoUpdateDescription
		}
		if cmd.Flags().Changed("default-branch") {
			input.Mainbranch = &bitbucket.MainbranchRef{Name: repoUpdateBranch, Type: "branch"}
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
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
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

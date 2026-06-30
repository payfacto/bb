package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search code, repositories, and pull requests",
}

var (
	searchCodeLimit   int
	searchCodeExt     string
	searchCodeLang    string
	searchCodeRepo    string
	searchCodeProject string
)

var searchCodeCmd = &cobra.Command{
	Use:   "code <query>...",
	Short: "Search file contents across the workspace",
	Long: "Search file contents across the workspace using Bitbucket's code search index.\n\n" +
		"Notes and limits (this is not git grep):\n" +
		"  - searches the indexed default branch of each repository only\n" +
		"  - token/word based, not regex\n" +
		"  - large, binary, and generated files may not be indexed\n" +
		"  - requires the workspace to have code search enabled\n\n" +
		"The query is passed to Bitbucket verbatim, so modifiers work inline\n" +
		"(e.g. 'bb search code ext:go parseConfig'). The --ext/--lang/--repo/--project\n" +
		"flags are conveniences folded into the query; comma-separated values are OR-combined.",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		results, err := client.Search(ws).Code(context.Background(), bitbucket.CodeSearchOptions{
			Query:   strings.Join(args, " "),
			Ext:     searchCodeExt,
			Lang:    searchCodeLang,
			Repo:    searchCodeRepo,
			Project: searchCodeProject,
			Limit:   searchCodeLimit,
		})
		if err != nil {
			return err
		}
		return printOutput(results, func() { render.CodeSearchResults(results) })
	},
}

var searchReposLimit int

var searchReposCmd = &cobra.Command{
	Use:   "repos <query>...",
	Short: "Search repositories by name or description",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		repos, err := client.Search(ws).Repos(context.Background(), strings.Join(args, " "), searchReposLimit)
		if err != nil {
			return err
		}
		return printOutput(repos, func() { render.RepoList(repos) })
	},
}

var (
	searchPrsLimit int
	searchPrsState string
)

var searchPrsCmd = &cobra.Command{
	Use:   "prs <query>...",
	Short: "Search pull requests by title or description (current repo)",
	Long: "Search pull requests in the current repository by title or description.\n\n" +
		"Bitbucket has no workspace-wide PR search, so this is scoped to a single\n" +
		"repository: set --repo or a default repo via 'bb setup'. For richer\n" +
		"per-repo filtering (state, branch, dates) use 'bb pr list'.",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		prs, err := client.PRs(ws, repo).List(context.Background(), bitbucket.PRListOptions{
			Query: strings.Join(args, " "),
			State: searchPrsState,
			Limit: searchPrsLimit,
		})
		if err != nil {
			return err
		}
		return printOutput(prs, func() { render.PRList(prs) })
	},
}

func init() {
	searchCodeCmd.Flags().IntVarP(&searchCodeLimit, "limit", "L", 100, "maximum results (0 = all)")
	searchCodeCmd.Flags().StringVar(&searchCodeExt, "ext", "", "filter by file extension (comma-separated, e.g. go,mod)")
	searchCodeCmd.Flags().StringVar(&searchCodeLang, "lang", "", "filter by language (comma-separated, e.g. go,python)")
	searchCodeCmd.Flags().StringVar(&searchCodeRepo, "repo-filter", "", "limit to repository slug(s) (comma-separated)")
	searchCodeCmd.Flags().StringVar(&searchCodeProject, "project", "", "limit to project key(s) (comma-separated)")

	searchReposCmd.Flags().IntVarP(&searchReposLimit, "limit", "L", 100, "maximum results (0 = all)")

	searchPrsCmd.Flags().IntVarP(&searchPrsLimit, "limit", "L", 100, "maximum results (0 = all)")
	searchPrsCmd.Flags().StringVar(&searchPrsState, "state", "", "filter by state: OPEN, MERGED, DECLINED, SUPERSEDED")

	searchCmd.AddCommand(searchCodeCmd, searchReposCmd, searchPrsCmd)
	rootCmd.AddCommand(searchCmd)
}

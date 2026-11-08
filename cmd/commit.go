package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Browse commit history",
}

var (
	commitListBranch string
	commitListSort   string
)

var commitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List commits on a branch, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		commits, err := client.Commits(ws, repo).List(context.Background(), commitListBranch, commitListSort)
		if err != nil {
			return err
		}
		return printOutput(commits, func() { render.CommitList(commits) })
	},
}

var commitGetHash string

var commitGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a single commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		c, err := client.Commits(ws, repo).Get(context.Background(), commitGetHash)
		if err != nil {
			return err
		}
		return printOutput(c, func() { render.CommitDetail(c) })
	},
}

var commitStatusesHash string

var commitStatusesCmd = &cobra.Command{
	Use:   "statuses",
	Short: "List build/CI statuses for a commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		statuses, err := client.Commits(ws, repo).Statuses(context.Background(), commitStatusesHash)
		if err != nil {
			return err
		}
		return printOutput(statuses, func() { render.CommitStatusList(statuses) })
	},
}

var commitReportsHash string

var commitReportsCmd = &cobra.Command{
	Use:   "reports",
	Short: "List Code Insights reports (coverage, scans, tests) for a commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		reports, err := client.Commits(ws, repo).Reports(context.Background(), commitReportsHash)
		if err != nil {
			return err
		}
		return printOutput(reports, func() { render.ReportList(reports) })
	},
}

var (
	commitReportHash string
	commitReportID   string
)

var commitReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Get a single Code Insights report for a commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		report, err := client.Commits(ws, repo).Report(context.Background(), commitReportHash, commitReportID)
		if err != nil {
			return err
		}
		return printOutput(report, func() { render.ReportDetail(report) })
	},
}

var (
	commitAnnotationsHash     string
	commitAnnotationsReportID string
)

var commitAnnotationsCmd = &cobra.Command{
	Use:   "annotations",
	Short: "List the annotations (findings) on a commit's Code Insights report",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		annotations, err := client.Commits(ws, repo).ReportAnnotations(context.Background(), commitAnnotationsHash, commitAnnotationsReportID)
		if err != nil {
			return err
		}
		return printOutput(annotations, func() { render.ReportAnnotationList(annotations) })
	},
}

// reportIDFlagUsage is shared by the commands that address a single report.
const reportIDFlagUsage = "report UUID or external ID, as shown by 'bb commit reports' (required)"

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Read file contents from the repository",
}

var (
	fileGetRef  string
	fileGetPath string
)

var fileGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get raw file contents at a ref (branch, tag, or commit hash)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		content, err := client.Commits(ws, repo).File(context.Background(), fileGetRef, fileGetPath)
		if err != nil {
			return err
		}
		fmt.Print(content)
		return nil
	},
}

func init() {
	commitListCmd.Flags().StringVarP(&commitListBranch, "branch", "b", "", "branch name (required)")
	commitListCmd.Flags().StringVar(&commitListSort, "sort", "",
		"sort by Bitbucket field, prefix with - for descending (e.g. -date); empty preserves API default")
	commitListCmd.MarkFlagRequired("branch")

	commitGetCmd.Flags().StringVarP(&commitGetHash, "hash", "x", "", "commit hash (required)")
	commitGetCmd.MarkFlagRequired("hash")

	commitStatusesCmd.Flags().StringVarP(&commitStatusesHash, "hash", "x", "", "commit hash (required)")
	commitStatusesCmd.MarkFlagRequired("hash")

	commitReportsCmd.Flags().StringVarP(&commitReportsHash, "hash", "x", "", "commit hash (required)")
	commitReportsCmd.MarkFlagRequired("hash")

	commitReportCmd.Flags().StringVarP(&commitReportHash, "hash", "x", "", "commit hash (required)")
	commitReportCmd.Flags().StringVar(&commitReportID, "report-id", "", reportIDFlagUsage)
	commitReportCmd.MarkFlagRequired("hash")
	commitReportCmd.MarkFlagRequired("report-id")

	commitAnnotationsCmd.Flags().StringVarP(&commitAnnotationsHash, "hash", "x", "", "commit hash (required)")
	commitAnnotationsCmd.Flags().StringVar(&commitAnnotationsReportID, "report-id", "", reportIDFlagUsage)
	commitAnnotationsCmd.MarkFlagRequired("hash")
	commitAnnotationsCmd.MarkFlagRequired("report-id")

	fileGetCmd.Flags().StringVar(&fileGetRef, "ref", "", "branch name, tag, or commit hash (required)")
	fileGetCmd.Flags().StringVar(&fileGetPath, "path", "", "file path within the repository (required)")
	fileGetCmd.MarkFlagRequired("ref")
	fileGetCmd.MarkFlagRequired("path")

	commitCmd.AddCommand(commitListCmd, commitGetCmd, commitStatusesCmd,
		commitReportsCmd, commitReportCmd, commitAnnotationsCmd)
	fileCmd.AddCommand(fileGetCmd)
	rootCmd.AddCommand(commitCmd, fileCmd)
}

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var pipelineCacheCmd = &cobra.Command{
	Use:   "pipeline-cache",
	Short: "Inspect and clear repository Pipelines caches",
}

var pipelineCacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cache versions stored for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		caches, err := client.PipelineCaches(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(caches, func() { render.PipelineCacheList(caches) })
	},
}

var pipelineCacheDeleteName string

var pipelineCacheDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete every version of a named cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.PipelineCaches(ws, repo).DeleteByName(context.Background(), pipelineCacheDeleteName); err != nil {
			return err
		}
		return printOutput(map[string]any{"deleted": true, "name": pipelineCacheDeleteName}, func() {
			fmt.Printf("Pipeline cache %q deleted\n", pipelineCacheDeleteName)
		})
	},
}

var pipelineCacheURLUUID string

var pipelineCacheURLCmd = &cobra.Command{
	Use:   "url",
	Short: "Get a short-lived download URL for a cache version",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		uri, err := client.PipelineCaches(ws, repo).ContentURI(context.Background(), pipelineCacheURLUUID)
		if err != nil {
			return err
		}
		return printOutput(uri, func() { fmt.Println(uri.URI) })
	},
}

func init() {
	pipelineCacheDeleteCmd.Flags().StringVar(&pipelineCacheDeleteName, "name", "", "cache name, as defined in bitbucket-pipelines.yml (required)")
	pipelineCacheDeleteCmd.MarkFlagRequired("name")

	pipelineCacheURLCmd.Flags().StringVar(&pipelineCacheURLUUID, "uuid", "", "cache version UUID from 'pipeline-cache list' (required)")
	pipelineCacheURLCmd.MarkFlagRequired("uuid")

	pipelineCacheCmd.AddCommand(pipelineCacheListCmd, pipelineCacheDeleteCmd, pipelineCacheURLCmd)
	rootCmd.AddCommand(pipelineCacheCmd)
}

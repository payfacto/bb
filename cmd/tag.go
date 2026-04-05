package cmd

import (
	"context"
	"fmt"

	"github.com/payfacto/bb/cmd/render"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage repository tags",
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tags in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		tags, err := client.Tags(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(tags, func() { render.TagList(tags) })
	},
}

var (
	tagCreateName string
	tagCreateFrom string
)

var tagCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new tag pointing at a commit hash or branch name",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		tag, err := client.Tags(ws, repo).Create(context.Background(), tagCreateName, tagCreateFrom)
		if err != nil {
			return err
		}
		return printOutput(tag, func() {
			fmt.Printf("Tag '%s' created at %s\n", tag.Name, tag.Target.Hash)
		})
	},
}

var tagDeleteName string

var tagDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a tag",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Tags(ws, repo).Delete(context.Background(), tagDeleteName); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "deleted", "name": tagDeleteName}, func() {
			fmt.Printf("Tag '%s' deleted.\n", tagDeleteName)
		})
	},
}

func init() {
	tagCreateCmd.Flags().StringVarP(&tagCreateName, "name", "n", "", "tag name (required)")
	tagCreateCmd.Flags().StringVar(&tagCreateFrom, "from", "", "commit hash or branch name to tag (required)")
	tagCreateCmd.MarkFlagRequired("name")
	tagCreateCmd.MarkFlagRequired("from")

	tagDeleteCmd.Flags().StringVarP(&tagDeleteName, "name", "n", "", "tag name to delete (required)")
	tagDeleteCmd.MarkFlagRequired("name")

	tagCmd.AddCommand(tagListCmd, tagCreateCmd, tagDeleteCmd)
	rootCmd.AddCommand(tagCmd)
}

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage repository tags",
}

var tagListSort string

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tags in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		tags, err := client.Tags(ws, repo).List(context.Background(), tagListSort)
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
		var input bitbucket.CreateTagInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreateTagInput {
			return bitbucket.CreateTagInput{
				Name:   tagCreateName,
				Target: bitbucket.BranchTarget{Hash: tagCreateFrom},
			}
		})
		if err != nil {
			return err
		}
		if !consumed {
			if err := requireFlag("name", tagCreateName); err != nil {
				return err
			}
			if err := requireFlag("from", tagCreateFrom); err != nil {
				return err
			}
		}
		tag, err := client.Tags(ws, repo).Create(context.Background(), input.Name, input.Target.Hash)
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
	// no MarkFlagRequired — tag create accepts JSON on stdin.

	tagListCmd.Flags().StringVar(&tagListSort, "sort", "",
		"sort by Bitbucket field, prefix with - for descending (e.g. -target.date); empty preserves API default")

	tagDeleteCmd.Flags().StringVarP(&tagDeleteName, "name", "n", "", "tag name to delete (required)")
	tagDeleteCmd.MarkFlagRequired("name")

	tagCmd.AddCommand(tagListCmd, tagCreateCmd, tagDeleteCmd)
	rootCmd.AddCommand(tagCmd)
}

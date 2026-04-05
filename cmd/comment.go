package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage pull request comments",
}

var commentListPRID int

var commentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List comments on a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		comments, err := client.Comments(ws, r, commentListPRID).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(comments, func() { render.CommentList(comments) })
	},
}

var (
	commentGetPRID      int
	commentGetCommentID int
)

var commentGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a single comment on a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		c, err := client.Comments(ws, r, commentGetPRID).Get(context.Background(), commentGetCommentID)
		if err != nil {
			return err
		}
		return printOutput(c, func() { render.CommentDetail(c) })
	},
}

var (
	commentAddPRID int
	commentAddText string
	commentAddFile string
	commentAddLine int
)

var commentAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a comment to a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		input := bitbucket.AddCommentInput{
			Content: bitbucket.Content{Raw: commentAddText},
		}
		if commentAddFile != "" {
			input.Inline = &bitbucket.Inline{
				Path: commentAddFile,
				To:   commentAddLine,
			}
		}
		c, err := client.Comments(ws, r, commentAddPRID).Add(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(c, func() {
			fmt.Printf("Comment added: id=%d\n", c.ID)
		})
	},
}

var (
	commentReplyPRID      int
	commentReplyCommentID int
	commentReplyText      string
)

var commentReplyCmd = &cobra.Command{
	Use:   "reply",
	Short: "Reply to a pull request comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		c, err := client.Comments(ws, r, commentReplyPRID).Reply(
			context.Background(), commentReplyCommentID, commentReplyText)
		if err != nil {
			return err
		}
		return printOutput(c, func() {
			fmt.Printf("Reply added: id=%d\n", c.ID)
		})
	},
}

func init() {
	prCmd.AddCommand(commentCmd)
	commentCmd.AddCommand(commentListCmd, commentGetCmd, commentAddCmd, commentReplyCmd)

	commentListCmd.Flags().IntVarP(&commentListPRID, "pr-id", "p", 0, "pull request ID")
	commentListCmd.MarkFlagRequired("pr-id")

	commentGetCmd.Flags().IntVarP(&commentGetPRID, "pr-id", "p", 0, "pull request ID")
	commentGetCmd.Flags().IntVarP(&commentGetCommentID, "comment-id", "c", 0, "comment ID")
	commentGetCmd.MarkFlagRequired("pr-id")
	commentGetCmd.MarkFlagRequired("comment-id")

	commentAddCmd.Flags().IntVarP(&commentAddPRID, "pr-id", "p", 0, "pull request ID")
	commentAddCmd.Flags().StringVarP(&commentAddText, "text", "t", "", "comment text")
	commentAddCmd.Flags().StringVar(&commentAddFile, "file", "", "file path for inline comment")
	commentAddCmd.Flags().IntVar(&commentAddLine, "line", 0, "line number for inline comment")
	commentAddCmd.MarkFlagRequired("pr-id")
	commentAddCmd.MarkFlagRequired("text")

	commentReplyCmd.Flags().IntVarP(&commentReplyPRID, "pr-id", "p", 0, "pull request ID")
	commentReplyCmd.Flags().IntVarP(&commentReplyCommentID, "comment-id", "c", 0, "parent comment ID")
	commentReplyCmd.Flags().StringVarP(&commentReplyText, "text", "t", "", "reply text")
	commentReplyCmd.MarkFlagRequired("pr-id")
	commentReplyCmd.MarkFlagRequired("comment-id")
	commentReplyCmd.MarkFlagRequired("text")
}

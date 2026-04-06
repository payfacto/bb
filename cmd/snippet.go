package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var snippetCmd = &cobra.Command{
	Use:   "snippet",
	Short: "Manage workspace snippets",
}

var snippetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List snippets in the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		snippets, err := client.Snippets(ws).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(snippets, func() { render.SnippetList(snippets) })
	},
}

var snippetGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a snippet by its ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		snippet, err := client.Snippets(ws).Get(context.Background(), args[0])
		if err != nil {
			return err
		}
		return printOutput(snippet, func() { render.SnippetDetail(snippet) })
	},
}

var (
	snippetCreateTitle   string
	snippetCreatePrivate bool
	snippetCreateFile    string
)

var snippetCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new snippet",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		var content *os.File
		var filename string
		if snippetCreateFile != "" {
			f, err := os.Open(snippetCreateFile) //nolint:gosec
			if err != nil {
				return fmt.Errorf("open file: %w", err)
			}
			defer f.Close()
			content = f
			filename = snippetCreateFile
		}
		snippet, err := client.Snippets(ws).Create(context.Background(), snippetCreateTitle, filename, snippetCreatePrivate, content)
		if err != nil {
			return err
		}
		return printOutput(snippet, func() { render.SnippetDetail(snippet) })
	},
}

var snippetDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a snippet by its ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		if err := client.Snippets(ws).Delete(context.Background(), args[0]); err != nil {
			return err
		}
		fmt.Printf("Snippet %s deleted\n", args[0])
		return nil
	},
}

func init() {
	snippetCreateCmd.Flags().StringVar(&snippetCreateTitle, "title", "", "snippet title (required)")
	snippetCreateCmd.Flags().BoolVar(&snippetCreatePrivate, "private", false, "make the snippet private")
	snippetCreateCmd.Flags().StringVar(&snippetCreateFile, "file", "", "path to a file to upload as snippet content")
	_ = snippetCreateCmd.MarkFlagRequired("title")

	snippetCmd.AddCommand(snippetListCmd)
	snippetCmd.AddCommand(snippetGetCmd)
	snippetCmd.AddCommand(snippetCreateCmd)
	snippetCmd.AddCommand(snippetDeleteCmd)
	rootCmd.AddCommand(snippetCmd)
}

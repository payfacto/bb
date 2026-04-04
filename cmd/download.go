package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Manage repository download artifacts",
}

var downloadListCmd = &cobra.Command{
	Use:   "list",
	Short: "List download artifacts in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		downloads, err := client.Downloads(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(downloads, func() {
			if len(downloads) == 0 {
				fmt.Println("No downloads found.")
				return
			}
			for _, d := range downloads {
				fmt.Printf("%-40s  %d bytes\n", truncate(d.Name, 40), d.Size)
			}
		})
	},
}

var downloadUploadFile string

var downloadUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file as a download artifact",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		f, err := os.Open(downloadUploadFile)
		if err != nil {
			return err
		}
		defer f.Close()
		name := filepath.Base(downloadUploadFile)
		if err := client.Downloads(ws, repo).Upload(context.Background(), name, f); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "uploaded", "file": name}, func() {
			fmt.Printf("Uploaded '%s'.\n", name)
		})
	},
}

var downloadDeleteName string

var downloadDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a download artifact by filename",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Downloads(ws, repo).Delete(context.Background(), downloadDeleteName); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "deleted", "name": downloadDeleteName}, func() {
			fmt.Printf("Download '%s' deleted.\n", downloadDeleteName)
		})
	},
}

func init() {
	downloadUploadCmd.Flags().StringVar(&downloadUploadFile, "file", "", "path to the file to upload (required)")
	downloadUploadCmd.MarkFlagRequired("file")

	downloadDeleteCmd.Flags().StringVar(&downloadDeleteName, "name", "", "filename to delete (required)")
	downloadDeleteCmd.MarkFlagRequired("name")

	downloadCmd.AddCommand(downloadListCmd, downloadUploadCmd, downloadDeleteCmd)
	rootCmd.AddCommand(downloadCmd)
}

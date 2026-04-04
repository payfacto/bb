package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/pkg/bitbucket"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage repository webhooks",
}

var webhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhooks for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		hooks, err := client.Webhooks(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(hooks, func() {
			if len(hooks) == 0 {
				fmt.Println("No webhooks found.")
				return
			}
			for _, h := range hooks {
				fmt.Printf("%-38s  %-8v  %s\n", h.UUID, h.Active, truncate(h.URL, 60))
			}
		})
	},
}

var (
	webhookCreateURL         string
	webhookCreateEvents      []string
	webhookCreateDescription string
	webhookCreateActive      bool
)

var webhookCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		input := bitbucket.CreateWebhookInput{
			Description: webhookCreateDescription,
			URL:         webhookCreateURL,
			Active:      webhookCreateActive,
			Events:      webhookCreateEvents,
		}
		h, err := client.Webhooks(ws, repo).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(h, func() {
			fmt.Printf("Webhook created (UUID: %s).\n", h.UUID)
		})
	},
}

var webhookDeleteUUID string

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a webhook by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Webhooks(ws, repo).Delete(context.Background(), webhookDeleteUUID); err != nil {
			return err
		}
		return printOutput(map[string]any{"result": "deleted", "uuid": webhookDeleteUUID}, func() {
			fmt.Printf("Webhook %s deleted.\n", webhookDeleteUUID)
		})
	},
}

func init() {
	webhookCreateCmd.Flags().StringVar(&webhookCreateURL, "url", "", "webhook endpoint URL (required)")
	webhookCreateCmd.Flags().StringArrayVar(&webhookCreateEvents, "events", nil, "event type to subscribe to, e.g. repo:push (repeatable, required)")
	webhookCreateCmd.Flags().StringVar(&webhookCreateDescription, "description", "", "webhook description")
	webhookCreateCmd.Flags().BoolVar(&webhookCreateActive, "active", true, "whether the webhook is active")
	webhookCreateCmd.MarkFlagRequired("url")
	webhookCreateCmd.MarkFlagRequired("events")

	webhookDeleteCmd.Flags().StringVar(&webhookDeleteUUID, "uuid", "", "webhook UUID to delete (required)")
	webhookDeleteCmd.MarkFlagRequired("uuid")

	webhookCmd.AddCommand(webhookListCmd, webhookCreateCmd, webhookDeleteCmd)
	rootCmd.AddCommand(webhookCmd)
}

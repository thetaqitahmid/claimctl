package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// webhooksCmd represents the webhooks command
var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhooks",
}

var (
	webhookName        string
	webhookUrl         string
	webhookMethod      string
	webhookHeaders     []string
	webhookTemplate    string
	webhookDescription string
	webhookEvents      string
	webhookFile        string
)

// listWebhooksCmd
var listWebhooksCmd = &cobra.Command{
	Use:   "list",
	Short: "List all webhooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		webhooks, err := client.ListWebhooks()
		if err != nil {
			return fmt.Errorf("error fetching webhooks: %w", err)
		}

		if viper.GetBool("json") {
			data, err := json.MarshalIndent(webhooks, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tURL\tMETHOD\tCREATED")
		for _, wbk := range webhooks {
			created := time.Unix(wbk.CreatedAt, 0).Format("2006-01-02 15:04")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", wbk.ID, wbk.Name, wbk.Url, wbk.Method, created)
		}
		w.Flush()
		return nil
	},
}

// createWebhookCmd
var createWebhookCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new webhook",
	Long: `Create a new webhook.

Use flags for single webhook creation:
  claimctl webhooks create --name "Slack" --url "https://hooks.slack.com/..."

Use --file for bulk creation from a JSON file:
  claimctl webhooks create --file webhooks.json

  Format for webhooks.json:
  [
    {
      "name": "Slack Notification",
      "url": "https://hooks.slack.com/services/...",
      "method": "POST",
      "headers": {"Content-Type": "application/json"},
      "template": "{\"text\": \"{{.message}}\"}",
      "description": "Slack webhook for notifications"
    }
  ]`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		// Bulk Creation from file
		if webhookFile != "" {
			data, err := os.ReadFile(webhookFile)
			if err != nil {
				return fmt.Errorf("error reading file %s: %w", webhookFile, err)
			}

			var webhooks []api.CreateWebhookRequest
			if err := json.Unmarshal(data, &webhooks); err != nil {
				return fmt.Errorf("error parsing JSON file: %w", err)
			}

			successCount := 0
			var createdWebhooks []*api.Webhook
			var errorMsgs []string

			for _, w := range webhooks {
				wbk, err := client.CreateWebhook(w)
				if err != nil {
					msg := fmt.Sprintf("Failed to create webhook '%s': %v", w.Name, err)
					fmt.Fprintln(os.Stderr, msg)
					errorMsgs = append(errorMsgs, msg)
					continue
				}
				if !viper.GetBool("json") {
					fmt.Printf("Created webhook '%s' (ID: %s)\n", wbk.Name, wbk.ID)
					if wbk.SigningSecret != "" {
						fmt.Printf("  Signing Secret: %s\n", wbk.SigningSecret)
					}
				}
				createdWebhooks = append(createdWebhooks, wbk)
				successCount++
			}

			if viper.GetBool("json") {
				result := struct {
					Created  int            `json:"created"`
					Total    int            `json:"total"`
					Webhooks []*api.Webhook `json:"webhooks"`
				}{
					Created:  successCount,
					Total:    len(webhooks),
					Webhooks: createdWebhooks,
				}
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("error marshalling to JSON: %w", err)
				}
				fmt.Println(string(data))
			} else {
				fmt.Printf("Successfully created %d/%d webhooks.\n", successCount, len(webhooks))
			}

			if len(errorMsgs) > 0 {
				return fmt.Errorf("bulk creation completed with errors")
			}
			return nil
		}

		// Single Creation - validate required flags
		if webhookName == "" {
			return fmt.Errorf("--name is required for single webhook creation")
		}
		if webhookUrl == "" {
			return fmt.Errorf("--url is required for single webhook creation")
		}

		// Parse Headers
		headerMap := make(map[string]string)
		for _, h := range webhookHeaders {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				headerMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		req := api.CreateWebhookRequest{
			Name:        webhookName,
			Url:         webhookUrl,
			Method:      webhookMethod,
			Headers:     headerMap,
			Template:    webhookTemplate,
			Description: webhookDescription,
		}

		wbk, err := client.CreateWebhook(req)
		if err != nil {
			return fmt.Errorf("error creating webhook: %w", err)
		}

		if viper.GetBool("json") {
			data, err := json.MarshalIndent(wbk, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Webhook created successfully with ID: %s\n", wbk.ID)
		if wbk.SigningSecret != "" {
			fmt.Printf("Signing Secret: %s\n", wbk.SigningSecret)
		}
		return nil
	},
}

// deleteWebhookCmd
var deleteWebhookCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a webhook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		id := args[0]

		if err := client.DeleteWebhook(id); err != nil {
			return fmt.Errorf("error deleting webhook: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("Webhook %s deleted successfully\n", id)
		}
		return nil
	},
}

// attachWebhookCmd
var attachWebhookCmd = &cobra.Command{
	Use:   "attach [resource-id] [webhook-id]",
	Short: "Attach a webhook to a resource",
	Args:  cobra.ExactArgs(2),
	Long: `Attach a webhook to a resource with specific events.
Available events: 'reservation.created', 'reservation.cancelled', 'reservation.activated', 'reservation.completed', 'reservation.expired'.
Use comma-separated list for --events (e.g. "reservation.created,reservation.cancelled")`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		resourceID := args[0]
		webhookID := args[1]

		events := []string{}
		if webhookEvents != "" {
			events = strings.Split(webhookEvents, ",")
			for i, e := range events {
				events[i] = strings.TrimSpace(e)
			}
		}

		req := api.AddResourceWebhookRequest{
			WebhookID: webhookID,
			Events:    events,
		}

		if err := client.AttachWebhook(resourceID, req); err != nil {
			return fmt.Errorf("error attaching webhook: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("Webhook %s attached to resource %s\n", webhookID, resourceID)
		}
		return nil
	},
}

// detachWebhookCmd
var detachWebhookCmd = &cobra.Command{
	Use:   "detach [resource-id] [webhook-id]",
	Short: "Detach a webhook from a resource",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		resourceID := args[0]
		webhookID := args[1]

		if err := client.DetachWebhook(resourceID, webhookID); err != nil {
			return fmt.Errorf("error detaching webhook: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("Webhook %s detached from resource %s\n", webhookID, resourceID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(webhooksCmd)
	webhooksCmd.AddCommand(listWebhooksCmd)
	webhooksCmd.AddCommand(createWebhookCmd)
	webhooksCmd.AddCommand(deleteWebhookCmd)
	webhooksCmd.AddCommand(attachWebhookCmd)
	webhooksCmd.AddCommand(detachWebhookCmd)

	createWebhookCmd.Flags().StringVar(&webhookName, "name", "", "Webhook Name")
	createWebhookCmd.Flags().StringVar(&webhookUrl, "url", "", "Webhook URL")
	createWebhookCmd.Flags().StringVar(&webhookMethod, "method", "POST", "HTTP Method")
	createWebhookCmd.Flags().StringSliceVar(&webhookHeaders, "header", []string{}, "HTTP Headers (key:value)")
	createWebhookCmd.Flags().StringVar(&webhookTemplate, "template", "", "Payload Template")
	createWebhookCmd.Flags().StringVar(&webhookDescription, "desc", "", "Description")
	createWebhookCmd.Flags().StringVar(&webhookFile, "file", "", "JSON file for bulk creation")
	// createWebhookCmd.MarkFlagRequired("name") // No longer required if file is present
	// We need to handle validation manually in Run logic now

	attachWebhookCmd.Flags().StringVar(&webhookEvents, "events", "", "Comma-separated list of events")
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var backupOutput string

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a backup of all application data",
	Long: `Downloads a full JSON backup from the claimctl server.
The backup includes all resources, users, reservations, settings,
webhooks, secrets, and other configuration data.

Secrets are exported in their encrypted form. The same
APP_ENCRYPTION_KEY must be configured on the target instance
for secrets to remain usable after a restore.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		data, err := client.CreateBackup()
		if err != nil {
			return fmt.Errorf("error creating backup: %w", err)
		}

		// Determine output filename
		output := backupOutput
		if output == "" {
			output = fmt.Sprintf("claimctl-backup-%s.json",
				time.Now().Format("2006-01-02T150405"))
		}

		if err := os.WriteFile(output, data, 0600); err != nil {
			return fmt.Errorf("error writing backup file: %w", err)
		}

		if viper.GetBool("json") {
			result := map[string]interface{}{
				"status": "success",
				"file":   output,
				"bytes":  len(data),
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("Backup saved to %s (%d bytes)\n", output, len(data))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "",
		"Output file path (default: claimctl-backup-<timestamp>.json)")
}

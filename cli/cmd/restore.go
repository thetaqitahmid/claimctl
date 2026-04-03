package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var forceRestore bool

var restoreCmd = &cobra.Command{
	Use:   "restore <file>",
	Short: "Restore application data from a backup file",
	Long: `Uploads a JSON backup file to the claimctl server and
replaces all existing data. This operation:

  - Truncates ALL existing tables
  - Imports data from the backup file
  - Resets database sequences

WARNING: This is a destructive operation. All existing data
will be permanently replaced.

Use --force to confirm that you want to proceed even if the
instance already has data.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		if !forceRestore {
			fmt.Println("WARNING: This will replace ALL existing data.")
			fmt.Println("Use --force to confirm this operation.")
			return fmt.Errorf("restore aborted: use --force to confirm")
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("error reading backup file: %w", err)
		}

		// Basic validation: check that it parses as JSON
		var check map[string]any
		if err := json.Unmarshal(data, &check); err != nil {
			return fmt.Errorf("invalid backup file (not valid JSON): %w", err)
		}
		if _, ok := check["metadata"]; !ok {
			return fmt.Errorf("invalid backup file: missing metadata field")
		}

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		if err := client.RestoreBackup(data); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}

		if viper.GetBool("json") {
			result := map[string]any{
				"status":  "success",
				"message": "Backup restored successfully",
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Println("Backup restored successfully.")
			fmt.Println("Note: All JWT tokens have been invalidated. Users will need to log in again.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().BoolVar(&forceRestore, "force", false,
		"Confirm destructive restore (required)")
}

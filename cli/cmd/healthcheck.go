package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// healthcheckCmd represents the healthcheck command
var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Manage health checks for resources",
	Long:  `Configure and monitor health checks for resources.`,
}

// configCmd represents the config subcommand
var healthConfigCmd = &cobra.Command{
	Use:   "config <resource-id>",
	Short: "Configure health check for a resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceID := args[0]

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		checkType, _ := cmd.Flags().GetString("type")
		target, _ := cmd.Flags().GetString("target")
		enabled, _ := cmd.Flags().GetBool("enabled")
		interval, _ := cmd.Flags().GetInt32("interval")
		timeout, _ := cmd.Flags().GetInt32("timeout")
		retry, _ := cmd.Flags().GetInt32("retry")

		req := api.HealthConfigRequest{
			Enabled:         enabled,
			CheckType:       checkType,
			Target:          target,
			IntervalSeconds: interval,
			TimeoutSeconds:  timeout,
			RetryCount:      retry,
		}

		config, err := client.UpsertHealthConfig(resourceID, req)
		if err != nil {
			return fmt.Errorf("error configuring health check: %w", err)
		}

		fmt.Println("Health check configured successfully:")
		displayHealthConfig(config)
		return nil
	},
}

// statusCmd represents the status subcommand
var healthStatusCmd = &cobra.Command{
	Use:   "status <resource-id>",
	Short: "Get current health status of a resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceID := args[0]

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		status, err := client.GetHealthStatus(resourceID)
		if err != nil {
			return fmt.Errorf("error getting health status: %w", err)
		}

		displayHealthStatus(status)
		return nil
	},
}

// historyCmd represents the history subcommand
var healthHistoryCmd = &cobra.Command{
	Use:   "history <resource-id>",
	Short: "Get health check history for a resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceID := args[0]

		limit, _ := cmd.Flags().GetInt32("limit")

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		history, err := client.GetHealthHistory(resourceID, limit)
		if err != nil {
			return fmt.Errorf("error getting health history: %w", err)
		}

		if len(history) == 0 {
			fmt.Println("No health check history found")
			return nil
		}

		fmt.Printf("Health Check History (showing last %d entries):\n", len(history))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TIME\tSTATUS\tRESPONSE TIME\tERROR")
		for _, h := range history {
			timestamp := time.Unix(h.CheckedAt, 0).Format("2006-01-02 15:04:05")
			errorMsg := ""
			if h.ErrorMessage != "" {
				errorMsg = h.ErrorMessage
			}
			fmt.Fprintf(w, "%s\t%s\t%dms\t%s\n", timestamp, h.Status, h.ResponseTimeMs, errorMsg)
		}
		w.Flush()
		return nil
	},
}

// triggerCmd represents the trigger subcommand
var healthTriggerCmd = &cobra.Command{
	Use:   "trigger <resource-id>",
	Short: "Trigger an immediate health check for a resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceID := args[0]

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		err = client.TriggerHealthCheck(resourceID)
		if err != nil {
			return fmt.Errorf("error triggering health check: %w", err)
		}

		fmt.Println("Health check triggered successfully")
		return nil
	},
}

// getCmd represents the get subcommand
var healthGetCmd = &cobra.Command{
	Use:   "get <resource-id>",
	Short: "Get health check configuration for a resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceID := args[0]

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		config, err := client.GetHealthConfig(resourceID)
		if err != nil {
			return fmt.Errorf("error getting health check configuration: %w", err)
		}

		displayHealthConfig(config)
		return nil
	},
}

// deleteCmd represents the delete subcommand
var healthDeleteCmd = &cobra.Command{
	Use:   "delete <resource-id>",
	Short: "Delete health check configuration for a resource",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceID := args[0]

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		err = client.DeleteHealthConfig(resourceID)
		if err != nil {
			return fmt.Errorf("error deleting health check configuration: %w", err)
		}

		fmt.Println("Health check configuration deleted successfully")
		return nil
	},
}

func displayHealthConfig(config *api.HealthConfig) {
	if viper.GetBool("json") {
		jsonData, _ := json.MarshalIndent(config, "", "  ")
		fmt.Println(string(jsonData))
		return
	}

	fmt.Printf("Resource ID:     %s\n", config.ResourceID)
	fmt.Printf("Enabled:         %t\n", config.Enabled)
	fmt.Printf("Check Type:      %s\n", config.CheckType)
	fmt.Printf("Target:          %s\n", config.Target)
	fmt.Printf("Interval:        %d seconds\n", config.IntervalSeconds)
	fmt.Printf("Timeout:         %d seconds\n", config.TimeoutSeconds)
	fmt.Printf("Retry Count:     %d\n", config.RetryCount)
	fmt.Printf("Created At:      %s\n", time.Unix(config.CreatedAt, 0).Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated At:      %s\n", time.Unix(config.UpdatedAt, 0).Format("2006-01-02 15:04:05"))
}

func displayHealthStatus(status *api.HealthStatus) {
	if viper.GetBool("json") {
		jsonData, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(jsonData))
		return
	}

	fmt.Printf("Resource ID:     %s\n", status.ResourceID)
	fmt.Printf("Status:          %s\n", status.Status)
	fmt.Printf("Response Time:   %d ms\n", status.ResponseTimeMs)
	if status.ErrorMessage != "" {
		fmt.Printf("Error:           %s\n", status.ErrorMessage)
	}
	fmt.Printf("Last Checked:    %s\n", time.Unix(status.CheckedAt, 0).Format("2006-01-02 15:04:05"))
}

func init() {
	rootCmd.AddCommand(healthcheckCmd)
	healthcheckCmd.AddCommand(healthConfigCmd)
	healthcheckCmd.AddCommand(healthStatusCmd)
	healthcheckCmd.AddCommand(healthHistoryCmd)
	healthcheckCmd.AddCommand(healthTriggerCmd)
	healthcheckCmd.AddCommand(healthGetCmd)
	healthcheckCmd.AddCommand(healthDeleteCmd)

	// Config flags
	healthConfigCmd.Flags().String("type", "", "Health check type: ping, http, or tcp")
	healthConfigCmd.Flags().String("target", "", "Target address or URL")
	healthConfigCmd.Flags().Bool("enabled", true, "Enable health check")
	healthConfigCmd.Flags().Int32("interval", 60, "Check interval in seconds")
	healthConfigCmd.Flags().Int32("timeout", 5, "Timeout in seconds")
	healthConfigCmd.Flags().Int32("retry", 3, "Number of retries")
	healthConfigCmd.MarkFlagRequired("type")
	healthConfigCmd.MarkFlagRequired("target")

	// History flags
	healthHistoryCmd.Flags().Int32("limit", 10, "Number of history entries to show")
}

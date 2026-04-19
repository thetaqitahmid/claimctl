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

var auditCmd = &cobra.Command{
	Use:   "audit-logs",
	Short: "View audit logs (admin only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		logs, err := client.GetAuditLogs(limit, offset)
		if err != nil {
			return fmt.Errorf("error fetching audit logs: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(logs, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(logs) == 0 {
			fmt.Println("No audit logs found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "TIME\tACTOR\tACTION\tENTITY\tENTITY ID\tIP")
		for _, l := range logs {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04:05"),
				l.ActorEmail,
				l.Action,
				l.EntityType,
				l.EntityID,
				l.IPAddress,
			)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)

	auditCmd.Flags().Int("limit", 50, "Number of log entries to fetch")
	auditCmd.Flags().Int("offset", 0, "Offset for pagination")
}

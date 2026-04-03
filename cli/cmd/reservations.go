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

// reservationsCmd represents the reservations command
var reservationsCmd = &cobra.Command{
	Use:   "reservations",
	Short: "Manage your reservations",
}

var reservationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your active and queued reservations",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		reservations, err := client.GetUserReservations()
		if err != nil {
			return fmt.Errorf("error fetching reservations: %w", err)
		}

		if len(reservations) == 0 {
			fmt.Println("No active or pending reservations found.")
			return nil
		}

		if viper.GetBool("json") {
			data, err := json.MarshalIndent(reservations, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tRESOURCE\tTYPE\tSTATUS\tQUEUE POS\tCREATED")
		for _, r := range reservations {
			queuePos := "-"
			if r.QueuePosition != nil && *r.QueuePosition > 0 {
				queuePos = fmt.Sprintf("%d", *r.QueuePosition)
			}

			// Format timestamp
			created := time.Unix(r.CreatedAt, 0).Format("2006-01-02 15:04")

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				r.ID,
				r.ResourceName,
				r.ResourceType,
				strings.Title(r.Status),
				queuePos,
				created,
			)
		}
		w.Flush()
		return nil
	},
}

var reservationsStatusCmd = &cobra.Command{
	Use:   "status <reservation-id>",
	Short: "Get status of a specific reservation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		id := args[0]

		reservation, err := client.GetReservation(id)
		if err != nil {
			return fmt.Errorf("error fetching reservation: %w", err)
		}

		if viper.GetBool("json") {
			data, err := json.MarshalIndent(reservation, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		// Human-readable output
		fmt.Printf("Reservation ID: %s\n", reservation.ID)
		fmt.Printf("Resource ID: %s\n", reservation.ResourceID)
		fmt.Printf("Status: %s\n", strings.Title(reservation.Status))
		if reservation.QueuePosition != nil && *reservation.QueuePosition > 0 {
			fmt.Printf("Queue Position: %d\n", *reservation.QueuePosition)
		}
		fmt.Printf("Created At: %s\n", time.Unix(reservation.CreatedAt, 0).Format("2006-01-02 15:04:05"))

		return nil
	},
}

var (
	waitTimeout      int
	waitPollInterval int
)

var reservationsWaitCmd = &cobra.Command{
	Use:   "wait <reservation-id>",
	Short: "Wait for a reservation to become active",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		id := args[0]

		// Check current status
		reservation, err := client.GetReservation(id)
		if err != nil {
			return fmt.Errorf("error fetching reservation: %w", err)
		}

		if reservation.Status == "active" {
			if !viper.GetBool("json") {
				fmt.Println("Reservation is already active")
			}
			return nil
		}

		if reservation.Status == "cancelled" || reservation.Status == "completed" {
			return fmt.Errorf("reservation is %s", reservation.Status)
		}

		// Wait for activation
		if !viper.GetBool("json") {
			fmt.Fprintf(os.Stderr, "Waiting for reservation to become active (timeout: %ds)...\n", waitTimeout)
		}

		progressFn := func(status string, queuePos *int32) {
			if !viper.GetBool("json") {
				pos := "N/A"
				if queuePos != nil && *queuePos > 0 {
					pos = fmt.Sprintf("%d", *queuePos)
				}
				fmt.Fprintf(os.Stderr, "Status: %s | Queue Position: %s\n", status, pos)
			}
		}

		err = client.WaitForReservation(id, waitTimeout, waitPollInterval, progressFn)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				// Cancel the reservation before exiting on timeout
				if !viper.GetBool("json") {
					fmt.Fprintf(os.Stderr, "Wait timeout expired. Cancelling reservation %s...\n", id)
				}
				if cancelErr := client.CancelReservation(id); cancelErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to cancel reservation: %v\n", cancelErr)
				}
				return NewTimeoutError("Timeout waiting for reservation to become active")
			} else if strings.Contains(err.Error(), "cancelled") {
				return NewCancelledError("Reservation was cancelled")
			}
			return err
		}

		if !viper.GetBool("json") {
			fmt.Println("Reservation is now active")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(reservationsCmd)
	reservationsCmd.AddCommand(reservationsListCmd)
	reservationsCmd.AddCommand(reservationsStatusCmd)
	reservationsCmd.AddCommand(reservationsWaitCmd)

	reservationsWaitCmd.Flags().IntVar(&waitTimeout, "timeout", 300, "Timeout in seconds (default: 300)")
	reservationsWaitCmd.Flags().IntVar(&waitPollInterval, "poll-interval", 5, "Polling interval in seconds (default: 5)")
}

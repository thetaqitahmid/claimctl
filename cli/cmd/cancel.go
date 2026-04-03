package cmd

import (
	"encoding/json"
	"fmt"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// cancelCmd represents the cancel command
var cancelCmd = &cobra.Command{
	Use:   "cancel [reservation-id]",
	Short: "Cancel a reservation",
	Long: `Cancel a reservation.
This is useful for removing yourself from the queue (pending reservations) or ending an active reservation early.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		reservationID := args[0]

		if err := client.CancelReservation(reservationID); err != nil {
			return fmt.Errorf("error cancelling reservation: %w", err)
		}

		if viper.GetBool("json") {
			res := map[string]interface{}{
				"id":     reservationID,
				"status": "cancelled",
			}
			data, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Successfully cancelled reservation %s\n", reservationID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}

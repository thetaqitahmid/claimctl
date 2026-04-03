package cmd

import (
	"encoding/json"
	"fmt"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release [reservation-id]",
	Short: "Release (complete) a reservation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		reservationID := args[0]

		if err := client.CompleteReservation(reservationID); err != nil {
			return fmt.Errorf("error releasing reservation: %w", err)
		}

		if viper.GetBool("json") {
			res := map[string]interface{}{
				"id":     reservationID,
				"status": "completed",
			}
			data, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Successfully released reservation %s\n", reservationID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}

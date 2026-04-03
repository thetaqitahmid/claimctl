package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	reserveType           string
	reserveLabelExpr      string
	reserveName           string
	reserveDuration       string
	reserveWait           bool
	reserveTimeout        int
	reservePollInterval   int
	reserveQuiet          bool
	reserveNoQueue        bool
	reserveRequireHealthy bool
)

// reserveCmd represents the reserve command
var reserveCmd = &cobra.Command{
	Use:   "reserve [resource-id]",
	Short: "Create a reservation",
	Long: `Reserve a resource by ID (default), Name, or by Type/Label.
If reserving by Type/Label, the system will select the first available resource matching the criteria.

Examples:
  claimctl reserve 12
  claimctl reserve --name "Meeting Room A"
  claimctl reserve --type "Conference Room"
  claimctl reserve --label "projector"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		var resourceID string
		var resourceName string

		if len(args) > 0 {
			// Expecting ID
			resourceID = args[0]

			// Check health if required
			if reserveRequireHealthy {
				status, err := client.GetHealthStatus(resourceID)
				if err != nil {
					return fmt.Errorf("error checking resource health: %w", err)
				}
				if !strings.EqualFold(status.Status, "healthy") {
					return fmt.Errorf("resource %s is not healthy (status: %s)", resourceID, status.Status)
				}
			}
		} else if reserveName != "" {
			resourceName = reserveName
		}

		if resourceID == "" && resourceName == "" && reserveType == "" && reserveLabelExpr == "" {
			cmd.Usage()
			return fmt.Errorf("must provide a Resource ID, --name, or --type/--label-expr flags")
		}

		if resourceID == "" {
			// Search for resource
			fmt.Printf("Searching for resource...")
			if resourceName != "" {
				fmt.Printf(" (Name: '%s')\n", resourceName)
			} else {
				fmt.Printf(" (Type: %s, Label Expression: %s)\n", reserveType, reserveLabelExpr)
			}

			resources, err := client.GetResources(reserveLabelExpr)
			if err != nil {
				return fmt.Errorf("error fetching resources: %w", err)
			}

			found := false
			for _, r := range resources {
				// 1. Name Match (Exact, Case-Insensitive?)
				if resourceName != "" {
					if strings.EqualFold(r.Name, resourceName) {
						resourceID = r.ID
						fmt.Printf("Found resource: %s (ID: %s)\n", r.Name, r.ID)
						found = true
						break
					}
					continue
				}

				// 2. Type/Label Match (Must be Available)
				// Check status
				if !strings.EqualFold(r.Status, "Available") && !strings.EqualFold(r.Status, "available") {
					// Note: CLI types.go sets "Available", "Reserved", "Queue".
					continue
				}

				// Check Health if required
				if reserveRequireHealthy {
					if !strings.EqualFold(r.Health, "healthy") {
						continue
					}
				}

				// Check Type
				if reserveType != "" && !strings.EqualFold(r.Type, reserveType) {
					continue
				}

				// Match found!
				resourceID = r.ID
				fmt.Printf("Found available resource: %s (ID: %s)\n", r.Name, r.ID)
				found = true
				break
			}

			if !found {
				return fmt.Errorf("no resource found matching criteria")
			}
		}

		// 2. Perform Reservation
		reservation, err := client.CreateReservation(resourceID, reserveDuration)
		if err != nil {
			return fmt.Errorf("error creating reservation: %w", err)
		}

		// 3. Wait for reservation to become active if requested
		if reserveWait && (strings.EqualFold(reservation.Status, "queued") || (reservation.QueuePosition != nil && *reservation.QueuePosition > 0)) {
			if !reserveQuiet && !viper.GetBool("json") {
				fmt.Fprintf(os.Stderr, "Reservation queued. Waiting for activation (timeout: %ds)...\n", reserveTimeout)
			}

			progressFn := func(status string, queuePos *int32) {
				if !reserveQuiet && !viper.GetBool("json") {
					pos := "N/A"
					if queuePos != nil && *queuePos > 0 {
						pos = fmt.Sprintf("%d", *queuePos)
					}
					fmt.Fprintf(os.Stderr, "Status: %s | Queue Position: %s\n", status, pos)
				}
			}

			err = client.WaitForReservation(reservation.ID, reserveTimeout, reservePollInterval, progressFn)
			if err != nil {
				if strings.Contains(err.Error(), "timeout") {
					// Cancel the reservation before exiting on timeout
					if !reserveQuiet && !viper.GetBool("json") {
						fmt.Fprintf(os.Stderr, "Wait timeout expired. Cancelling reservation %s...\n", reservation.ID)
					}
					if cancelErr := client.CancelReservation(reservation.ID); cancelErr != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to cancel reservation: %v\n", cancelErr)
					}
					return NewTimeoutError("Timeout waiting for reservation to become active")
				} else if strings.Contains(err.Error(), "cancelled") {
					return NewCancelledError("Reservation was cancelled")
				}
				return err
			}

			// Fetch updated reservation status
			reservation, err = client.GetReservation(reservation.ID)
			if err != nil {
				return fmt.Errorf("error fetching updated reservation: %w", err)
			}
		}
		if viper.GetBool("json") {
			data, err := json.MarshalIndent(reservation, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		if reserveQuiet {
			// In quiet mode, only output reservation ID
			fmt.Println(reservation.ID)
			return nil
		}

		fmt.Printf("Successfully reserved resource %s. Reservation ID: %s\n", reservation.ResourceID, reservation.ID)

		if strings.EqualFold(reservation.Status, "queued") || (reservation.QueuePosition != nil && *reservation.QueuePosition > 0) {
			if reserveNoQueue {
				// Cancel the reservation immediately
				if cancelErr := client.CancelReservation(reservation.ID); cancelErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to cancel reservation: %v\n", cancelErr)
				}
				return NewResourceBusyError("Resource is busy and --no-queue was specified")
			}

			pos := 0
			if reservation.QueuePosition != nil {
				pos = int(*reservation.QueuePosition)
			}
			fmt.Printf("NOTE: Resource is currently busy. You have been added to the queue at position %d.\n", pos)
			fmt.Printf("To cancel this request, run: ./claimctl cancel %s\n", reservation.ID)
		} else {
			fmt.Printf("To release this resource later, run: ./claimctl release %s\n", reservation.ID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reserveCmd)

	reserveCmd.Flags().StringVar(&reserveName, "name", "", "Reserve by Resource Name")
	reserveCmd.Flags().StringVar(&reserveType, "type", "", "Reserve first available resource of this type")
	reserveCmd.Flags().StringVar(&reserveLabelExpr, "label-expr", "", "Reserve first available resource matching this label expression")
	reserveCmd.Flags().StringVar(&reserveDuration, "duration", "", "Duration of the reservation (e.g. 1h, 30m)")
	reserveCmd.Flags().BoolVar(&reserveWait, "wait", false, "Wait for reservation to become active")
	reserveCmd.Flags().IntVar(&reserveTimeout, "timeout", 300, "Timeout in seconds when waiting (default: 300)")
	reserveCmd.Flags().IntVar(&reservePollInterval, "poll-interval", 5, "Polling interval in seconds (default: 5)")
	reserveCmd.Flags().BoolVar(&reserveQuiet, "quiet", false, "Quiet mode - only output reservation ID")
	reserveCmd.Flags().BoolVar(&reserveNoQueue, "no-queue", false, "Fail if resource is busy (do not queue)")
	reserveCmd.Flags().BoolVar(&reserveRequireHealthy, "require-healthy", false, "Fail if resource is not healthy")

	// Dynamic Completion for Resource IDs (Positional)
	reserveCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		resources, err := client.GetResources("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var suggestions []string
		for _, r := range resources {
			suggestions = append(suggestions, fmt.Sprintf("%s\t%s", r.ID, r.Name))
		}
		return suggestions, cobra.ShellCompDirectiveNoFileComp
	}

	// Dynamic Completion for Flags
	reserveCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		resources, err := client.GetResources("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		seen := make(map[string]bool)
		var types []string
		for _, r := range resources {
			if !seen[r.Type] {
				types = append(types, r.Type)
				seen[r.Type] = true
			}
		}
		return types, cobra.ShellCompDirectiveNoFileComp
	})

	reserveCmd.RegisterFlagCompletionFunc("label-expr", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		resources, err := client.GetResources("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		seen := make(map[string]bool)
		var labels []string
		for _, r := range resources {
			for _, l := range r.Labels {
				if !seen[l] {
					labels = append(labels, l)
					seen[l] = true
				}
			}
		}
		return labels, cobra.ShellCompDirectiveNoFileComp
	})
}

package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"claimctl-cli/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	reserveType           string
	reserveLabelExpr      string
	reserveName           string
	reserveDuration       string
	reserveLeaseDuration  string
	reserveIndefinite     bool
	reserveKey            string
	reserveWait           bool
	reserveTimeout        int
	reservePollInterval   int
	reserveQuiet          bool
	reserveNoQueue        bool
	reserveRequireHealthy bool
)

var reserveCmd = &cobra.Command{
	Use: "reserve [resource-id]", Short: "Atomically claim a resource by ID or selector", Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if reserveTimeout <= 0 || reservePollInterval <= 0 {
			return fmt.Errorf("timeout and poll interval must be positive")
		}
		if reserveIndefinite && reserveDuration != "" {
			return fmt.Errorf("indefinite and duration cannot be combined")
		}
		req := api.ClaimRequest{Name: reserveName, Type: reserveType, LabelExpr: reserveLabelExpr, Duration: reserveDuration, LeaseDuration: reserveLeaseDuration, Indefinite: reserveIndefinite, NoQueue: reserveNoQueue, RequireHealthy: reserveRequireHealthy}
		if len(args) > 0 {
			req.ResourceID = args[0]
		}
		if req.ResourceID == "" && req.Name == "" && req.Type == "" && req.LabelExpr == "" {
			return fmt.Errorf("provide a resource ID or selector")
		}
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		r, err := client.Claim(req, reserveKey)
		if err != nil {
			return err
		}
		if reserveWait && r.Status != "active" {
			var progress func(string, *int32)
			if !reserveQuiet && !viper.GetBool("json") {
				progress = func(status string, pos *int32) { fmt.Fprintf(cmd.ErrOrStderr(), "Reservation %s: %s\n", r.ID, status) }
			}
			if err = client.WaitForReservation(r.ID, reserveTimeout, reservePollInterval, progress); err != nil {
				cleanupErr := cleanupReservation(client, r.ID)
				if cleanupErr != nil {
					return errors.Join(err, fmt.Errorf("cleanup reservation %s: %w", r.ID, cleanupErr))
				}
				return err
			}
			r, err = client.GetReservation(r.ID)
			if err != nil {
				return err
			}
		}
		if viper.GetBool("json") {
			return outputJSON(cmd, r)
		}
		if reserveQuiet {
			fmt.Fprintln(cmd.OutOrStdout(), r.ID)
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Reservation %s: %s (resource %s)\n", r.ID, r.Status, r.ResourceID)
		return nil
	},
}

func cleanupReservation(client *api.Client, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cleanup := *client
	cleanup.Context = ctx
	return cleanup.CancelReservation(id)
}

func init() {
	rootCmd.AddCommand(reserveCmd)
	reserveCmd.Flags().StringVar(&reserveName, "name", "", "Exact resource name")
	reserveCmd.Flags().StringVar(&reserveType, "type", "", "Resource type")
	reserveCmd.Flags().StringVar(&reserveLabelExpr, "label-expr", "", "Label expression")
	reserveCmd.Flags().StringVar(&reserveDuration, "duration", "", "Finite hold duration (defaults to server policy)")
	reserveCmd.Flags().StringVar(&reserveLeaseDuration, "lease-duration", "", "Heartbeat lease duration")
	reserveCmd.Flags().BoolVar(&reserveIndefinite, "indefinite", false, "Explicit indefinite hold, if server policy permits")
	reserveCmd.Flags().StringVar(&reserveKey, "idempotency-key", "", "Creation retry key (random by default)")
	reserveCmd.Flags().BoolVar(&reserveWait, "wait", false, "Wait for activation")
	reserveCmd.Flags().IntVar(&reserveTimeout, "timeout", 300, "Activation timeout in seconds")
	reserveCmd.Flags().IntVar(&reservePollInterval, "poll-interval", 5, "Polling interval in seconds")
	reserveCmd.Flags().BoolVar(&reserveQuiet, "quiet", false, "Print only the reservation ID")
	reserveCmd.Flags().BoolVar(&reserveNoQueue, "no-queue", false, "Fail atomically when busy without entering the queue")
	reserveCmd.Flags().BoolVar(&reserveRequireHealthy, "require-healthy", false, "Require persisted healthy status")
	reserveCmd.ValidArgsFunction = resourceCompletion
}

func resourceCompletion(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
	client, err := newClient(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	resources, err := client.GetResources("")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var result []string
	for _, r := range resources {
		result = append(result, r.ID+"\t"+r.Name)
	}
	return result, cobra.ShellCompDirectiveNoFileComp
}

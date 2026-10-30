package cmd

import (
	"fmt"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
)

func init() {
	policy := &cobra.Command{Use: "policy", Short: "Show the server reservation policy", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		result, err := client.Policy()
		if err != nil {
			return err
		}
		return outputJSON(cmd, result)
	}}
	rootCmd.AddCommand(policy)
	var lease string
	renew := &cobra.Command{Use: "renew <reservation-id>", Aliases: []string{"heartbeat"}, Short: "Renew an existing heartbeat lease", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		result, err := client.Renew(args[0], lease)
		if err != nil {
			return err
		}
		return outputJSON(cmd, result)
	}}
	renew.Flags().StringVar(&lease, "lease-duration", "", "New lease interval (omission reuses existing interval)")
	rootCmd.AddCommand(renew)
	var duration, bundleLease, key string
	bundle := &cobra.Command{Use: "bundle <resource-id>...", Short: "Atomically acquire all resources or none", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		result, err := client.Bundle(api.BundleRequest{ResourceIDs: args, Duration: duration, LeaseDuration: bundleLease}, key)
		if err != nil {
			return err
		}
		return outputJSON(cmd, result)
	}}
	bundle.Flags().StringVar(&duration, "duration", "", "Finite hold duration")
	bundle.Flags().StringVar(&bundleLease, "lease-duration", "", "Heartbeat lease duration")
	bundle.Flags().StringVar(&key, "idempotency-key", "", "Creation retry key (random by default)")
	rootCmd.AddCommand(bundle)
	var from, to int64
	metrics := &cobra.Command{Use: "metrics [resource-id]", Short: "Show resource metrics (admin aggregate when ID omitted)", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if from < 0 || to < 0 || (cmd.Flags().Changed("from") && from == 0) || (cmd.Flags().Changed("to") && to == 0) {
			return fmt.Errorf("metrics timestamps must be positive")
		}
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		id := ""
		if len(args) > 0 {
			id = args[0]
		}
		result, err := client.Metrics(id, from, to)
		if err != nil {
			return err
		}
		return outputJSON(cmd, result)
	}}
	metrics.Flags().Int64Var(&from, "from", 0, "Window start in epoch seconds")
	metrics.Flags().Int64Var(&to, "to", 0, "Window end in epoch seconds")
	rootCmd.AddCommand(metrics)
}

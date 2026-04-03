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

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets",
}

var (
	secretKey         string
	secretValue       string
	secretDescription string
)

// listSecretsCmd
var listSecretsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		secrets, err := client.ListSecrets()
		if err != nil {
			return fmt.Errorf("error fetching secrets: %w", err)
		}

		if viper.GetBool("json") {
			data, err := json.MarshalIndent(secrets, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tKEY\tVALUE\tDESCRIPTION\tCREATED")
		for _, s := range secrets {
			created := time.Unix(s.CreatedAt, 0).Format("2006-01-02 15:04")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.ID, s.Key, s.Value, s.Description, created)
		}
		w.Flush()
		return nil
	},
}

// createSecretCmd
var createSecretCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new secret",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		if secretKey == "" || secretValue == "" {
			return fmt.Errorf("--key and --value are required")
		}

		req := api.CreateSecretRequest{
			Key:         secretKey,
			Value:       secretValue,
			Description: secretDescription,
		}

		secret, err := client.CreateSecret(req)
		if err != nil {
			return fmt.Errorf("error creating secret: %w", err)
		}

		if viper.GetBool("json") {
			data, err := json.MarshalIndent(secret, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Secret created successfully with ID: %s\n", secret.ID)
		return nil
	},
}

// updateSecretCmd
var updateSecretCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		id := args[0]

		if secretValue == "" {
			return fmt.Errorf("--value is required for update")
		}

		secret, err := client.UpdateSecret(id, secretValue, secretDescription)
		if err != nil {
			return fmt.Errorf("error updating secret: %w", err)
		}

		if viper.GetBool("json") {
			data, err := json.MarshalIndent(secret, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Secret %s updated successfully\n", id)
		return nil
	},
}

// deleteSecretCmd
var deleteSecretCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		id := args[0]

		if err := client.DeleteSecret(id); err != nil {
			return fmt.Errorf("error deleting secret: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("Secret %s deleted successfully\n", id)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.AddCommand(listSecretsCmd)
	secretsCmd.AddCommand(createSecretCmd)
	secretsCmd.AddCommand(updateSecretCmd)
	secretsCmd.AddCommand(deleteSecretCmd)

	createSecretCmd.Flags().StringVar(&secretKey, "key", "", "Secret Key")
	createSecretCmd.Flags().StringVar(&secretValue, "value", "", "Secret Value")
	createSecretCmd.Flags().StringVar(&secretDescription, "desc", "", "Description")

	updateSecretCmd.Flags().StringVar(&secretValue, "value", "", "New Secret Value")
	updateSecretCmd.Flags().StringVar(&secretDescription, "desc", "", "New Description")
}

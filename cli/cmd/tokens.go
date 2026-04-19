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

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Manage API tokens",
}

var listTokensCmd = &cobra.Command{
	Use:   "list",
	Short: "List your API tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		tokens, err := client.ListTokens()
		if err != nil {
			return fmt.Errorf("error fetching tokens: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(tokens, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(tokens) == 0 {
			fmt.Println("No API tokens found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tCREATED\tEXPIRES")
		for _, t := range tokens {
			expires := "Never"
			if t.ExpiresAt != nil {
				expires = time.Unix(*t.ExpiresAt, 0).Format("2006-01-02 15:04")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, t.Name, time.Unix(t.CreatedAt, 0).Format("2006-01-02 15:04"), expires)
		}
		w.Flush()
		return nil
	},
}

var (
	tokenName      string
	tokenExpiresIn string
)

var generateTokenCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new API token",
	RunE: func(cmd *cobra.Command, args []string) error {
		if tokenName == "" {
			return fmt.Errorf("--name is required")
		}

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		token, err := client.GenerateToken(tokenName, tokenExpiresIn)
		if err != nil {
			return fmt.Errorf("error generating token: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(token, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Token generated (ID: %s)\n", token.ID)
		fmt.Printf("Token: %s\n", token.Token)
		fmt.Println("Store this token securely — it will not be shown again.")
		return nil
	},
}

var revokeTokenCmd = &cobra.Command{
	Use:   "revoke [id]",
	Short: "Revoke an API token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		if err := client.RevokeToken(args[0]); err != nil {
			return fmt.Errorf("error revoking token: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("Token %s revoked successfully\n", args[0])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tokensCmd)
	tokensCmd.AddCommand(listTokensCmd)
	tokensCmd.AddCommand(generateTokenCmd)
	tokensCmd.AddCommand(revokeTokenCmd)

	generateTokenCmd.Flags().StringVar(&tokenName, "name", "", "Token name")
	generateTokenCmd.Flags().StringVar(&tokenExpiresIn, "expires-in", "", "Expiry duration (e.g. 30d, 1y)")
}

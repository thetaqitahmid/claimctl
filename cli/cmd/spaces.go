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

var spacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "Manage spaces",
}

var listSpacesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all spaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		spaces, err := client.ListSpaces()
		if err != nil {
			return fmt.Errorf("error fetching spaces: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(spaces, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tCREATED")
		for _, s := range spaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Name, s.Description, time.Unix(s.CreatedAt, 0).Format("2006-01-02 15:04"))
		}
		w.Flush()
		return nil
	},
}

var (
	spaceName        string
	spaceDescription string
)

var createSpaceCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new space",
	RunE: func(cmd *cobra.Command, args []string) error {
		if spaceName == "" {
			return fmt.Errorf("--name is required")
		}

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		space, err := client.CreateSpace(spaceName, spaceDescription)
		if err != nil {
			return fmt.Errorf("error creating space: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(space, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Space created: %s (ID: %s)\n", space.Name, space.ID)
		return nil
	},
}

var updateSpaceCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a space",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		space, err := client.UpdateSpace(args[0], spaceName, spaceDescription)
		if err != nil {
			return fmt.Errorf("error updating space: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(space, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Space %s updated successfully\n", space.ID)
		return nil
	},
}

var deleteSpaceCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a space",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		if err := client.DeleteSpace(args[0]); err != nil {
			return fmt.Errorf("error deleting space: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("Space %s deleted successfully\n", args[0])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(spacesCmd)
	spacesCmd.AddCommand(listSpacesCmd)
	spacesCmd.AddCommand(createSpaceCmd)
	spacesCmd.AddCommand(updateSpaceCmd)
	spacesCmd.AddCommand(deleteSpaceCmd)

	createSpaceCmd.Flags().StringVar(&spaceName, "name", "", "Space name")
	createSpaceCmd.Flags().StringVar(&spaceDescription, "desc", "", "Space description")

	updateSpaceCmd.Flags().StringVar(&spaceName, "name", "", "New space name")
	updateSpaceCmd.Flags().StringVar(&spaceDescription, "desc", "", "New space description")
}

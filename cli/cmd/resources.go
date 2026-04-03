package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	filterType      string
	filterLabelExpr string
)

// resourcesCmd represents the resources command
var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Manage resources",
}

// listResourcesCmd represents the list command
var listResourcesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		resources, err := client.GetResources(filterLabelExpr)
		if err != nil {
			return fmt.Errorf("error fetching resources: %w", err)
		}

		var filtered []api.Resource
		for _, r := range resources {
			// Filter by Type
			if filterType != "" && !strings.EqualFold(r.Type, filterType) {
				continue
			}
			filtered = append(filtered, r)
		}

		if viper.GetBool("json") {
			data, err := json.MarshalIndent(filtered, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

		fmt.Fprintln(w, "ID\tNAME\tTYPE\tSTATUS\tHEALTH\tLABELS")

		for _, r := range filtered {
			labels := strings.Join(r.Labels, ",")
			status := r.Status
			if status == "" {
				status = "Unknown"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", r.ID, r.Name, r.Type, status, r.Health, labels)
		}
		w.Flush()
		return nil
	},
}

var (
	createName       string
	createType       string
	createLabels     []string
	createProperties map[string]string
	createFile       string
)

// createResourceCmd
var createResourceCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new resource",
	Long: `Create a new resource.
Use flags for single resource creation:
  claimctl resources create --name "Room A" --type "Meeting Room" --label "projector,whiteboard"

Use --file for bulk creation from a JSON file:
  claimctl resources create --file resources.json

  Format for resources.json:
  [
    {
      "name": "Room A",
      "type": "Meeting Room",
      "labels": ["projector"],
      "properties": {"capacity": 10}
    }
  ]`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		// Bulk Creation
		if createFile != "" {
			data, err := os.ReadFile(createFile)
			if err != nil {
				return fmt.Errorf("error reading file %s: %w", createFile, err)
			}

			var resources []api.CreateResourceRequest
			if err := json.Unmarshal(data, &resources); err != nil {
				return fmt.Errorf("error parsing JSON file: %w", err)
			}

			successCount := 0
			var errorMsgs []string
			for _, r := range resources {
				res, err := client.CreateResource(r)
				if err != nil {
					msg := fmt.Sprintf("Failed to create resource '%s': %v", r.Name, err)
					fmt.Fprintln(os.Stderr, msg)
					errorMsgs = append(errorMsgs, msg)
					continue
				}
				if !viper.GetBool("json") {
					fmt.Printf("Created resource '%s' (ID: %s)\n", res.Name, res.ID)
				}
				successCount++
			}

			if viper.GetBool("json") {
				fmt.Printf("{\"created\": %d, \"total\": %d}\n", successCount, len(resources))
			} else {
				fmt.Printf("Successfully created %d/%d resources.\n", successCount, len(resources))
			}

			if len(errorMsgs) > 0 {
				return fmt.Errorf("bulk creation completed with errors")
			}
			return nil
		}

		// Single Creation
		if createName == "" {
			return fmt.Errorf("--name is required for single resource creation")
		}

		// Convert properties map[string]string to map[string]interface{}
		props := make(map[string]interface{})
		for k, v := range createProperties {
			props[k] = v
		}

		req := api.CreateResourceRequest{
			Name:       createName,
			Type:       createType,
			Labels:     createLabels,
			Properties: props,
		}

		res, err := client.CreateResource(req)
		if err != nil {
			return fmt.Errorf("error creating resource: %w", err)
		}

		if viper.GetBool("json") {
			data, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Resource created successfully: %s (ID: %s)\n", res.Name, res.ID)
		return nil
	},
}

// deleteResourceCmd
var deleteResourceCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a resource",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		// TODO: Add bulk delete via --file if needed, for now supports single ID
		if len(args) == 0 {
			return fmt.Errorf("resource ID or --file required")
		}

		id := args[0]

		if err := client.DeleteResource(id); err != nil {
			return fmt.Errorf("error deleting resource: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("Resource %s deleted successfully\n", id)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resourcesCmd)
	resourcesCmd.AddCommand(listResourcesCmd)
	resourcesCmd.AddCommand(createResourceCmd)
	resourcesCmd.AddCommand(deleteResourceCmd)

	listResourcesCmd.Flags().StringVar(&filterType, "type", "", "Filter resources by type")
	listResourcesCmd.Flags().StringVar(&filterLabelExpr, "label-expr", "", "Filter resources using a boolean label expression (e.g. 'dev AND ubuntu')")

	createResourceCmd.Flags().StringVar(&createName, "name", "", "Resource Name")
	createResourceCmd.Flags().StringVar(&createType, "type", "Generic", "Resource Type")
	createResourceCmd.Flags().StringSliceVar(&createLabels, "label", []string{}, "Resource Labels")
	createResourceCmd.Flags().StringToStringVar(&createProperties, "property", nil, "Resource Properties (key=value)")
	createResourceCmd.Flags().StringVar(&createFile, "file", "", "JSON file for bulk creation")
}

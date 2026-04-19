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

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage access control groups",
}

var listGroupsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		groups, err := client.ListGroups()
		if err != nil {
			return fmt.Errorf("error fetching groups: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(groups, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tCREATED")
		for _, g := range groups {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", g.ID, g.Name, g.Description, time.Unix(g.CreatedAt, 0).Format("2006-01-02 15:04"))
		}
		w.Flush()
		return nil
	},
}

var (
	groupName        string
	groupDescription string
)

var createGroupCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new group",
	RunE: func(cmd *cobra.Command, args []string) error {
		if groupName == "" {
			return fmt.Errorf("--name is required")
		}

		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		group, err := client.CreateGroup(groupName, groupDescription)
		if err != nil {
			return fmt.Errorf("error creating group: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(group, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Group created: %s (ID: %s)\n", group.Name, group.ID)
		return nil
	},
}

var updateGroupCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		group, err := client.UpdateGroup(args[0], groupName, groupDescription)
		if err != nil {
			return fmt.Errorf("error updating group: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(group, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Group %s updated successfully\n", group.ID)
		return nil
	},
}

var deleteGroupCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		if err := client.DeleteGroup(args[0]); err != nil {
			return fmt.Errorf("error deleting group: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("Group %s deleted successfully\n", args[0])
		}
		return nil
	},
}

var listGroupMembersCmd = &cobra.Command{
	Use:   "members [group-id]",
	Short: "List members of a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		members, err := client.ListGroupMembers(args[0])
		if err != nil {
			return fmt.Errorf("error fetching group members: %w", err)
		}

		if viper.GetBool("json") {
			data, _ := json.MarshalIndent(members, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(members) == 0 {
			fmt.Println("No members in this group.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "USER ID\tNAME\tEMAIL\tROLE")
		for _, m := range members {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.UserID, m.Name, m.Email, m.Role)
		}
		w.Flush()
		return nil
	},
}

var addGroupMemberCmd = &cobra.Command{
	Use:   "add-member [group-id] [user-id]",
	Short: "Add a user to a group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		if err := client.AddGroupMember(args[0], args[1]); err != nil {
			return fmt.Errorf("error adding member: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("User %s added to group %s\n", args[1], args[0])
		}
		return nil
	},
}

var removeGroupMemberCmd = &cobra.Command{
	Use:   "remove-member [group-id] [user-id]",
	Short: "Remove a user from a group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient(viper.GetString("url"), viper.GetString("token"), viper.GetBool("netrc"))
		if err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}

		if err := client.RemoveGroupMember(args[0], args[1]); err != nil {
			return fmt.Errorf("error removing member: %w", err)
		}

		if !viper.GetBool("json") {
			fmt.Printf("User %s removed from group %s\n", args[1], args[0])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(groupsCmd)
	groupsCmd.AddCommand(listGroupsCmd)
	groupsCmd.AddCommand(createGroupCmd)
	groupsCmd.AddCommand(updateGroupCmd)
	groupsCmd.AddCommand(deleteGroupCmd)
	groupsCmd.AddCommand(listGroupMembersCmd)
	groupsCmd.AddCommand(addGroupMemberCmd)
	groupsCmd.AddCommand(removeGroupMemberCmd)

	createGroupCmd.Flags().StringVar(&groupName, "name", "", "Group name")
	createGroupCmd.Flags().StringVar(&groupDescription, "desc", "", "Group description")

	updateGroupCmd.Flags().StringVar(&groupName, "name", "", "New group name")
	updateGroupCmd.Flags().StringVar(&groupDescription, "desc", "", "New group description")
}

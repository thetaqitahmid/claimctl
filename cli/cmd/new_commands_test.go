package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// --- spaces ---

func TestSpacesCmd_Structure(t *testing.T) {
	assert.Equal(t, "spaces", spacesCmd.Use)
	assert.Equal(t, "Manage spaces", spacesCmd.Short)

	names := commandNames(spacesCmd)
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "create")
	assert.Contains(t, names, "update")
	assert.Contains(t, names, "delete")
	assert.Len(t, names, 4)
}

func TestSpacesCmd_Flags(t *testing.T) {
	assert.NotNil(t, createSpaceCmd.Flags().Lookup("name"))
	assert.NotNil(t, createSpaceCmd.Flags().Lookup("desc"))
	assert.NotNil(t, updateSpaceCmd.Flags().Lookup("name"))
	assert.NotNil(t, updateSpaceCmd.Flags().Lookup("desc"))
}

func TestSpacesCmd_Args(t *testing.T) {
	assert.NotNil(t, updateSpaceCmd.Args)
	assert.NotNil(t, deleteSpaceCmd.Args)

	assert.NoError(t, updateSpaceCmd.Args(updateSpaceCmd, []string{"some-id"}))
	assert.Error(t, updateSpaceCmd.Args(updateSpaceCmd, []string{}))

	assert.NoError(t, deleteSpaceCmd.Args(deleteSpaceCmd, []string{"some-id"}))
	assert.Error(t, deleteSpaceCmd.Args(deleteSpaceCmd, []string{}))
}

func TestCreateSpaceCmd_RequiresName(t *testing.T) {
	spaceName = ""
	err := createSpaceCmd.RunE(createSpaceCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

// --- groups ---

func TestGroupsCmd_Structure(t *testing.T) {
	assert.Equal(t, "groups", groupsCmd.Use)
	assert.Equal(t, "Manage access control groups", groupsCmd.Short)

	names := commandNames(groupsCmd)
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "create")
	assert.Contains(t, names, "update")
	assert.Contains(t, names, "delete")
	assert.Contains(t, names, "members")
	assert.Contains(t, names, "add-member")
	assert.Contains(t, names, "remove-member")
	assert.Len(t, names, 7)
}

func TestGroupsCmd_Flags(t *testing.T) {
	assert.NotNil(t, createGroupCmd.Flags().Lookup("name"))
	assert.NotNil(t, createGroupCmd.Flags().Lookup("desc"))
	assert.NotNil(t, updateGroupCmd.Flags().Lookup("name"))
	assert.NotNil(t, updateGroupCmd.Flags().Lookup("desc"))
}

func TestGroupsCmd_Args(t *testing.T) {
	for _, cmd := range []*cobra.Command{
		updateGroupCmd, deleteGroupCmd, listGroupMembersCmd,
	} {
		assert.NotNil(t, cmd.Args)
		assert.NoError(t, cmd.Args(cmd, []string{"some-id"}))
		assert.Error(t, cmd.Args(cmd, []string{}))
	}

	assert.NoError(t, addGroupMemberCmd.Args(addGroupMemberCmd, []string{"g-id", "u-id"}))
	assert.Error(t, addGroupMemberCmd.Args(addGroupMemberCmd, []string{"g-id"}))

	assert.NoError(t, removeGroupMemberCmd.Args(removeGroupMemberCmd, []string{"g-id", "u-id"}))
	assert.Error(t, removeGroupMemberCmd.Args(removeGroupMemberCmd, []string{"g-id"}))
}

func TestCreateGroupCmd_RequiresName(t *testing.T) {
	groupName = ""
	err := createGroupCmd.RunE(createGroupCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

// --- tokens ---

func TestTokensCmd_Structure(t *testing.T) {
	assert.Equal(t, "tokens", tokensCmd.Use)
	assert.Equal(t, "Manage API tokens", tokensCmd.Short)

	names := commandNames(tokensCmd)
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "generate")
	assert.Contains(t, names, "revoke")
	assert.Len(t, names, 3)
}

func TestTokensCmd_Flags(t *testing.T) {
	assert.NotNil(t, generateTokenCmd.Flags().Lookup("name"))
	assert.NotNil(t, generateTokenCmd.Flags().Lookup("expires-in"))
}

func TestRevokeTokenCmd_Args(t *testing.T) {
	assert.NotNil(t, revokeTokenCmd.Args)
	assert.NoError(t, revokeTokenCmd.Args(revokeTokenCmd, []string{"some-id"}))
	assert.Error(t, revokeTokenCmd.Args(revokeTokenCmd, []string{}))
}

func TestGenerateTokenCmd_RequiresName(t *testing.T) {
	tokenName = ""
	err := generateTokenCmd.RunE(generateTokenCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

// --- audit-logs ---

func TestAuditCmd_Structure(t *testing.T) {
	assert.Equal(t, "audit-logs", auditCmd.Use)
	assert.NotNil(t, auditCmd.RunE)
	assert.NotNil(t, auditCmd.Flags().Lookup("limit"))
	assert.NotNil(t, auditCmd.Flags().Lookup("offset"))
}

func TestAuditCmd_FlagDefaults(t *testing.T) {
	limit, err := auditCmd.Flags().GetInt("limit")
	assert.NoError(t, err)
	assert.Equal(t, 50, limit)

	offset, err := auditCmd.Flags().GetInt("offset")
	assert.NoError(t, err)
	assert.Equal(t, 0, offset)
}

// --- resources history & maintenance ---

func TestResourceHistoryCmd_Structure(t *testing.T) {
	assert.NotNil(t, resourceHistoryCmd.RunE)
	assert.NotNil(t, resourceHistoryCmd.Args)
	assert.NoError(t, resourceHistoryCmd.Args(resourceHistoryCmd, []string{"some-id"}))
	assert.Error(t, resourceHistoryCmd.Args(resourceHistoryCmd, []string{}))
}

func TestMaintenanceCmd_Structure(t *testing.T) {
	assert.Equal(t, "maintenance", maintenanceCmd.Use)

	names := commandNames(maintenanceCmd)
	assert.Contains(t, names, "enable")
	assert.Contains(t, names, "disable")
	assert.Contains(t, names, "history")
	assert.Len(t, names, 3)
}

func TestMaintenanceCmd_Args(t *testing.T) {
	for _, cmd := range []*cobra.Command{
		maintenanceEnableCmd, maintenanceDisableCmd, maintenanceHistoryCmd,
	} {
		assert.NotNil(t, cmd.Args)
		assert.NoError(t, cmd.Args(cmd, []string{"some-id"}))
		assert.Error(t, cmd.Args(cmd, []string{}))
	}
}

func TestMaintenanceEnableCmd_Flags(t *testing.T) {
	assert.NotNil(t, maintenanceEnableCmd.Flags().Lookup("reason"))
}

// --- reservations history ---

func TestReservationsHistoryCmd_Structure(t *testing.T) {
	assert.NotNil(t, reservationsHistoryCmd.RunE)
	assert.Nil(t, reservationsHistoryCmd.Args) // no args required
}

// --- webhooks update ---

func TestWebhooksUpdateCmd_Structure(t *testing.T) {
	assert.NotNil(t, updateWebhookCmd.RunE)
	assert.NotNil(t, updateWebhookCmd.Args)
	assert.NoError(t, updateWebhookCmd.Args(updateWebhookCmd, []string{"some-id"}))
	assert.Error(t, updateWebhookCmd.Args(updateWebhookCmd, []string{}))
}

func TestWebhooksUpdateCmd_Flags(t *testing.T) {
	flags := updateWebhookCmd.Flags()
	assert.NotNil(t, flags.Lookup("name"))
	assert.NotNil(t, flags.Lookup("url"))
	assert.NotNil(t, flags.Lookup("method"))
	assert.NotNil(t, flags.Lookup("header"))
	assert.NotNil(t, flags.Lookup("template"))
	assert.NotNil(t, flags.Lookup("desc"))
}

// --- registration on root ---

func TestNewCommandsRegisteredOnRoot(t *testing.T) {
	names := commandNames(rootCmd)
	assert.Contains(t, names, "spaces")
	assert.Contains(t, names, "groups")
	assert.Contains(t, names, "tokens")
	assert.Contains(t, names, "audit-logs")
}

// --- helpers ---

func commandNames(parent *cobra.Command) []string {
	cmds := parent.Commands()
	names := make([]string, len(cmds))
	for i, c := range cmds {
		parts := strings.Split(c.Use, " ")
		names[i] = parts[0]
	}
	return names
}

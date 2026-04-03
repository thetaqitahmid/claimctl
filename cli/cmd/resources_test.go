package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListResourcesCmd_Structure(t *testing.T) {
	// Test command structure
	assert.Equal(t, "list", listResourcesCmd.Use)
	assert.Equal(t, "List all resources", listResourcesCmd.Short)
	assert.NotNil(t, listResourcesCmd.RunE)
}

func TestCreateResourceCmd_Structure(t *testing.T) {
	// Test command structure
	assert.Equal(t, "create", createResourceCmd.Use)
	assert.Equal(t, "Create a new resource", createResourceCmd.Short)
	assert.NotNil(t, createResourceCmd.RunE)
	assert.NotEmpty(t, createResourceCmd.Long)
}

func TestDeleteResourceCmd_Structure(t *testing.T) {
	// Test command structure
	assert.Equal(t, "delete [id]", deleteResourceCmd.Use)
	assert.Equal(t, "Delete a resource", deleteResourceCmd.Short)
	assert.NotNil(t, deleteResourceCmd.RunE)

	// Test argument validation - check that it has args validation
	assert.NotNil(t, deleteResourceCmd.Args)
}

func TestUpdateResourceCmd_Structure(t *testing.T) {
	// Test command structure
	assert.Equal(t, "update [id]", updateResourceCmd.Use)
	assert.Equal(t, "Update a resource and merge its properties", updateResourceCmd.Short)
	assert.NotNil(t, updateResourceCmd.RunE)

	// Test argument validation
	assert.NotNil(t, updateResourceCmd.Args)
}

func TestResourcesCommand_Structure(t *testing.T) {
	// Test that resources command has proper subcommands
	assert.NotNil(t, resourcesCmd)
	assert.Equal(t, "resources", resourcesCmd.Use)
	assert.Equal(t, "Manage resources", resourcesCmd.Short)

	// Check subcommands
	subcommands := resourcesCmd.Commands()
	assert.Len(t, subcommands, 4, "Should have exactly 4 subcommands")

	// Extract just the command name (before any arguments)
	commandNames := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		parts := strings.Split(cmd.Use, " ")
		commandNames[i] = parts[0]
	}
	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "create")
	assert.Contains(t, commandNames, "update")
	assert.Contains(t, commandNames, "delete")
}

func TestResourceCommandFlags(t *testing.T) {
	// Test list command flags
	listFlags := listResourcesCmd.Flags()
	assert.NotNil(t, listFlags.Lookup("type"))
	assert.NotNil(t, listFlags.Lookup("label-expr"))

	// Test create command flags
	createFlags := createResourceCmd.Flags()
	assert.NotNil(t, createFlags.Lookup("name"))
	assert.NotNil(t, createFlags.Lookup("type"))
	assert.NotNil(t, createFlags.Lookup("label"))
	assert.NotNil(t, createFlags.Lookup("property"))
	assert.NotNil(t, createFlags.Lookup("file"))

	// Test delete command flags (should not have specific flags, just args)
	deleteFlags := deleteResourceCmd.Flags()
	assert.NotNil(t, deleteFlags)

	// Test update command flags
	updateFlags := updateResourceCmd.Flags()
	assert.NotNil(t, updateFlags.Lookup("name"))
	assert.NotNil(t, updateFlags.Lookup("type"))
	assert.NotNil(t, updateFlags.Lookup("label"))
	assert.NotNil(t, updateFlags.Lookup("property"))
}

func TestResourceCommandFlagDefaults(t *testing.T) {
	// Test that flags have correct default values

	// List command defaults
	assert.Equal(t, "", filterType)
	assert.Equal(t, "", filterLabelExpr)

	// Create command defaults
	assert.Equal(t, "", createName)
	assert.Equal(t, "Generic", createType) // Default type should be "Generic"
	assert.Equal(t, []string{}, createLabels)
	assert.Equal(t, "", createFile)
	assert.Nil(t, createProperties)

	// Update command defaults
	assert.Equal(t, "", updateName)
	assert.Equal(t, "", updateType)
	assert.Equal(t, []string{}, updateLabels)
	assert.Nil(t, updateProperties)
}

func TestResourceCommandFilterLogic(t *testing.T) {
	// Test the filter logic used in listResourcesCmd
	// We can't easily test the full command without mocking API calls,
	// but we can test the flag parsing behavior

	// Setup test flags
	testFilterType := "Conference Room"
	testFilterLabel := "projector"

	// Test flag setting
	err := listResourcesCmd.Flags().Set("type", testFilterType)
	assert.NoError(t, err)

	err = listResourcesCmd.Flags().Set("label-expr", testFilterLabel)
	assert.NoError(t, err)

	assert.Equal(t, testFilterType, filterType)
	assert.Equal(t, testFilterLabel, filterLabelExpr)
}

func TestCreateResourceFlagParsing(t *testing.T) {
	// Test create resource flag parsing
	testName := "Test Resource"
	testType := "Test Type"
	testLabels := []string{"label1", "label2"}
	testProperties := map[string]string{"key": "value"}
	testFile := "/path/to/file.json"

	// Test flag setting
	err := createResourceCmd.Flags().Set("name", testName)
	assert.NoError(t, err)

	err = createResourceCmd.Flags().Set("type", testType)
	assert.NoError(t, err)

	err = createResourceCmd.Flags().Set("label", strings.Join(testLabels, ","))
	assert.NoError(t, err)

	err = createResourceCmd.Flags().Set("property", "key=value")
	assert.NoError(t, err)

	err = createResourceCmd.Flags().Set("file", testFile)
	assert.NoError(t, err)

	assert.Equal(t, testName, createName)
	assert.Equal(t, testType, createType)
	assert.Equal(t, testLabels, createLabels)
	assert.Equal(t, testProperties, createProperties)
	assert.Equal(t, testFile, createFile)
}

func TestDeleteResourceArgumentValidation(t *testing.T) {
	// Test that delete resource command validates arguments correctly
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Valid single ID",
			args:        []string{"550e8400-e29b-41d4-a716-446655440000"},
			expectError: false,
		},
		{
			name:        "No arguments",
			args:        []string{},
			expectError: false, // MaximumNArgs(1) allows 0 args
		},
		{
			name:        "Multiple arguments",
			args:        []string{"550e8400-e29b-41d4-a716-446655440000", "660f9511-f30c-52e5-b827-557766551111"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deleteResourceCmd.Args(deleteResourceCmd, tt.args)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourcesCommandInitialization(t *testing.T) {
	// Test that commands are properly initialized
	require.NotNil(t, resourcesCmd)
	require.NotNil(t, listResourcesCmd)
	require.NotNil(t, createResourceCmd)
	require.NotNil(t, updateResourceCmd)
	require.NotNil(t, deleteResourceCmd)

	// Test that commands are added to root
	rootCommands := rootCmd.Commands()
	var hasResourcesCommand bool
	for _, cmd := range rootCommands {
		if cmd.Use == "resources" {
			hasResourcesCommand = true
			break
		}
	}
	assert.True(t, hasResourcesCommand, "Resources command should be added to root")

	// Test that subcommands are added to resources
	resourceCommands := resourcesCmd.Commands()
	assert.Len(t, resourceCommands, 4)
}

func TestCommandExecutionSetup(t *testing.T) {
	// Test that commands are set up correctly with viper integration
	// Reset viper
	viper.Reset()
	viper.Set("url", "http://test.example.com")
	viper.Set("token", "test-token")
	viper.Set("json", false)

	// Test that viper values are accessible
	assert.Equal(t, "http://test.example.com", viper.GetString("url"))
	assert.Equal(t, "test-token", viper.GetString("token"))
	assert.Equal(t, false, viper.GetBool("json"))
}

func TestCommandHelpMessages(t *testing.T) {
	// Test that help messages are properly set
	assert.NotEmpty(t, createResourceCmd.Long)
	assert.Contains(t, createResourceCmd.Long, "Create a new resource")
	assert.Contains(t, createResourceCmd.Long, "Use flags for single resource creation")
	assert.Contains(t, createResourceCmd.Long, "Use --file for bulk creation")
}

func TestFlagConsistency(t *testing.T) {
	// Test that flag names are consistent across commands

	// Both list and create should have type flag
	listTypeFlag := listResourcesCmd.Flags().Lookup("type")
	createTypeFlag := createResourceCmd.Flags().Lookup("type")

	assert.NotNil(t, listTypeFlag)
	assert.NotNil(t, createTypeFlag)

	// Both list and create should have label flag
	listLabelFlag := listResourcesCmd.Flags().Lookup("label-expr")
	createLabelFlag := createResourceCmd.Flags().Lookup("label")

	assert.NotNil(t, listLabelFlag)
	assert.NotNil(t, createLabelFlag)

	// Type flag should be string
	assert.Equal(t, "string", listTypeFlag.Value.Type())
	assert.Equal(t, "string", createTypeFlag.Value.Type())

	// Label flag should be string slice
	assert.Equal(t, "stringSlice", createLabelFlag.Value.Type())
}

func TestResourceCommandContext(t *testing.T) {
	// Test that commands work within Cobra context
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Test that root command doesn't panic with resources subcommand
	rootCmd.SetArgs([]string{"resources", "--help"})

	// This should not panic
	assert.NotPanics(t, func() {
		// We don't execute to avoid actual API calls
		_, _ = rootCmd.ExecuteC()
	})
}

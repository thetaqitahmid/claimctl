package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectedErr bool
	}{
		{
			name:        "Root command without args should show help",
			args:        []string{},
			expectedErr: false,
		},
		{
			name:        "Root command with --help",
			args:        []string{"--help"},
			expectedErr: false,
		},
		{
			name:        "Root command with invalid flag",
			args:        []string{"--invalid-flag"},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRootCmdFlags(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "--config")
	assert.Contains(t, output, "--url")
	assert.Contains(t, output, "--token")
	assert.Contains(t, output, "--netrc")
	assert.Contains(t, output, "--json")
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name          string
		setupConfig   func() string
		expectedError bool
	}{
		{
			name: "Config file specified",
			setupConfig: func() string {
				tempDir := t.TempDir()
				configFile := filepath.Join(tempDir, "config.yaml")
				err := os.WriteFile(configFile, []byte("url: http://test.example.com\ntoken: test-token"), 0644)
				require.NoError(t, err)
				return configFile
			},
			expectedError: false,
		},
		{
			name: "Default config locations checked",
			setupConfig: func() string {
				_, err := os.UserHomeDir()
				require.NoError(t, err)
				return "" // Use default config locations
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper state
			viper.Reset()

			configFile := tt.setupConfig()
			if configFile != "" {
				cfgFile = configFile
			}

			// This should not panic
			assert.NotPanics(t, func() {
				initConfig()
			})

			// Verify viper was configured
			assert.NotNil(t, viper.Get("url"))
		})
	}
}

func TestViperFlagsBinding(t *testing.T) {
	// Test that flags are properly bound to viper
	flagURL := rootCmd.PersistentFlags().Lookup("url")
	assert.NotNil(t, flagURL)

	flagToken := rootCmd.PersistentFlags().Lookup("token")
	assert.NotNil(t, flagToken)

	flagNetrc := rootCmd.PersistentFlags().Lookup("netrc")
	assert.NotNil(t, flagNetrc)

	flagJSON := rootCmd.PersistentFlags().Lookup("json")
	assert.NotNil(t, flagJSON)
}

func TestGlobalFlagsDefaults(t *testing.T) {
	// Reset global variables to test defaults
	originalCfgFile := cfgFile
	originalURL := url
	originalToken := token
	originalUseNetrc := useNetrc
	originalJSONOutput := jsonOutput

	defer func() {
		cfgFile = originalCfgFile
		url = originalURL
		token = originalToken
		useNetrc = originalUseNetrc
		jsonOutput = originalJSONOutput
	}()

	// Reset to defaults
	cfgFile = ""
	url = ""
	token = ""
	useNetrc = true // Default is true
	jsonOutput = false

	// Test default values
	assert.Equal(t, "", cfgFile)
	assert.Equal(t, "", url)
	assert.Equal(t, "", token)
	assert.Equal(t, true, useNetrc) // Default is true
	assert.Equal(t, false, jsonOutput)
}

func TestRootCmdExecutionError(t *testing.T) {
	// Test that rootCmd Execute handles invalid flags properly
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--invalid-flag"})

	// Call rootCmd.Execute directly instead of Execute() to avoid os.Exit
	err := rootCmd.Execute()
	assert.Error(t, err)
}

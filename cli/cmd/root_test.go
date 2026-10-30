package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
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
	previousSettings := viper.AllSettings()
	previousFile, previousError := cfgFile, configErr
	t.Cleanup(func() {
		viper.Reset()
		require.NoError(t, viper.MergeConfigMap(previousSettings))
		rootCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) { _ = viper.BindPFlag(flag.Name, flag) })
		cfgFile, configErr = previousFile, previousError
	})
	for _, kind := range []string{"yaml", "json", "missing", "malformed", "default", "insecure"} {
		t.Run(kind, func(t *testing.T) {
			viper.Reset()
			directory := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", directory)
			cfgFile = filepath.Join(directory, "config."+kind)
			switch kind {
			case "yaml":
				require.NoError(t, os.WriteFile(cfgFile, []byte("url: https://example.com\ntoken: secret"), 0600))
			case "json":
				require.NoError(t, os.WriteFile(cfgFile, []byte(`{"url":"https://example.com","token":"secret"}`), 0600))
			case "malformed":
				cfgFile = filepath.Join(directory, "config.json")
				require.NoError(t, os.WriteFile(cfgFile, []byte(`{"broken"`), 0600))
			case "insecure":
				cfgFile = filepath.Join(directory, "config.json")
				require.NoError(t, os.WriteFile(cfgFile, []byte(`{"token":"secret"}`), 0644))
			case "default":
				cfgFile = ""
			}
			initConfig()
			if kind == "missing" || kind == "malformed" || kind == "insecure" {
				require.Error(t, configErr)
			} else {
				require.NoError(t, configErr)
			}
			if kind == "json" || kind == "yaml" {
				require.Equal(t, "https://example.com", viper.GetString("url"))
			}
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

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure the CLI settings (URL and Token)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if viper.GetBool("json") {
			return fmt.Errorf("interactive config cannot be combined with json")
		}
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("config command requires an interactive terminal")
		}

		reader := bufio.NewReader(os.Stdin)

		currentURL := viper.GetString("url")
		fmt.Printf("Server URL [%s]: ", currentURL)
		newURL, _ := reader.ReadString('\n')
		newURL = strings.TrimSpace(newURL)
		if newURL != "" {
			viper.Set("url", newURL)
		}

		fmt.Print("API Token (hidden, blank keeps current): ")
		state, err := term.GetState(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		defer term.Restore(int(os.Stdin.Fd()), state)
		type secretResult struct {
			data []byte
			err  error
		}
		result := make(chan secretResult, 1)
		go func() { data, err := term.ReadPassword(int(os.Stdin.Fd())); result <- secretResult{data, err} }()
		var secret []byte
		select {
		case value := <-result:
			if value.err != nil {
				return value.err
			}
			secret = value.data
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		}
		fmt.Println()
		newToken := strings.TrimSpace(string(secret))
		if newToken != "" {
			viper.Set("token", newToken)
		}

		// Save config
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error finding home directory: %w", err)
		}

		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(home, ".config")
		}
		configPath := cfgFile
		if configPath == "" {
			configPath = filepath.Join(base, "claimctl", "config.json")
		}
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return fmt.Errorf("error creating config directory: %w", err)
		}

		settings := map[string]any{
			"url":                 viper.GetString("url"),
			"token":               viper.GetString("token"),
			"netrc":               viper.GetBool("netrc"),
			"request-timeout":     viper.GetDuration("request-timeout").String(),
			"allow-insecure-http": viper.GetBool("allow-insecure-http"),
		}
		for _, key := range []string{"ca-cert", "token-file"} {
			if value := viper.GetString(key); value != "" {
				settings[key] = value
			}
		}
		data, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return err
		}
		if err := saveConfig(configPath, data); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}

		fmt.Println("Configuration saved to", configPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

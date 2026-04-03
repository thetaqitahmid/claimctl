package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure the CLI settings (URL and Token)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fileInfo, _ := os.Stdin.Stat()
		if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
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

		currentToken := viper.GetString("token")
		maskedToken := ""
		if len(currentToken) > 4 {
			maskedToken = currentToken[:4] + "..."
		}
		fmt.Printf("API Token [%s]: ", maskedToken)
		newToken, _ := reader.ReadString('\n')
		newToken = strings.TrimSpace(newToken)
		if newToken != "" {
			viper.Set("token", newToken)
		}

		// Save config
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error finding home directory: %w", err)
		}

		configDir := filepath.Join(home, ".config", "claimctl")
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return fmt.Errorf("error creating config directory: %w", err)
		}

		configPath := filepath.Join(configDir, "config.json")
		if err := viper.WriteConfigAs(configPath); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}

		fmt.Println("Configuration saved to", configPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

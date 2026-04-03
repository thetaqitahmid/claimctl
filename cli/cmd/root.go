package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	url        string
	token      string
	useNetrc   bool
	jsonOutput bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "claimctl",
	Short: "CLI wrapper for claimctl",
	Long: `A CLI tool to interact with the claimctl application.
You can list resources and create reservations directly from your terminal.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(GetExitCode(err))
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/claimctl/config.json)")
	rootCmd.PersistentFlags().StringVar(&url, "url", "", "Server URL")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "API Token")
	rootCmd.PersistentFlags().BoolVar(&useNetrc, "netrc", true, "Use .netrc for authentication")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("netrc", rootCmd.PersistentFlags().Lookup("netrc"))
	viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name "config" inside ".config/claimctl" directory.
		viper.AddConfigPath(filepath.Join(home, ".config", "claimctl"))
		viper.SetConfigName("config")
		viper.SetConfigType("json")
	}

	// read in environment variables that match
	viper.SetEnvPrefix("claimctl")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	viper.ReadInConfig()
}

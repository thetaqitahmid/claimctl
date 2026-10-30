package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	url        string
	token      string
	useNetrc   bool
	jsonOutput bool
	configErr  error
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "claimctl",
	Short: "CLI wrapper for claimctl",
	Long: `A CLI tool to interact with the claimctl application.
You can list resources and create reservations directly from your terminal.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd == configCmd && errors.Is(configErr, os.ErrNotExist) {
			return nil
		}
		return configErr
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	ctx, cancel := context.WithCancelCause(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	defer cancel(nil)
	go func() {
		select {
		case sig := <-signals:
			cancel(&CLIError{Message: "interrupted", ExitCode: signalExitCode(sig), Err: context.Canceled})
		case <-ctx.Done():
		}
	}()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		if viper.GetBool("json") {
			_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"error": err.Error(), "exitCode": GetExitCode(err)})
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
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
		kind := strings.TrimPrefix(filepath.Ext(cfgFile), ".")
		if kind == "" {
			kind = "json"
		}
		viper.SetConfigType(kind)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			configErr = err
			return
		}

		// Search config in home directory with name "config" inside ".config/claimctl" directory.
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(home, ".config")
		}
		viper.AddConfigPath(filepath.Join(base, "claimctl"))
		viper.SetConfigName("config")
		viper.SetConfigType("json")
	}

	// read in environment variables that match
	viper.SetEnvPrefix("claimctl")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	configErr = nil
	if err := viper.ReadInConfig(); err != nil {
		var missing viper.ConfigFileNotFoundError
		if cfgFile != "" || !errors.As(err, &missing) {
			configErr = fmt.Errorf("read config: %w", err)
		}
		return
	}
	if viper.InConfig("token") {
		info, err := os.Stat(viper.ConfigFileUsed())
		if err != nil {
			configErr = err
			return
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			configErr = fmt.Errorf("config containing credentials requires owner-only permissions")
		}
	}
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("claimctl CLI version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

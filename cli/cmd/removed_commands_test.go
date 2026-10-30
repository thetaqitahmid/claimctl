package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootHasNoApplicationBackupCommands(t *testing.T) {
	for _, command := range rootCmd.Commands() {
		require.NotEqual(t, "backup", command.Name())
		require.NotEqual(t, "restore", command.Name())
	}
}

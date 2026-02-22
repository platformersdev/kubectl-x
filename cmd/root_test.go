package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	require.NotNil(t, rootCmd)
	assert.Equal(t, "kubectl x", rootCmd.Use)
	assert.True(t, rootCmd.TraverseChildren)
}

func TestRootCmdHasSubcommands(t *testing.T) {
	expected := []string{"version", "get", "logs", "top", "wait", "events", "api-resources", "api-versions"}
	registered := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		registered[cmd.Use] = true
	}
	for _, name := range expected {
		assert.True(t, registered[name], "expected subcommand %q to be registered on rootCmd", name)
	}
}

func TestRootCmdFlags(t *testing.T) {
	batchFlag := rootCmd.PersistentFlags().Lookup("batch-size")
	require.NotNil(t, batchFlag)
	assert.Equal(t, "b", batchFlag.Shorthand)
	assert.Equal(t, "25", batchFlag.DefValue)

	filterFlag := rootCmd.PersistentFlags().Lookup("filter")
	require.NotNil(t, filterFlag)
}

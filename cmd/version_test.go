package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	require.NotNil(t, versionCmd)
	assert.Equal(t, "version", versionCmd.Use)
	assert.True(t, versionCmd.DisableFlagParsing)
}

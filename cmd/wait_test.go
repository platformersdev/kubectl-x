package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitCmd(t *testing.T) {
	require.NotNil(t, waitCmd)
	assert.Equal(t, "wait", waitCmd.Use)
	assert.True(t, waitCmd.DisableFlagParsing)
}

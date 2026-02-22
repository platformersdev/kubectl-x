package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopCmd(t *testing.T) {
	require.NotNil(t, topCmd)
	assert.Equal(t, "top", topCmd.Use)
	assert.True(t, topCmd.DisableFlagParsing)
}

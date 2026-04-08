package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthCmd(t *testing.T) {
	require.NotNil(t, authCmd)
	assert.Equal(t, "auth", authCmd.Use)
	assert.True(t, authCmd.DisableFlagParsing)
}

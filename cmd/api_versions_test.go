package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiVersionsCmd(t *testing.T) {
	require.NotNil(t, apiVersionsCmd)
	assert.Equal(t, "api-versions", apiVersionsCmd.Use)
	assert.True(t, apiVersionsCmd.DisableFlagParsing)
}

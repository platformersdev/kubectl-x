package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiResourcesCmd(t *testing.T) {
	require.NotNil(t, apiResourcesCmd)
	assert.Equal(t, "api-resources", apiResourcesCmd.Use)
	assert.True(t, apiResourcesCmd.DisableFlagParsing)
}

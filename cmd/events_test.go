package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventsCmd(t *testing.T) {
	require.NotNil(t, eventsCmd)
	assert.Equal(t, "events", eventsCmd.Use)
	assert.True(t, eventsCmd.DisableFlagParsing)
}

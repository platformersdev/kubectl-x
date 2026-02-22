package cmd

import (
	"testing"
)

func TestEventsCmd(t *testing.T) {
	if eventsCmd == nil {
		t.Fatal("eventsCmd should not be nil")
	}
	if eventsCmd.Use != "events" {
		t.Errorf("eventsCmd.Use = %q, want %q", eventsCmd.Use, "events")
	}
	if !eventsCmd.DisableFlagParsing {
		t.Error("eventsCmd should have DisableFlagParsing enabled")
	}
}

package cmd

import (
	"testing"
)

func TestWaitCmd(t *testing.T) {
	if waitCmd == nil {
		t.Fatal("waitCmd should not be nil")
	}
	if waitCmd.Use != "wait" {
		t.Errorf("waitCmd.Use = %q, want %q", waitCmd.Use, "wait")
	}
	if !waitCmd.DisableFlagParsing {
		t.Error("waitCmd should have DisableFlagParsing enabled")
	}
}

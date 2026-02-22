package cmd

import (
	"testing"
)

func TestTopCmd(t *testing.T) {
	if topCmd == nil {
		t.Fatal("topCmd should not be nil")
	}
	if topCmd.Use != "top" {
		t.Errorf("topCmd.Use = %q, want %q", topCmd.Use, "top")
	}
	if !topCmd.DisableFlagParsing {
		t.Error("topCmd should have DisableFlagParsing enabled")
	}
}

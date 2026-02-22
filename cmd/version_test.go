package cmd

import (
	"testing"
)

func TestVersionCmd(t *testing.T) {
	if versionCmd == nil {
		t.Fatal("versionCmd should not be nil")
	}
	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want %q", versionCmd.Use, "version")
	}
	if !versionCmd.DisableFlagParsing {
		t.Error("versionCmd should have DisableFlagParsing enabled")
	}
}

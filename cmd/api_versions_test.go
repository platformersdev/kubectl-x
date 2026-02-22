package cmd

import (
	"testing"
)

func TestApiVersionsCmd(t *testing.T) {
	if apiVersionsCmd == nil {
		t.Fatal("apiVersionsCmd should not be nil")
	}
	if apiVersionsCmd.Use != "api-versions" {
		t.Errorf("apiVersionsCmd.Use = %q, want %q", apiVersionsCmd.Use, "api-versions")
	}
	if !apiVersionsCmd.DisableFlagParsing {
		t.Error("apiVersionsCmd should have DisableFlagParsing enabled")
	}
}

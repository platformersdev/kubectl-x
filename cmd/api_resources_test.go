package cmd

import (
	"testing"
)

func TestApiResourcesCmd(t *testing.T) {
	if apiResourcesCmd == nil {
		t.Fatal("apiResourcesCmd should not be nil")
	}
	if apiResourcesCmd.Use != "api-resources" {
		t.Errorf("apiResourcesCmd.Use = %q, want %q", apiResourcesCmd.Use, "api-resources")
	}
	if !apiResourcesCmd.DisableFlagParsing {
		t.Error("apiResourcesCmd should have DisableFlagParsing enabled")
	}
}

package cmd

import (
	"testing"
)

func TestRootCmd(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}
	if rootCmd.Use != "kubectl x" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "kubectl x")
	}
	if !rootCmd.TraverseChildren {
		t.Error("rootCmd should have TraverseChildren enabled")
	}
}

func TestRootCmdHasSubcommands(t *testing.T) {
	expected := []string{"version", "get", "logs", "top", "wait", "events", "api-resources", "api-versions"}
	registered := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		registered[cmd.Use] = true
	}
	for _, name := range expected {
		if !registered[name] {
			t.Errorf("expected subcommand %q to be registered on rootCmd", name)
		}
	}
}

func TestRootCmdFlags(t *testing.T) {
	batchFlag := rootCmd.PersistentFlags().Lookup("batch-size")
	if batchFlag == nil {
		t.Fatal("rootCmd should have a 'batch-size' persistent flag")
	}
	if batchFlag.Shorthand != "b" {
		t.Errorf("batch-size shorthand = %q, want %q", batchFlag.Shorthand, "b")
	}
	if batchFlag.DefValue != "25" {
		t.Errorf("batch-size default = %q, want %q", batchFlag.DefValue, "25")
	}

	filterFlag := rootCmd.PersistentFlags().Lookup("filter")
	if filterFlag == nil {
		t.Fatal("rootCmd should have a 'filter' persistent flag")
	}
}

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

func TestEventsWatchMode(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no watch flag",
			args:     []string{},
			expected: false,
		},
		{
			name:     "short watch flag",
			args:     []string{"-w"},
			expected: true,
		},
		{
			name:     "long watch flag",
			args:     []string{"--watch"},
			expected: true,
		},
		{
			name:     "watch-only flag",
			args:     []string{"--watch-only"},
			expected: true,
		},
		{
			name:     "watch with namespace flag",
			args:     []string{"-n", "default", "-w"},
			expected: true,
		},
		{
			name:     "no watch with other flags",
			args:     []string{"-n", "default", "--for-object", "pod/my-pod"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWatchMode(tt.args)
			if result != tt.expected {
				t.Errorf("isWatchMode(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

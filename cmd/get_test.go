package cmd

import (
	"testing"
)

func TestIsWatchMode(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no watch flag",
			args:     []string{"pods"},
			expected: false,
		},
		{
			name:     "short watch flag",
			args:     []string{"pods", "-w"},
			expected: true,
		},
		{
			name:     "long watch flag",
			args:     []string{"pods", "--watch"},
			expected: true,
		},
		{
			name:     "watch-only flag",
			args:     []string{"pods", "--watch-only"},
			expected: true,
		},
		{
			name:     "watch flag before resource",
			args:     []string{"-w", "pods"},
			expected: true,
		},
		{
			name:     "watch flag with other flags",
			args:     []string{"-n", "default", "-w", "pods"},
			expected: true,
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: false,
		},
		{
			name:     "similar but not watch flag",
			args:     []string{"-ww", "pods"},
			expected: false,
		},
		{
			name:     "output flag is not watch",
			args:     []string{"pods", "-o", "wide"},
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

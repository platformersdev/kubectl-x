package cmd

import (
	"testing"
)

func TestIsFollowMode(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no follow flag",
			args:     []string{"pod-name"},
			expected: false,
		},
		{
			name:     "short follow flag",
			args:     []string{"-f", "pod-name"},
			expected: true,
		},
		{
			name:     "long follow flag",
			args:     []string{"--follow", "pod-name"},
			expected: true,
		},
		{
			name:     "follow flag after pod name",
			args:     []string{"pod-name", "-f"},
			expected: true,
		},
		{
			name:     "follow flag with other flags",
			args:     []string{"-n", "default", "-f", "pod-name"},
			expected: true,
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: false,
		},
		{
			name:     "similar but not follow flag",
			args:     []string{"-ff", "pod-name"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFollowMode(tt.args)
			if result != tt.expected {
				t.Errorf("isFollowMode(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

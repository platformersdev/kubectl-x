package cmd

import (
	"testing"
)

func TestHasFollowFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no follow flag",
			args:     []string{"pod", "test-pod"},
			expected: false,
		},
		{
			name:     "short follow flag -f",
			args:     []string{"pod", "test-pod", "-f"},
			expected: true,
		},
		{
			name:     "long follow flag --follow",
			args:     []string{"pod", "test-pod", "--follow"},
			expected: true,
		},
		{
			name:     "follow flag at start",
			args:     []string{"-f", "pod", "test-pod"},
			expected: true,
		},
		{
			name:     "follow flag at end",
			args:     []string{"pod", "test-pod", "--follow"},
			expected: true,
		},
		{
			name:     "follow flag in middle",
			args:     []string{"pod", "-f", "test-pod"},
			expected: true,
		},
		{
			name:     "multiple flags with follow",
			args:     []string{"pod", "test-pod", "-f", "--tail=100"},
			expected: true,
		},
		{
			name:     "multiple flags without follow",
			args:     []string{"pod", "test-pod", "--tail=100", "--since=1h"},
			expected: false,
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: false,
		},
		{
			name:     "flag that contains f but is not follow",
			args:     []string{"pod", "test-pod", "--field-selector=name=test"},
			expected: false,
		},
		{
			name:     "short flag with value (not follow)",
			args:     []string{"pod", "test-pod", "-f", "some-file"},
			expected: true, // This matches -f flag, which is correct
		},
		{
			name:     "long flag with equals (not follow)",
			args:     []string{"pod", "test-pod", "--follow=false"},
			expected: false, // This doesn't match --follow exactly
		},
		{
			name:     "flag prefix but not exact match",
			args:     []string{"pod", "test-pod", "--follow-logs"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasFollowFlag(tt.args)
			if result != tt.expected {
				t.Errorf("hasFollowFlag(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

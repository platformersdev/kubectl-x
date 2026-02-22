package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetKubeconfigPath(t *testing.T) {
	tests := []struct {
		name          string
		kubeconfigEnv string
		wantExact     string
		wantSuffix    string
	}{
		{
			name:          "with KUBECONFIG env set",
			kubeconfigEnv: "/custom/path/config",
			wantExact:     "/custom/path/config",
		},
		{
			name:       "without KUBECONFIG env",
			wantSuffix: ".kube/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalKubeconfig := os.Getenv("KUBECONFIG")

			if tt.kubeconfigEnv != "" {
				os.Setenv("KUBECONFIG", tt.kubeconfigEnv)
			} else {
				os.Unsetenv("KUBECONFIG")
			}

			defer func() {
				if originalKubeconfig != "" {
					os.Setenv("KUBECONFIG", originalKubeconfig)
				} else {
					os.Unsetenv("KUBECONFIG")
				}
			}()

			result := getKubeconfigPath()

			if tt.wantExact != "" {
				assert.Equal(t, tt.wantExact, result)
			} else {
				assert.True(t, filepath.IsAbs(result), "expected absolute path, got %q", result)
				assert.Equal(t, "config", filepath.Base(result))
				assert.Equal(t, ".kube", filepath.Base(filepath.Dir(result)))
			}
		})
	}
}

func TestFilterContexts(t *testing.T) {
	tests := []struct {
		name      string
		contexts  []string
		patterns  []string
		want      []string
		wantError string
	}{
		{
			name:     "empty patterns returns all contexts",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster"},
			patterns: []string{},
			want:     []string{"prod-cluster", "dev-cluster", "staging-cluster"},
		},
		{
			name:     "single pattern matches substring",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster"},
			patterns: []string{"prod"},
			want:     []string{"prod-cluster"},
		},
		{
			name:     "case-insensitive matching",
			contexts: []string{"Prod-Cluster", "dev-cluster", "PROD-CLUSTER"},
			patterns: []string{"prod"},
			want:     []string{"Prod-Cluster", "PROD-CLUSTER"},
		},
		{
			name:     "multiple patterns with OR logic",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster", "test-cluster"},
			patterns: []string{"prod", "dev"},
			want:     []string{"prod-cluster", "dev-cluster"},
		},
		{
			name:     "regex pattern with start anchor",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-prod-cluster"},
			patterns: []string{"^prod"},
			want:     []string{"prod-cluster"},
		},
		{
			name:     "regex pattern with end anchor",
			contexts: []string{"prod-cluster", "dev-cluster", "cluster-prod"},
			patterns: []string{"prod$"},
			want:     []string{"cluster-prod"},
		},
		{
			name:     "regex pattern with alternation",
			contexts: []string{"prod-cluster", "production-cluster", "dev-cluster"},
			patterns: []string{"prod(uction)?"},
			want:     []string{"prod-cluster", "production-cluster"},
		},
		{
			name:     "regex pattern with character class",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster"},
			patterns: []string{"[pd]ev"},
			want:     []string{"dev-cluster"},
		},
		{
			name:     "no matches returns empty slice",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster"},
			patterns: []string{"test"},
			want:     []string{},
		},
		{
			name:     "multiple patterns, some match",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster"},
			patterns: []string{"test", "prod"},
			want:     []string{"prod-cluster"},
		},
		{
			name:     "pattern matches multiple contexts",
			contexts: []string{"prod-cluster-1", "prod-cluster-2", "dev-cluster"},
			patterns: []string{"prod"},
			want:     []string{"prod-cluster-1", "prod-cluster-2"},
		},
		{
			name:     "empty contexts list",
			contexts: []string{},
			patterns: []string{"prod"},
			want:     []string{},
		},
		{
			name:      "invalid regex pattern",
			contexts:  []string{"prod-cluster", "dev-cluster"},
			patterns:  []string{"[invalid"},
			wantError: "invalid regex pattern",
		},
		{
			name:      "invalid regex pattern in multiple patterns",
			contexts:  []string{"prod-cluster", "dev-cluster"},
			patterns:  []string{"prod", "[invalid", "dev"},
			wantError: "invalid regex pattern",
		},
		{
			name:     "complex regex pattern",
			contexts: []string{"prod-cluster-us-east-1", "prod-cluster-us-west-2", "dev-cluster-us-east-1"},
			patterns: []string{"prod.*us-east"},
			want:     []string{"prod-cluster-us-east-1"},
		},
		{
			name:     "pattern with word boundary",
			contexts: []string{"prod-cluster", "production-cluster", "dev-cluster"},
			patterns: []string{"\\bprod\\b"},
			want:     []string{"prod-cluster"},
		},
		{
			name:     "pattern with quantifier",
			contexts: []string{"prod-cluster", "prodd-cluster", "dev-cluster"},
			patterns: []string{"prod+"},
			want:     []string{"prod-cluster", "prodd-cluster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterContexts(tt.contexts, tt.patterns)

			if tt.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
				return
			}

			require.NoError(t, err)

			if len(tt.want) == 0 {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestExcludeContexts(t *testing.T) {
	tests := []struct {
		name      string
		contexts  []string
		patterns  []string
		want      []string
		wantError string
	}{
		{
			name:     "empty patterns returns all contexts",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster"},
			patterns: []string{},
			want:     []string{"prod-cluster", "dev-cluster", "staging-cluster"},
		},
		{
			name:     "single pattern excludes matching",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster"},
			patterns: []string{"prod"},
			want:     []string{"dev-cluster", "staging-cluster"},
		},
		{
			name:     "case-insensitive exclusion",
			contexts: []string{"Prod-Cluster", "dev-cluster", "PROD-CLUSTER"},
			patterns: []string{"prod"},
			want:     []string{"dev-cluster"},
		},
		{
			name:     "multiple patterns with OR logic",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster", "test-cluster"},
			patterns: []string{"prod", "dev"},
			want:     []string{"staging-cluster", "test-cluster"},
		},
		{
			name:     "regex pattern with start anchor",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-prod-cluster"},
			patterns: []string{"^prod"},
			want:     []string{"dev-cluster", "staging-prod-cluster"},
		},
		{
			name:     "no matches excludes nothing",
			contexts: []string{"prod-cluster", "dev-cluster", "staging-cluster"},
			patterns: []string{"test"},
			want:     []string{"prod-cluster", "dev-cluster", "staging-cluster"},
		},
		{
			name:     "all contexts excluded",
			contexts: []string{"prod-cluster", "dev-cluster"},
			patterns: []string{"cluster"},
			want:     []string{},
		},
		{
			name:     "empty contexts list",
			contexts: []string{},
			patterns: []string{"prod"},
			want:     []string{},
		},
		{
			name:      "invalid regex pattern",
			contexts:  []string{"prod-cluster", "dev-cluster"},
			patterns:  []string{"[invalid"},
			wantError: "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := excludeContexts(tt.contexts, tt.patterns)

			if tt.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
				return
			}

			require.NoError(t, err)

			if len(tt.want) == 0 {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestFilterAndExcludeCombined(t *testing.T) {
	contexts := []string{"prod-us-east", "prod-us-west", "dev-us-east", "dev-us-west", "staging-eu"}

	included, err := filterContexts(contexts, []string{"prod", "dev"})
	require.NoError(t, err)
	assert.Equal(t, []string{"prod-us-east", "prod-us-west", "dev-us-east", "dev-us-west"}, included)

	result, err := excludeContexts(included, []string{"us-west"})
	require.NoError(t, err)
	assert.Equal(t, []string{"prod-us-east", "dev-us-east"}, result)
}

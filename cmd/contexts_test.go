package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGetKubeconfigPath(t *testing.T) {
	tests := []struct {
		name           string
		kubeconfigEnv  string
		expectedPrefix string
		expectedSuffix string
	}{
		{
			name:           "with KUBECONFIG env set",
			kubeconfigEnv:  "/custom/path/config",
			expectedPrefix: "/custom/path/config",
			expectedSuffix: "",
		},
		{
			name:           "without KUBECONFIG env",
			kubeconfigEnv:  "",
			expectedSuffix: ".kube/config",
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

			if tt.expectedPrefix != "" {
				if result != tt.expectedPrefix {
					t.Errorf("getKubeconfigPath() = %q, want %q", result, tt.expectedPrefix)
				}
			} else {
				if !filepath.IsAbs(result) {
					t.Errorf("getKubeconfigPath() = %q, want absolute path", result)
				}
				if filepath.Base(result) != "config" {
					dir := filepath.Dir(result)
					if filepath.Base(dir) != ".kube" {
						t.Errorf("getKubeconfigPath() = %q, want path ending in .kube/config", result)
					}
				}
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
		wantError bool
		errorMsg  string
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
			wantError: true,
			errorMsg:  "invalid regex pattern",
		},
		{
			name:      "invalid regex pattern in multiple patterns",
			contexts:  []string{"prod-cluster", "dev-cluster"},
			patterns:  []string{"prod", "[invalid", "dev"},
			wantError: true,
			errorMsg:  "invalid regex pattern",
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

			if tt.wantError {
				if err == nil {
					t.Errorf("filterContexts() expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("filterContexts() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("filterContexts() unexpected error = %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("filterContexts() length = %d, want %d", len(got), len(tt.want))
				return
			}
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterContexts() = %v, want %v", got, tt.want)
			}
		})
	}
}

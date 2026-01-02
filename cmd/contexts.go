package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubeconfig represents the minimal structure needed to read contexts from a kubeconfig file
type Kubeconfig struct {
	Contexts []ContextEntry `yaml:"contexts"`
}

// ContextEntry represents a single context entry in the kubeconfig
type ContextEntry struct {
	Name string `yaml:"name"`
}

func getContexts() ([]string, error) {
	kubeconfigPath := getKubeconfigPath()
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("could not determine kubeconfig path")
	}

	file, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	var config Kubeconfig
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	var contexts []string
	for _, entry := range config.Contexts {
		if entry.Name != "" {
			contexts = append(contexts, entry.Name)
		}
	}

	if len(contexts) == 0 {
		// Fallback to clientcmd if YAML parsing doesn't find contexts
		kubeconfig, err := clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}

		for name := range kubeconfig.Contexts {
			contexts = append(contexts, name)
		}
	}

	if len(contexts) == 0 {
		return nil, fmt.Errorf("no contexts found in kubeconfig")
	}

	// Apply filters if specified
	if len(filterPatterns) > 0 {
		var err error
		contexts, err = filterContexts(contexts, filterPatterns)
		if err != nil {
			return nil, fmt.Errorf("invalid filter pattern: %w", err)
		}
		if len(contexts) == 0 {
			return nil, fmt.Errorf("no contexts match filter patterns: %s", strings.Join(filterPatterns, ", "))
		}
	}

	return contexts, nil
}

// filterContexts filters contexts by regex pattern matching (case-insensitive)
// Multiple patterns are OR'd together - a context matches if it matches any of the patterns
func filterContexts(contexts []string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return contexts, nil
	}

	// Compile regex patterns
	regexes := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		// Add case-insensitive flag (?i) to the pattern
		regex, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		regexes = append(regexes, regex)
	}

	var filtered []string
	for _, ctx := range contexts {
		for _, regex := range regexes {
			if regex.MatchString(ctx) {
				filtered = append(filtered, ctx)
				break // Match found, no need to check other patterns for this context
			}
		}
	}
	return filtered, nil
}

func getKubeconfigPath() string {
	path := os.Getenv("KUBECONFIG")
	if path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s/.kube/config", home)
}

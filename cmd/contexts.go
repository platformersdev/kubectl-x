package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/tools/clientcmd"
)

type Kubeconfig struct {
	Contexts []ContextEntry `yaml:"contexts"`
}

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

	if len(excludePatterns) > 0 {
		var err error
		contexts, err = excludeContexts(contexts, excludePatterns)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern: %w", err)
		}
		if len(contexts) == 0 {
			return nil, fmt.Errorf("all contexts excluded by patterns: %s", strings.Join(excludePatterns, ", "))
		}
	}

	return contexts, nil
}

// Multiple patterns are OR'd together - a context matches if it matches any pattern.
func filterContexts(contexts []string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return contexts, nil
	}

	regexes := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
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

// Multiple patterns are OR'd together - a context is excluded if it matches any pattern.
func excludeContexts(contexts []string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return contexts, nil
	}

	regexes := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		regex, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		regexes = append(regexes, regex)
	}

	var filtered []string
	for _, ctx := range contexts {
		excluded := false
		for _, regex := range regexes {
			if regex.MatchString(ctx) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, ctx)
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

package cmd

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

type outputFormat string

const (
	formatDefault outputFormat = "default"
	formatJSON    outputFormat = "json"
	formatYAML    outputFormat = "yaml"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorGray   = "\033[90m"
)

// Color palette for context names - using bright colors for better visibility
var contextColors = []string{
	"\033[91m", // Bright Red
	"\033[92m", // Bright Green
	"\033[93m", // Bright Yellow
	"\033[94m", // Bright Blue
	"\033[95m", // Bright Magenta
	"\033[96m", // Bright Cyan
	"\033[97m", // Bright White
	"\033[31m", // Red
	"\033[32m", // Green
	"\033[33m", // Yellow
	"\033[34m", // Blue
	"\033[35m", // Magenta
	"\033[36m", // Cyan
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// getContextColor returns a consistent color for a given context name
func getContextColor(context string) string {
	if !isTerminal() {
		return "" // No colors when piping to files
	}

	// Use hash of context name to consistently assign colors
	hash := fnv.New32a()
	hash.Write([]byte(context))
	hashValue := hash.Sum32()

	return contextColors[hashValue%uint32(len(contextColors))]
}

// colorizeContext returns a colored version of the context name
func colorizeContext(context string) string {
	color := getContextColor(context)
	if color == "" {
		return context
	}
	return color + context + colorReset
}

func detectOutputFormat(args []string) outputFormat {
	parseFormat := func(format string) outputFormat {
		format = strings.ToLower(format)
		if format == "json" {
			return formatJSON
		}
		if format == "yaml" {
			return formatYAML
		}
		return formatDefault
	}

	for i, arg := range args {
		// Handle separate flag and value: -o json, --output yaml
		if arg == "-o" || arg == "--output" {
			if i+1 < len(args) {
				if format := parseFormat(args[i+1]); format != formatDefault {
					return format
				}
			}
		}

		// Handle concatenated short flag: -ojson, -oyaml
		if strings.HasPrefix(arg, "-o") && len(arg) > 2 {
			if format := parseFormat(strings.TrimPrefix(arg, "-o")); format != formatDefault {
				return format
			}
		}

		// Handle equals format: --output=json, --output=yaml
		if strings.HasPrefix(arg, "--output=") {
			if format := parseFormat(strings.TrimPrefix(arg, "--output=")); format != formatDefault {
				return format
			}
		}
	}
	return formatDefault
}

func formatOutput(results []contextResult, format outputFormat, subcommand string) error {
	switch format {
	case formatJSON:
		return formatJSONOutput(results, subcommand)
	case formatYAML:
		return formatYAMLOutput(results, subcommand)
	default:
		if subcommand == "version" {
			return formatVersionOutput(results)
		}
		return formatDefaultOutput(results)
	}
}

func formatDefaultOutput(results []contextResult) error {
	// parseColumns splits a line into columns by detecting column boundaries (2+ spaces or tabs)
	// kubectl output uses multiple spaces to separate columns
	columnSeparator := regexp.MustCompile(`[ \t]{2,}`)
	parseColumns := func(line string) []string {
		// Split on 2+ spaces or tabs
		parts := columnSeparator.Split(line, -1)
		var columns []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			// Only include non-empty parts (skip empty strings from multiple consecutive separators)
			if trimmed != "" {
				columns = append(columns, trimmed)
			}
		}
		return columns
	}

	// First pass: collect all contexts and their outputs
	type outputData struct {
		context string
		lines   []string
		columns [][]string // Parsed columns for each line
		err     error
		errMsg  string
	}
	var allOutputs []outputData
	maxContextWidth := len("CONTEXT")

	for _, result := range results {
		if result.err != nil {
			if len(result.context) > maxContextWidth {
				maxContextWidth = len(result.context)
			}
			allOutputs = append(allOutputs, outputData{
				context: result.context,
				err:     result.err,
				errMsg:  result.output,
			})
			continue
		}

		output := strings.TrimSpace(result.output)
		if output == "" {
			continue
		}

		lines := strings.Split(output, "\n")
		if len(lines) == 0 {
			continue
		}

		if len(result.context) > maxContextWidth {
			maxContextWidth = len(result.context)
		}

		// Parse columns for each line
		columns := make([][]string, len(lines))
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				columns[i] = parseColumns(trimmed)
			}
		}

		allOutputs = append(allOutputs, outputData{
			context: result.context,
			lines:   lines,
			columns: columns,
		})
	}

	// Find the header from the first valid output
	var headerColumns []string
	var headerFound bool
	for _, data := range allOutputs {
		if data.err == nil && len(data.columns) > 1 && len(data.columns[0]) > 0 {
			headerColumns = data.columns[0]
			headerFound = true
			break
		}
	}

	// Second pass: find max width for each column position across all outputs
	maxColumnWidths := make(map[int]int)
	if headerFound {
		for i, col := range headerColumns {
			// Ensure we only count non-empty columns and use trimmed length
			trimmed := strings.TrimSpace(col)
			if trimmed != "" && len(trimmed) > maxColumnWidths[i] {
				maxColumnWidths[i] = len(trimmed)
			}
		}
	}

	for _, data := range allOutputs {
		if data.err != nil {
			continue
		}
		startIdx := 0
		if headerFound && len(data.columns) > 1 {
			startIdx = 1 // Skip header line
		}
		for i := startIdx; i < len(data.columns); i++ {
			for j, col := range data.columns[i] {
				// Ensure we only count non-empty columns and use trimmed length
				trimmed := strings.TrimSpace(col)
				if trimmed != "" && len(trimmed) > maxColumnWidths[j] {
					maxColumnWidths[j] = len(trimmed)
				}
			}
		}
	}

	// Helper function to pad and format columns
	formatColumns := func(columns []string) string {
		var parts []string
		for i, col := range columns {
			width := maxColumnWidths[i]
			if width == 0 {
				width = len(col) // Fallback if column not found in max widths
			}
			padded := col + strings.Repeat(" ", width-len(col))
			parts = append(parts, padded)
		}
		// Join with 4 spaces (kubectl standard) and trim trailing spaces
		return strings.TrimRight(strings.Join(parts, "    "), " ")
	}

	// Print header if found
	if headerFound {
		contextPadding := strings.Repeat(" ", maxContextWidth-len("CONTEXT"))
		formattedHeader := formatColumns(headerColumns)
		fmt.Printf("%s%s  %s\n", "CONTEXT", contextPadding, formattedHeader)
	}

	// Print all outputs
	for _, data := range allOutputs {
		if data.err != nil {
			coloredContext := colorizeContext(data.context)
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", coloredContext, data.err)
			if data.errMsg != "" {
				fmt.Fprintf(os.Stderr, "Output: %s\n", data.errMsg)
			}
			continue
		}

		startIdx := 0
		if headerFound && len(data.columns) > 1 {
			startIdx = 1 // Skip header line
		}

		coloredContext := colorizeContext(data.context)
		contextPadding := strings.Repeat(" ", maxContextWidth-len(data.context))

		for i := startIdx; i < len(data.columns); i++ {
			if len(data.columns[i]) == 0 {
				continue
			}
			formattedLine := formatColumns(data.columns[i])
			fmt.Printf("%s%s  %s\n", coloredContext, contextPadding, formattedLine)
		}
	}

	return nil
}

func formatVersionOutput(results []contextResult) error {
	type versionInfo struct {
		clientVersion    string
		kustomizeVersion string
		serverVersion    string
	}

	// Parse version information from results
	versionData := make(map[string]versionInfo)
	var clientVersion, kustomizeVersion string

	// First pass: extract client and kustomize version from first successful result
	for _, result := range results {
		if result.err != nil {
			continue
		}

		output := strings.TrimSpace(result.output)
		if output == "" {
			continue
		}

		// Extract client and kustomize version (same for all contexts)
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if clientVersion == "" && strings.HasPrefix(line, "Client Version:") {
				clientVersion = strings.TrimPrefix(line, "Client Version:")
				clientVersion = strings.TrimSpace(clientVersion)
			}
			if kustomizeVersion == "" && strings.HasPrefix(line, "Kustomize Version:") {
				kustomizeVersion = strings.TrimPrefix(line, "Kustomize Version:")
				kustomizeVersion = strings.TrimSpace(kustomizeVersion)
			}
		}
		// Continue looking if we haven't found both yet
		if clientVersion != "" && kustomizeVersion != "" {
			break
		}
	}

	// Second pass: extract server version for each context
	for _, result := range results {
		if result.err != nil {
			versionData[result.context] = versionInfo{
				serverVersion: "ERROR",
			}
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", result.context, result.err)
			if result.output != "" {
				fmt.Fprintf(os.Stderr, "Output: %s\n", result.output)
			}
			continue
		}

		output := strings.TrimSpace(result.output)
		if output == "" {
			versionData[result.context] = versionInfo{
				serverVersion: "N/A",
			}
			continue
		}

		// Extract server version
		var serverVersion string
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Server Version:") {
				serverVersion = strings.TrimPrefix(line, "Server Version:")
				serverVersion = strings.TrimSpace(serverVersion)
				break
			}
		}

		if serverVersion == "" {
			serverVersion = "N/A"
		}

		versionData[result.context] = versionInfo{
			serverVersion: serverVersion,
		}
	}

	// Print client and kustomize version at the top
	if clientVersion != "" {
		fmt.Printf("Client Version: %s\n", clientVersion)
	}
	if kustomizeVersion != "" {
		fmt.Printf("Kustomize Version: %s\n", kustomizeVersion)
	}
	if clientVersion != "" || kustomizeVersion != "" {
		fmt.Println()
	}

	// Print table header
	fmt.Printf("%-30s  %s\n", "CONTEXT", "SERVER VERSION")
	fmt.Println(strings.Repeat("-", 50))

	// Print table rows
	for _, result := range results {
		info := versionData[result.context]
		coloredContext := colorizeContext(result.context)
		// Calculate padding based on actual context length (without ANSI codes)
		contextLen := len(result.context)
		padding := ""
		if contextLen < 30 {
			padding = strings.Repeat(" ", 30-contextLen)
		}
		fmt.Printf("%s%s  %s\n", coloredContext, padding, info.serverVersion)
	}

	return nil
}

func formatLogsOutput(results []contextResult) error {
	maxContextWidth := 0
	for _, result := range results {
		if len(result.context) > maxContextWidth {
			maxContextWidth = len(result.context)
		}
	}

	for _, result := range results {
		if result.err != nil {
			coloredContext := colorizeContext(result.context)
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", coloredContext, result.err)
			if result.output != "" {
				fmt.Fprintf(os.Stderr, "Output: %s\n", result.output)
			}
			continue
		}

		output := strings.TrimSpace(result.output)
		if output == "" {
			continue
		}

		lines := strings.Split(output, "\n")
		coloredContext := colorizeContext(result.context)
		padding := strings.Repeat(" ", maxContextWidth-len(result.context))

		for _, line := range lines {
			fmt.Printf("%s%s  %s\n", coloredContext, padding, line)
		}
	}

	return nil
}

func formatJSONOutput(results []contextResult, subcommand string) error {
	var allItems []map[string]interface{}

	for _, result := range results {
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", result.context, result.err)
			if result.output != "" {
				// Try to parse error output anyway
				var errorData map[string]interface{}
				if err := json.Unmarshal([]byte(result.output), &errorData); err == nil {
					errorData["context"] = result.context
					errorData["error"] = result.err.Error()
					allItems = append(allItems, errorData)
				}
			}
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result.output), &data); err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Failed to parse JSON: %v\n", result.context, err)
			continue
		}

		// Extract items array if it exists
		if itemsArray, exists := data["items"]; exists {
			items, ok := itemsArray.([]interface{})
			if !ok {
				// Try to convert if it's not the right type
				if itemsSlice, ok := itemsArray.([]interface{}); ok {
					items = itemsSlice
				} else {
					continue
				}
			}

			// Add context metadata to each item
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
						metadata["context"] = result.context
					} else {
						itemMap["metadata"] = map[string]interface{}{
							"context": result.context,
						}
					}
					allItems = append(allItems, itemMap)
				}
			}
		} else {
			// No items array - this might be a single object or non-list response
			// Add context to the root object
			if metadata, ok := data["metadata"].(map[string]interface{}); ok {
				metadata["context"] = result.context
			} else {
				data["context"] = result.context
			}
			allItems = append(allItems, data)
		}
	}

	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "List",
		"items":      allItems,
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func formatYAMLOutput(results []contextResult, subcommand string) error {
	var allItems []map[string]interface{}

	for _, result := range results {
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", result.context, result.err)
			if result.output != "" {
				// Try to parse error output anyway
				var errorData map[string]interface{}
				if err := yaml.Unmarshal([]byte(result.output), &errorData); err == nil {
					errorData["context"] = result.context
					errorData["error"] = result.err.Error()
					allItems = append(allItems, errorData)
				}
			}
			continue
		}

		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(result.output), &data); err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Failed to parse YAML: %v\n", result.context, err)
			continue
		}

		// Extract items array if it exists
		if itemsArray, exists := data["items"]; exists {
			items, ok := itemsArray.([]interface{})
			if !ok {
				// Try to convert if it's not the right type
				if itemsSlice, ok := itemsArray.([]interface{}); ok {
					items = itemsSlice
				} else {
					continue
				}
			}

			// Add context metadata to each item
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
						metadata["context"] = result.context
					} else {
						itemMap["metadata"] = map[string]interface{}{
							"context": result.context,
						}
					}
					allItems = append(allItems, itemMap)
				}
			}
		} else {
			// No items array - this might be a single object or non-list response
			// Add context to the root object
			if metadata, ok := data["metadata"].(map[string]interface{}); ok {
				metadata["context"] = result.context
			} else {
				data["context"] = result.context
			}
			allItems = append(allItems, data)
		}
	}

	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "List",
		"items":      allItems,
	}

	yamlData, err := yaml.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	fmt.Print(string(yamlData))
	return nil
}

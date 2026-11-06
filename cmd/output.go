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
	formatRaw     outputFormat = "raw"
)

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
		if format == "name" ||
			strings.HasPrefix(format, "jsonpath=") ||
			strings.HasPrefix(format, "jsonpath-as-json=") ||
			strings.HasPrefix(format, "jsonpath-file=") ||
			strings.HasPrefix(format, "go-template=") ||
			strings.HasPrefix(format, "go-template-file=") ||
			strings.HasPrefix(format, "custom-columns=") ||
			strings.HasPrefix(format, "custom-columns-file=") {
			return formatRaw
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
	if subcommand == "version" && (format == formatJSON || format == formatYAML) {
		return formatVersionStructuredOutput(results, format)
	}

	switch format {
	case formatJSON:
		return formatJSONOutput(results, subcommand)
	case formatYAML:
		return formatYAMLOutput(results, subcommand)
	case formatRaw:
		return formatRawOutput(results)
	default:
		if subcommand == "version" {
			return formatVersionOutput(results)
		}
		if subcommand == "logs" || subcommand == "api-versions" {
			return formatRawOutput(results)
		}
		return formatDefaultOutput(results)
	}
}

// kubectl output uses multiple spaces to separate columns
var columnSeparator = regexp.MustCompile(`[ \t]{2,}`)

func parseColumns(line string) []string {
	var columns []string
	for _, part := range columnSeparator.Split(strings.TrimSpace(line), -1) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			columns = append(columns, trimmed)
		}
	}
	return columns
}

type tableRow struct {
	context string
	columns []string
}

type table struct {
	header []string
	rows   []tableRow
}

// splitSections splits kubectl output into blank-line separated tables, as
// produced when requesting multiple resource types (e.g. `get no,ns`).
func splitSections(output string) [][]string {
	var sections [][]string
	var current []string
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			if len(current) > 0 {
				sections = append(sections, current)
				current = nil
			}
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		sections = append(sections, current)
	}
	return sections
}

// tableKey identifies tables that should be merged across contexts. The
// resource type prefix (e.g. "node" in "node/foo") distinguishes resource
// types that happen to share the same header.
func tableKey(header []string, firstRow []string) string {
	key := strings.Join(header, "\x00")
	if len(firstRow) > 0 {
		if idx := strings.Index(firstRow[0], "/"); idx > 0 {
			key += "\x00" + firstRow[0][:idx]
		}
	}
	return key
}

// groupTables merges the tables from each context's output so that each
// distinct table is printed once with a single header.
func groupTables(results []contextResult) []*table {
	var tables []*table
	byKey := make(map[string]*table)

	for _, result := range results {
		if result.err != nil {
			continue
		}
		for _, section := range splitSections(result.output) {
			var header []string
			lines := section
			if len(section) > 1 {
				header = parseColumns(section[0])
				lines = section[1:]
			}

			var rows []tableRow
			for _, line := range lines {
				if columns := parseColumns(line); len(columns) > 0 {
					rows = append(rows, tableRow{context: result.context, columns: columns})
				}
			}
			if len(rows) == 0 {
				continue
			}

			key := tableKey(header, rows[0].columns)
			t, ok := byKey[key]
			if !ok {
				t = &table{header: header}
				byKey[key] = t
				tables = append(tables, t)
			}
			t.rows = append(t.rows, rows...)
		}
	}
	return tables
}

func formatColumns(columns []string, widths []int) string {
	parts := make([]string, len(columns))
	for i, col := range columns {
		width := len(col)
		if i < len(widths) && widths[i] > width {
			width = widths[i]
		}
		parts[i] = col + strings.Repeat(" ", width-len(col))
	}
	// Join with 4 spaces (kubectl standard) and trim trailing spaces
	return strings.TrimRight(strings.Join(parts, "    "), " ")
}

func columnWidths(t *table) []int {
	var widths []int
	update := func(columns []string) {
		for i, col := range columns {
			if i >= len(widths) {
				widths = append(widths, 0)
			}
			if len(col) > widths[i] {
				widths[i] = len(col)
			}
		}
	}
	update(t.header)
	for _, row := range t.rows {
		update(row.columns)
	}
	return widths
}

func formatDefaultOutput(results []contextResult) error {
	maxContextWidth := len("CONTEXT")
	for _, result := range results {
		if result.err != nil || strings.TrimSpace(result.output) != "" {
			if len(result.context) > maxContextWidth {
				maxContextWidth = len(result.context)
			}
		}
	}

	for _, result := range results {
		if result.err != nil {
			coloredContext := colorizeContext(result.context)
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", coloredContext, result.err)
			if result.output != "" {
				fmt.Fprintf(os.Stderr, "Output: %s\n", result.output)
			}
		}
	}

	for i, t := range groupTables(results) {
		if i > 0 {
			fmt.Println()
		}
		printTable(t, maxContextWidth)
	}

	return nil
}

// printTable prints a table prefixed with a CONTEXT column of the given width.
func printTable(t *table, contextWidth int) {
	widths := columnWidths(t)

	if len(t.header) > 0 {
		contextPadding := strings.Repeat(" ", contextWidth-len("CONTEXT"))
		fmt.Printf("%s%s  %s\n", "CONTEXT", contextPadding, formatColumns(t.header, widths))
	}

	for _, row := range t.rows {
		contextPadding := strings.Repeat(" ", contextWidth-len(row.context))
		fmt.Printf("%s%s  %s\n", colorizeContext(row.context), contextPadding, formatColumns(row.columns, widths))
	}
}

func formatVersionOutput(results []contextResult) error {
	type versionInfo struct {
		clientVersion    string
		kustomizeVersion string
		serverVersion    string
	}

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
			continue
		}

		output := strings.TrimSpace(result.output)
		if output == "" {
			versionData[result.context] = versionInfo{
				serverVersion: "N/A",
			}
			continue
		}

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

	for _, result := range results {
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", result.context, result.err)
			if result.output != "" {
				fmt.Fprintf(os.Stderr, "Output: %s\n", result.output)
			}
		}
	}

	if clientVersion != "" {
		fmt.Printf("Client Version: %s\n", clientVersion)
	}
	if kustomizeVersion != "" {
		fmt.Printf("Kustomize Version: %s\n", kustomizeVersion)
	}
	if clientVersion != "" || kustomizeVersion != "" {
		fmt.Println()
	}

	versionTable := &table{header: []string{"SERVER VERSION"}}
	contextWidth := len("CONTEXT")
	for _, result := range results {
		if len(result.context) > contextWidth {
			contextWidth = len(result.context)
		}
		versionTable.rows = append(versionTable.rows, tableRow{
			context: result.context,
			columns: []string{versionData[result.context].serverVersion},
		})
	}
	printTable(versionTable, contextWidth)

	return nil
}

func formatRawOutput(results []contextResult) error {
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
		}
	}

	for _, result := range results {
		if result.err != nil {
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

		if itemsArray, exists := data["items"]; exists {
			items, ok := itemsArray.([]interface{})
			if !ok {
				if itemsSlice, ok := itemsArray.([]interface{}); ok {
					items = itemsSlice
				} else {
					continue
				}
			}

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
			// Single object or non-list response
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

		if itemsArray, exists := data["items"]; exists {
			items, ok := itemsArray.([]interface{})
			if !ok {
				if itemsSlice, ok := itemsArray.([]interface{}); ok {
					items = itemsSlice
				} else {
					continue
				}
			}

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
			// Single object or non-list response
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

type versionContextEntry struct {
	Context       string      `json:"context" yaml:"context"`
	ServerVersion interface{} `json:"serverVersion,omitempty" yaml:"serverVersion,omitempty"`
	Error         string      `json:"error,omitempty" yaml:"error,omitempty"`
}

type structuredVersionOutput struct {
	ClientVersion    interface{}           `json:"clientVersion,omitempty" yaml:"clientVersion,omitempty"`
	KustomizeVersion interface{}           `json:"kustomizeVersion,omitempty" yaml:"kustomizeVersion,omitempty"`
	ServerVersions   []versionContextEntry `json:"serverVersions" yaml:"serverVersions"`
}

// parseVersionDocument parses kubectl version JSON or YAML output. Failed
// contexts may have stderr appended to the document, so trailing lines are
// dropped until the remainder parses.
func parseVersionDocument(output string) (map[string]interface{}, bool) {
	isVersionDocument := func(data map[string]interface{}) bool {
		_, hasClient := data["clientVersion"]
		_, hasServer := data["serverVersion"]
		return hasClient || hasServer
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for end := len(lines); end > 0; end-- {
		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(strings.Join(lines[:end], "\n")), &data); err == nil && isVersionDocument(data) {
			return data, true
		}
	}
	return nil, false
}

func formatVersionStructuredOutput(results []contextResult, format outputFormat) error {
	out := structuredVersionOutput{ServerVersions: []versionContextEntry{}}

	for _, result := range results {
		entry := versionContextEntry{Context: result.context}
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", result.context, result.err)
			if result.output != "" {
				fmt.Fprintf(os.Stderr, "Output: %s\n", result.output)
			}
			entry.Error = result.err.Error()
		}

		if data, ok := parseVersionDocument(result.output); ok {
			if out.ClientVersion == nil {
				out.ClientVersion = data["clientVersion"]
			}
			if out.KustomizeVersion == nil {
				out.KustomizeVersion = data["kustomizeVersion"]
			}
			entry.ServerVersion = data["serverVersion"]
		} else if result.err == nil {
			fmt.Fprintf(os.Stderr, "Context %s: Failed to parse output: %s\n", result.context, strings.TrimSpace(result.output))
			entry.Error = "failed to parse output"
		}

		out.ServerVersions = append(out.ServerVersions, entry)
	}

	if format == formatYAML {
		yamlData, err := yaml.Marshal(out)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Print(string(yamlData))
		return nil
	}

	jsonData, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

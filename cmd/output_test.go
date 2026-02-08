package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestDetectOutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected outputFormat
	}{
		{
			name:     "no output flag",
			args:     []string{"pod", "test-pod"},
			expected: formatDefault,
		},
		{
			name:     "json output short flag",
			args:     []string{"pod", "-o", "json"},
			expected: formatJSON,
		},
		{
			name:     "json output long flag",
			args:     []string{"pod", "--output", "json"},
			expected: formatJSON,
		},
		{
			name:     "yaml output short flag",
			args:     []string{"pod", "-o", "yaml"},
			expected: formatYAML,
		},
		{
			name:     "yaml output long flag",
			args:     []string{"pod", "--output", "yaml"},
			expected: formatYAML,
		},
		{
			name:     "case insensitive json",
			args:     []string{"pod", "-o", "JSON"},
			expected: formatJSON,
		},
		{
			name:     "case insensitive yaml",
			args:     []string{"pod", "-o", "YAML"},
			expected: formatYAML,
		},
		{
			name:     "unknown format",
			args:     []string{"pod", "-o", "table"},
			expected: formatDefault,
		},
		{
			name:     "output flag without value",
			args:     []string{"pod", "-o"},
			expected: formatDefault,
		},
		{
			name:     "output flag at end",
			args:     []string{"pod", "--output"},
			expected: formatDefault,
		},
		{
			name:     "concatenated json short flag",
			args:     []string{"pod", "-ojson"},
			expected: formatJSON,
		},
		{
			name:     "concatenated yaml short flag",
			args:     []string{"pod", "-oyaml"},
			expected: formatYAML,
		},
		{
			name:     "equals format json",
			args:     []string{"pod", "--output=json"},
			expected: formatJSON,
		},
		{
			name:     "equals format yaml",
			args:     []string{"pod", "--output=yaml"},
			expected: formatYAML,
		},
		{
			name:     "case insensitive concatenated json",
			args:     []string{"pod", "-oJSON"},
			expected: formatJSON,
		},
		{
			name:     "case insensitive equals format",
			args:     []string{"pod", "--output=YAML"},
			expected: formatYAML,
		},
		{
			name:     "concatenated flag with unknown format",
			args:     []string{"pod", "-otable"},
			expected: formatDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectOutputFormat(tt.args)
			if result != tt.expected {
				t.Errorf("detectOutputFormat(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestFormatDefaultOutput(t *testing.T) {
	tests := []struct {
		name     string
		results  []contextResult
		expected string
	}{
		{
			name: "single context with header",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    STATUS    AGE\npod1    Running   5m",
					err:     nil,
				},
			},
			expected: "CONTEXT  NAME    STATUS     AGE\nctx1     pod1    Running    5m\n",
		},
		{
			name: "multiple contexts with header",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    STATUS    AGE\npod1    Running   5m",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "NAME    STATUS    AGE\npod2    Pending   3m",
					err:     nil,
				},
			},
			expected: "CONTEXT  NAME    STATUS     AGE\nctx1     pod1    Running    5m\nctx2     pod2    Pending    3m\n",
		},
		{
			name: "contexts with different length names",
			results: []contextResult{
				{
					context: "short",
					output:  "NAME    STATUS\npod1    Running",
					err:     nil,
				},
				{
					context: "very-long-context-name",
					output:  "NAME    STATUS\npod2    Pending",
					err:     nil,
				},
			},
			expected: "CONTEXT                 NAME    STATUS\nshort                   pod1    Running\nvery-long-context-name  pod2    Pending\n",
		},
		{
			name: "context with error",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    STATUS\npod1    Running",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "error message",
					err:     fmt.Errorf("connection failed"),
				},
			},
			expected: "CONTEXT  NAME    STATUS\nctx1     pod1    Running\n",
		},
		{
			name: "context with empty output",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    STATUS\npod1    Running",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "",
					err:     nil,
				},
			},
			expected: "CONTEXT  NAME    STATUS\nctx1     pod1    Running\n",
		},
		{
			name: "no header in output",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "pod1    Running",
					err:     nil,
				},
			},
			expected: "ctx1     pod1    Running\n",
		},
		{
			name: "different column widths across contexts",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    STATUS    AGE\npod1    Running   5m",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "NAME         STATUS    AGE\nvery-long-pod-name    Pending   3m",
					err:     nil,
				},
			},
			expected: "CONTEXT  NAME                  STATUS     AGE\nctx1     pod1                  Running    5m\nctx2     very-long-pod-name    Pending    3m\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				os.Stdout = oldStdout
				w.Close()
			}()

			done := make(chan bool)
			go func() {
				io.Copy(&stdout, r)
				done <- true
			}()

			err := formatDefaultOutput(tt.results)
			w.Close()
			<-done

			if err != nil {
				t.Errorf("formatDefaultOutput() error = %v, want nil", err)
			}

			output := stdout.String()
			if output != tt.expected {
				t.Errorf("formatDefaultOutput() output = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestFormatVersionOutput(t *testing.T) {
	tests := []struct {
		name     string
		results  []contextResult
		expected string
	}{
		{
			name: "single context",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "Client Version: v1.34.3\nKustomize Version: v5.7.1\nServer Version: v1.34.0",
					err:     nil,
				},
			},
			expected: "Client Version: v1.34.3\nKustomize Version: v5.7.1\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\n",
		},
		{
			name: "multiple contexts",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "Client Version: v1.34.3\nServer Version: v1.34.0",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "Client Version: v1.34.3\nServer Version: v1.34.0",
					err:     nil,
				},
			},
			expected: "Client Version: v1.34.3\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\nctx2                            v1.34.0\n",
		},
		{
			name: "context with error",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "Client Version: v1.34.3\nServer Version: v1.34.0",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "error message",
					err:     fmt.Errorf("connection failed"),
				},
			},
			expected: "Client Version: v1.34.3\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\nctx2                            ERROR\n",
		},
		{
			name: "context with empty output",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "Client Version: v1.34.3\nServer Version: v1.34.0",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "",
					err:     nil,
				},
			},
			expected: "Client Version: v1.34.3\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\nctx2                            N/A\n",
		},
		{
			name: "output with empty lines",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "Client Version: v1.34.3\n\nServer Version: v1.34.0",
					err:     nil,
				},
			},
			expected: "Client Version: v1.34.3\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				os.Stdout = oldStdout
				w.Close()
			}()

			done := make(chan bool)
			go func() {
				io.Copy(&stdout, r)
				done <- true
			}()

			err := formatVersionOutput(tt.results)
			w.Close()
			<-done

			if err != nil {
				t.Errorf("formatVersionOutput() error = %v, want nil", err)
			}

			output := stdout.String()
			if output != tt.expected {
				t.Errorf("formatVersionOutput() output = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestFormatJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		results  []contextResult
		expected string
	}{
		{
			name: "single context with items array",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"items":[{"metadata":{"name":"pod1"}}]}`,
					err:     nil,
				},
			},
			expected: `{
  "apiVersion": "v1",
  "items": [
    {
      "metadata": {
        "context": "ctx1",
        "name": "pod1"
      }
    }
  ],
  "kind": "List"
}
`,
		},
		{
			name: "multiple contexts with items",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"items":[{"metadata":{"name":"pod1"}}]}`,
					err:     nil,
				},
				{
					context: "ctx2",
					output:  `{"items":[{"metadata":{"name":"pod2"}}]}`,
					err:     nil,
				},
			},
			expected: `{
  "apiVersion": "v1",
  "items": [
    {
      "metadata": {
        "context": "ctx1",
        "name": "pod1"
      }
    },
    {
      "metadata": {
        "context": "ctx2",
        "name": "pod2"
      }
    }
  ],
  "kind": "List"
}
`,
		},
		{
			name: "single object without items",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"metadata":{"name":"pod1"}}`,
					err:     nil,
				},
			},
			expected: `{
  "apiVersion": "v1",
  "items": [
    {
      "metadata": {
        "context": "ctx1",
        "name": "pod1"
      }
    }
  ],
  "kind": "List"
}
`,
		},
		{
			name: "object without metadata",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"name":"pod1"}`,
					err:     nil,
				},
			},
			expected: `{
  "apiVersion": "v1",
  "items": [
    {
      "context": "ctx1",
      "name": "pod1"
    }
  ],
  "kind": "List"
}
`,
		},
		{
			name: "context with error",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"items":[{"metadata":{"name":"pod1"}}]}`,
					err:     nil,
				},
				{
					context: "ctx2",
					output:  `{"error":"connection failed"}`,
					err:     fmt.Errorf("connection failed"),
				},
			},
			expected: `{
  "apiVersion": "v1",
  "items": [
    {
      "metadata": {
        "context": "ctx1",
        "name": "pod1"
      }
    },
    {
      "context": "ctx2",
      "error": "connection failed"
    }
  ],
  "kind": "List"
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				os.Stdout = oldStdout
				w.Close()
			}()

			done := make(chan bool)
			go func() {
				io.Copy(&stdout, r)
				done <- true
			}()

			err := formatJSONOutput(tt.results, "get")
			w.Close()
			<-done

			if err != nil {
				t.Errorf("formatJSONOutput() error = %v, want nil", err)
			}

			output := stdout.String()
			output = strings.TrimSpace(output)
			expected := strings.TrimSpace(tt.expected)
			if output != expected {
				t.Errorf("formatJSONOutput() output = %q, want %q", output, expected)
			}
		})
	}
}

func TestFormatYAMLOutput(t *testing.T) {
	tests := []struct {
		name    string
		results []contextResult
		checkFn func(t *testing.T, output string)
	}{
		{
			name: "single context with items array",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"items":[{"metadata":{"name":"pod1"}}]}`,
					err:     nil,
				},
			},
			checkFn: func(t *testing.T, output string) {
				if !strings.Contains(output, "context: ctx1") {
					t.Errorf("formatYAMLOutput() should contain 'context: ctx1'")
				}
				if !strings.Contains(output, "name: pod1") {
					t.Errorf("formatYAMLOutput() should contain 'name: pod1'")
				}
			},
		},
		{
			name: "multiple contexts",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"items":[{"metadata":{"name":"pod1"}}]}`,
					err:     nil,
				},
				{
					context: "ctx2",
					output:  `{"items":[{"metadata":{"name":"pod2"}}]}`,
					err:     nil,
				},
			},
			checkFn: func(t *testing.T, output string) {
				if !strings.Contains(output, "context: ctx1") {
					t.Errorf("formatYAMLOutput() should contain 'context: ctx1'")
				}
				if !strings.Contains(output, "context: ctx2") {
					t.Errorf("formatYAMLOutput() should contain 'context: ctx2'")
				}
			},
		},
		{
			name: "object without metadata",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"name":"pod1"}`,
					err:     nil,
				},
			},
			checkFn: func(t *testing.T, output string) {
				if !strings.Contains(output, "context: ctx1") {
					t.Errorf("formatYAMLOutput() should contain 'context: ctx1'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				os.Stdout = oldStdout
				w.Close()
			}()

			done := make(chan bool)
			go func() {
				io.Copy(&stdout, r)
				done <- true
			}()

			err := formatYAMLOutput(tt.results, "get")
			w.Close()
			<-done

			if err != nil {
				t.Errorf("formatYAMLOutput() error = %v, want nil", err)
			}

			output := stdout.String()
			tt.checkFn(t, output)
		})
	}
}

func TestFormatOutput(t *testing.T) {
	tests := []struct {
		name       string
		format     outputFormat
		subcommand string
		results    []contextResult
		checkFn    func(t *testing.T, output string)
	}{
		{
			name:       "default format with get subcommand",
			format:     formatDefault,
			subcommand: "get",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    STATUS\npod1    Running",
					err:     nil,
				},
			},
			checkFn: func(t *testing.T, output string) {
				if !strings.Contains(output, "ctx1") {
					t.Errorf("formatOutput() should contain context name")
				}
				if !strings.Contains(output, "pod1") {
					t.Errorf("formatOutput() should contain pod name")
				}
			},
		},
		{
			name:       "default format with version subcommand",
			format:     formatDefault,
			subcommand: "version",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "Client Version: v1.34.3\nServer Version: v1.34.0",
					err:     nil,
				},
			},
			checkFn: func(t *testing.T, output string) {
				if !strings.Contains(output, "CONTEXT") {
					t.Errorf("formatOutput() should use tabular format with CONTEXT header")
				}
				if !strings.Contains(output, "SERVER VERSION") {
					t.Errorf("formatOutput() should use tabular format with SERVER VERSION header")
				}
				if !strings.Contains(output, "Client Version") {
					t.Errorf("formatOutput() should contain client version info")
				}
			},
		},
		{
			name:       "json format",
			format:     formatJSON,
			subcommand: "get",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"items":[{"metadata":{"name":"pod1"}}]}`,
					err:     nil,
				},
			},
			checkFn: func(t *testing.T, output string) {
				if !strings.Contains(output, `"context": "ctx1"`) {
					t.Errorf("formatOutput() should contain context in JSON")
				}
				if !strings.Contains(output, `"kind": "List"`) {
					t.Errorf("formatOutput() should contain List kind")
				}
			},
		},
		{
			name:       "yaml format",
			format:     formatYAML,
			subcommand: "get",
			results: []contextResult{
				{
					context: "ctx1",
					output:  `{"items":[{"metadata":{"name":"pod1"}}]}`,
					err:     nil,
				},
			},
			checkFn: func(t *testing.T, output string) {
				if !strings.Contains(output, "context: ctx1") {
					t.Errorf("formatOutput() should contain context in YAML")
				}
				if !strings.Contains(output, "kind: List") {
					t.Errorf("formatOutput() should contain List kind")
				}
			},
		},
		{
			name:       "default format with logs subcommand",
			format:     formatDefault,
			subcommand: "logs",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "2025-01-01T00:00:00Z first log line\n2025-01-01T00:00:01Z second log line",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "2025-01-01T00:00:00Z another log line",
					err:     nil,
				},
			},
			checkFn: func(t *testing.T, output string) {
				if strings.Contains(output, "CONTEXT") {
					t.Errorf("formatOutput() for logs should not contain a CONTEXT header row")
				}
				lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
				if len(lines) != 3 {
					t.Errorf("formatOutput() for logs should produce 3 lines, got %d", len(lines))
				}
				for _, line := range lines {
					if !strings.HasPrefix(line, "ctx1") && !strings.HasPrefix(line, "ctx2") {
						t.Errorf("each line should be prefixed with a context name, got %q", line)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				os.Stdout = oldStdout
				w.Close()
			}()

			done := make(chan bool)
			go func() {
				io.Copy(&stdout, r)
				done <- true
			}()

			err := formatOutput(tt.results, tt.format, tt.subcommand)
			w.Close()
			<-done

			if err != nil {
				t.Errorf("formatOutput() error = %v, want nil", err)
			}

			output := stdout.String()
			tt.checkFn(t, output)
		})
	}
}

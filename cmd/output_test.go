package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, tt.expected, detectOutputFormat(tt.args))
		})
	}
}

func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func captureOutputCombined(fn func()) string {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = w
	os.Stderr = w

	fn()

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
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
				{context: "ctx1", output: "NAME    STATUS    AGE\npod1    Running   5m"},
			},
			expected: "CONTEXT  NAME    STATUS     AGE\nctx1     pod1    Running    5m\n",
		},
		{
			name: "multiple contexts with header",
			results: []contextResult{
				{context: "ctx1", output: "NAME    STATUS    AGE\npod1    Running   5m"},
				{context: "ctx2", output: "NAME    STATUS    AGE\npod2    Pending   3m"},
			},
			expected: "CONTEXT  NAME    STATUS     AGE\nctx1     pod1    Running    5m\nctx2     pod2    Pending    3m\n",
		},
		{
			name: "contexts with different length names",
			results: []contextResult{
				{context: "short", output: "NAME    STATUS\npod1    Running"},
				{context: "very-long-context-name", output: "NAME    STATUS\npod2    Pending"},
			},
			expected: "CONTEXT                 NAME    STATUS\nshort                   pod1    Running\nvery-long-context-name  pod2    Pending\n",
		},
		{
			name: "context with error",
			results: []contextResult{
				{context: "ctx1", output: "NAME    STATUS\npod1    Running"},
				{context: "ctx2", output: "error message", err: fmt.Errorf("connection failed")},
			},
			expected: "CONTEXT  NAME    STATUS\nctx1     pod1    Running\n",
		},
		{
			name: "context with empty output",
			results: []contextResult{
				{context: "ctx1", output: "NAME    STATUS\npod1    Running"},
				{context: "ctx2", output: ""},
			},
			expected: "CONTEXT  NAME    STATUS\nctx1     pod1    Running\n",
		},
		{
			name: "no header in output",
			results: []contextResult{
				{context: "ctx1", output: "pod1    Running"},
			},
			expected: "ctx1     pod1    Running\n",
		},
		{
			name: "different column widths across contexts",
			results: []contextResult{
				{context: "ctx1", output: "NAME    STATUS    AGE\npod1    Running   5m"},
				{context: "ctx2", output: "NAME         STATUS    AGE\nvery-long-pod-name    Pending   3m"},
			},
			expected: "CONTEXT  NAME                  STATUS     AGE\nctx1     pod1                  Running    5m\nctx2     very-long-pod-name    Pending    3m\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				err := formatDefaultOutput(tt.results)
				require.NoError(t, err)
			})
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestFormatDefaultOutputErrorsBeforeOutput(t *testing.T) {
	results := []contextResult{
		{context: "ctx1", output: "NAME    STATUS\npod1    Running"},
		{context: "ctx2", output: "error message", err: fmt.Errorf("connection failed")},
	}

	combined := captureOutputCombined(func() {
		formatDefaultOutput(results)
	})

	errIdx := strings.Index(combined, "Error:")
	normalIdx := strings.Index(combined, "pod1")

	require.NotEqual(t, -1, errIdx, "expected error message in combined output")
	require.NotEqual(t, -1, normalIdx, "expected normal output in combined output")
	assert.Less(t, errIdx, normalIdx, "error should appear before normal output")
}

func TestFormatLogsOutput(t *testing.T) {
	tests := []struct {
		name     string
		results  []contextResult
		expected string
	}{
		{
			name: "single context with log lines",
			results: []contextResult{
				{context: "ctx1", output: "2025-01-01 log line 1\n2025-01-01 log line 2"},
			},
			expected: "ctx1  2025-01-01 log line 1\nctx1  2025-01-01 log line 2\n",
		},
		{
			name: "multiple contexts with consistent padding",
			results: []contextResult{
				{context: "short", output: "log line from short"},
				{context: "very-long-context-name", output: "log line from long"},
			},
			expected: "short                   log line from short\nvery-long-context-name  log line from long\n",
		},
		{
			name: "context with error",
			results: []contextResult{
				{context: "ctx1", output: "log line 1"},
				{context: "ctx2", output: "error output", err: fmt.Errorf("connection failed")},
			},
			expected: "ctx1  log line 1\n",
		},
		{
			name: "context with empty output",
			results: []contextResult{
				{context: "ctx1", output: "log line 1"},
				{context: "ctx2", output: ""},
			},
			expected: "ctx1  log line 1\n",
		},
		{
			name: "multiple lines from multiple contexts",
			results: []contextResult{
				{context: "ctx1", output: "line1\nline2"},
				{context: "ctx2", output: "line3\nline4"},
			},
			expected: "ctx1  line1\nctx1  line2\nctx2  line3\nctx2  line4\n",
		},
		{
			name:     "all errors",
			results:  []contextResult{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				err := formatLogsOutput(tt.results)
				require.NoError(t, err)
			})
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestFormatLogsOutputErrorsToStderr(t *testing.T) {
	oldStderr := os.Stderr
	stderrR, stderrW, _ := os.Pipe()
	os.Stderr = stderrW

	oldStdout := os.Stdout
	stdoutR, stdoutW, _ := os.Pipe()
	os.Stdout = stdoutW

	defer func() {
		os.Stderr = oldStderr
		os.Stdout = oldStdout
		stderrW.Close()
		stdoutW.Close()
	}()

	var stderrBuf, stdoutBuf bytes.Buffer
	stderrDone := make(chan bool)
	stdoutDone := make(chan bool)
	go func() { io.Copy(&stderrBuf, stderrR); stderrDone <- true }()
	go func() { io.Copy(&stdoutBuf, stdoutR); stdoutDone <- true }()

	results := []contextResult{
		{context: "good-ctx", output: "healthy log line"},
		{context: "bad-ctx", output: "some error detail", err: fmt.Errorf("connection refused")},
	}

	err := formatLogsOutput(results)
	stdoutW.Close()
	stderrW.Close()
	<-stdoutDone
	<-stderrDone

	require.NoError(t, err)

	assert.Contains(t, stdoutBuf.String(), "healthy log line")
	assert.NotContains(t, stdoutBuf.String(), "bad-ctx")
	assert.Contains(t, stderrBuf.String(), "bad-ctx")
	assert.Contains(t, stderrBuf.String(), "connection refused")
}

func TestFormatLogsOutputErrorsBeforeOutput(t *testing.T) {
	results := []contextResult{
		{context: "ctx1", output: "log line one\nlog line two"},
		{context: "ctx2", output: "error message", err: fmt.Errorf("connection failed")},
	}

	combined := captureOutputCombined(func() {
		formatLogsOutput(results)
	})

	errIdx := strings.Index(combined, "Error:")
	normalIdx := strings.Index(combined, "log line one")

	require.NotEqual(t, -1, errIdx, "expected error message in combined output")
	require.NotEqual(t, -1, normalIdx, "expected normal output in combined output")
	assert.Less(t, errIdx, normalIdx, "error should appear before normal output")
}

func TestFormatVersionOutputErrorsBeforeOutput(t *testing.T) {
	results := []contextResult{
		{context: "ctx1", output: "Client Version: v1.34.3\nServer Version: v1.34.0"},
		{context: "ctx2", output: "error message", err: fmt.Errorf("connection failed")},
	}

	combined := captureOutputCombined(func() {
		formatVersionOutput(results)
	})

	errIdx := strings.Index(combined, "Error:")
	tableIdx := strings.Index(combined, "SERVER VERSION")

	require.NotEqual(t, -1, errIdx, "expected error message in combined output")
	require.NotEqual(t, -1, tableIdx, "expected table header in combined output")
	assert.Less(t, errIdx, tableIdx, "error should appear before table output")
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
				{context: "ctx1", output: "Client Version: v1.34.3\nKustomize Version: v5.7.1\nServer Version: v1.34.0"},
			},
			expected: "Client Version: v1.34.3\nKustomize Version: v5.7.1\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\n",
		},
		{
			name: "multiple contexts",
			results: []contextResult{
				{context: "ctx1", output: "Client Version: v1.34.3\nServer Version: v1.34.0"},
				{context: "ctx2", output: "Client Version: v1.34.3\nServer Version: v1.34.0"},
			},
			expected: "Client Version: v1.34.3\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\nctx2                            v1.34.0\n",
		},
		{
			name: "context with error",
			results: []contextResult{
				{context: "ctx1", output: "Client Version: v1.34.3\nServer Version: v1.34.0"},
				{context: "ctx2", output: "error message", err: fmt.Errorf("connection failed")},
			},
			expected: "Client Version: v1.34.3\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\nctx2                            ERROR\n",
		},
		{
			name: "context with empty output",
			results: []contextResult{
				{context: "ctx1", output: "Client Version: v1.34.3\nServer Version: v1.34.0"},
				{context: "ctx2", output: ""},
			},
			expected: "Client Version: v1.34.3\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\nctx2                            N/A\n",
		},
		{
			name: "output with empty lines",
			results: []contextResult{
				{context: "ctx1", output: "Client Version: v1.34.3\n\nServer Version: v1.34.0"},
			},
			expected: "Client Version: v1.34.3\n\nCONTEXT                         SERVER VERSION\n--------------------------------------------------\nctx1                            v1.34.0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				err := formatVersionOutput(tt.results)
				require.NoError(t, err)
			})
			assert.Equal(t, tt.expected, output)
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
				{context: "ctx1", output: `{"items":[{"metadata":{"name":"pod1"}}]}`},
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
}`,
		},
		{
			name: "multiple contexts with items",
			results: []contextResult{
				{context: "ctx1", output: `{"items":[{"metadata":{"name":"pod1"}}]}`},
				{context: "ctx2", output: `{"items":[{"metadata":{"name":"pod2"}}]}`},
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
}`,
		},
		{
			name: "single object without items",
			results: []contextResult{
				{context: "ctx1", output: `{"metadata":{"name":"pod1"}}`},
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
}`,
		},
		{
			name: "object without metadata",
			results: []contextResult{
				{context: "ctx1", output: `{"name":"pod1"}`},
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
}`,
		},
		{
			name: "context with error",
			results: []contextResult{
				{context: "ctx1", output: `{"items":[{"metadata":{"name":"pod1"}}]}`},
				{context: "ctx2", output: `{"error":"connection failed"}`, err: fmt.Errorf("connection failed")},
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
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				err := formatJSONOutput(tt.results, "get")
				require.NoError(t, err)
			})
			assert.Equal(t, strings.TrimSpace(tt.expected), strings.TrimSpace(output))
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
				{context: "ctx1", output: `{"items":[{"metadata":{"name":"pod1"}}]}`},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Contains(t, output, "context: ctx1")
				assert.Contains(t, output, "name: pod1")
			},
		},
		{
			name: "multiple contexts",
			results: []contextResult{
				{context: "ctx1", output: `{"items":[{"metadata":{"name":"pod1"}}]}`},
				{context: "ctx2", output: `{"items":[{"metadata":{"name":"pod2"}}]}`},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Contains(t, output, "context: ctx1")
				assert.Contains(t, output, "context: ctx2")
			},
		},
		{
			name: "object without metadata",
			results: []contextResult{
				{context: "ctx1", output: `{"name":"pod1"}`},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Contains(t, output, "context: ctx1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				err := formatYAMLOutput(tt.results, "get")
				require.NoError(t, err)
			})
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
				{context: "ctx1", output: "NAME    STATUS\npod1    Running"},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Contains(t, output, "ctx1")
				assert.Contains(t, output, "pod1")
			},
		},
		{
			name:       "default format with version subcommand",
			format:     formatDefault,
			subcommand: "version",
			results: []contextResult{
				{context: "ctx1", output: "Client Version: v1.34.3\nServer Version: v1.34.0"},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Contains(t, output, "CONTEXT")
				assert.Contains(t, output, "SERVER VERSION")
				assert.Contains(t, output, "Client Version")
			},
		},
		{
			name:       "json format",
			format:     formatJSON,
			subcommand: "get",
			results: []contextResult{
				{context: "ctx1", output: `{"items":[{"metadata":{"name":"pod1"}}]}`},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Contains(t, output, `"context": "ctx1"`)
				assert.Contains(t, output, `"kind": "List"`)
			},
		},
		{
			name:       "yaml format",
			format:     formatYAML,
			subcommand: "get",
			results: []contextResult{
				{context: "ctx1", output: `{"items":[{"metadata":{"name":"pod1"}}]}`},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Contains(t, output, "context: ctx1")
				assert.Contains(t, output, "kind: List")
			},
		},
		{
			name:       "default format with logs subcommand",
			format:     formatDefault,
			subcommand: "logs",
			results: []contextResult{
				{context: "ctx1", output: "2025-01-01T00:00:00Z first log line\n2025-01-01T00:00:01Z second log line"},
				{context: "ctx2", output: "2025-01-01T00:00:00Z another log line"},
			},
			checkFn: func(t *testing.T, output string) {
				assert.NotContains(t, output, "CONTEXT")
				lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
				assert.Len(t, lines, 3)
				for _, line := range lines {
					assert.True(t, strings.HasPrefix(line, "ctx1") || strings.HasPrefix(line, "ctx2"),
						"each line should be prefixed with a context name, got %q", line)
				}
			},
		},
		{
			name:       "default format with api-versions subcommand",
			format:     formatDefault,
			subcommand: "api-versions",
			results: []contextResult{
				{context: "ctx1", output: "apps/v1\nv1"},
				{context: "ctx2", output: "apps/v1\nv1"},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Equal(t, "ctx1  apps/v1\nctx1  v1\nctx2  apps/v1\nctx2  v1\n", output)
			},
		},
		{
			name:       "api-versions with error context skipped",
			format:     formatDefault,
			subcommand: "api-versions",
			results: []contextResult{
				{context: "ctx1", output: "apps/v1\nv1"},
				{context: "ctx2", output: "error: connection refused", err: fmt.Errorf("connection refused")},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Equal(t, "ctx1  apps/v1\nctx1  v1\n", output)
			},
		},
		{
			name:       "default format with api-resources subcommand",
			format:     formatDefault,
			subcommand: "api-resources",
			results: []contextResult{
				{context: "ctx1", output: "NAME          SHORTNAMES   APIVERSION   NAMESPACED   KIND\nbindings                   v1           true         Binding\npods          po           v1           true         Pod"},
				{context: "ctx2", output: "NAME          SHORTNAMES   APIVERSION   NAMESPACED   KIND\nbindings                   v1           true         Binding\npods          po           v1           true         Pod"},
			},
			checkFn: func(t *testing.T, output string) {
				assert.Contains(t, output, "CONTEXT")
				assert.Contains(t, output, "SHORTNAMES")
				assert.Equal(t, 1, strings.Count(output, "SHORTNAMES"), "header should appear exactly once")
				assert.Contains(t, output, "ctx1")
				assert.Contains(t, output, "ctx2")
				assert.Contains(t, output, "pods")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				err := formatOutput(tt.results, tt.format, tt.subcommand)
				require.NoError(t, err)
			})
			tt.checkFn(t, output)
		})
	}
}

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
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

func TestFormatLogsOutput(t *testing.T) {
	tests := []struct {
		name     string
		results  []contextResult
		expected string
	}{
		{
			name: "single context with log lines",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "2025-01-01 log line 1\n2025-01-01 log line 2",
					err:     nil,
				},
			},
			expected: "ctx1  2025-01-01 log line 1\nctx1  2025-01-01 log line 2\n",
		},
		{
			name: "multiple contexts with consistent padding",
			results: []contextResult{
				{
					context: "short",
					output:  "log line from short",
					err:     nil,
				},
				{
					context: "very-long-context-name",
					output:  "log line from long",
					err:     nil,
				},
			},
			expected: "short                   log line from short\nvery-long-context-name  log line from long\n",
		},
		{
			name: "context with error",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "log line 1",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "error output",
					err:     fmt.Errorf("connection failed"),
				},
			},
			expected: "ctx1  log line 1\n",
		},
		{
			name: "context with empty output",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "log line 1",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "",
					err:     nil,
				},
			},
			expected: "ctx1  log line 1\n",
		},
		{
			name: "multiple lines from multiple contexts",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "line1\nline2",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "line3\nline4",
					err:     nil,
				},
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

			err := formatLogsOutput(tt.results)
			w.Close()
			<-done

			if err != nil {
				t.Errorf("formatLogsOutput() error = %v, want nil", err)
			}

			output := stdout.String()
			if output != tt.expected {
				t.Errorf("formatLogsOutput() output = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestStreamLines(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		coloredCtx string
		padding    string
		expected   string
	}{
		{
			name:       "single line",
			input:      "log line 1\n",
			coloredCtx: "ctx1",
			padding:    "  ",
			expected:   "ctx1    log line 1\n",
		},
		{
			name:       "multiple lines",
			input:      "line1\nline2\nline3\n",
			coloredCtx: "ctx1",
			padding:    "",
			expected:   "ctx1  line1\nctx1  line2\nctx1  line3\n",
		},
		{
			name:       "empty input",
			input:      "",
			coloredCtx: "ctx1",
			padding:    "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)

			r, w, _ := os.Pipe()
			var buf bytes.Buffer
			done := make(chan bool)
			go func() {
				io.Copy(&buf, r)
				done <- true
			}()

			var wg sync.WaitGroup
			var mu sync.Mutex
			wg.Add(1)
			streamLines(&wg, &mu, reader, tt.coloredCtx, tt.padding, w)
			wg.Wait()
			w.Close()
			<-done

			output := buf.String()
			if output != tt.expected {
				t.Errorf("streamLines() output = %q, want %q", output, tt.expected)
			}
		})
	}
}

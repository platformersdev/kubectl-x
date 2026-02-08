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

func TestFormatLogsOutputErrorsToStderr(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	stderrR, stderrW, _ := os.Pipe()
	os.Stderr = stderrW

	// Capture stdout
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
		{
			context: "good-ctx",
			output:  "healthy log line",
			err:     nil,
		},
		{
			context: "bad-ctx",
			output:  "some error detail",
			err:     fmt.Errorf("connection refused"),
		},
	}

	err := formatLogsOutput(results)
	stdoutW.Close()
	stderrW.Close()
	<-stdoutDone
	<-stderrDone

	if err != nil {
		t.Fatalf("formatLogsOutput() returned error: %v", err)
	}

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()

	if !strings.Contains(stdout, "healthy log line") {
		t.Errorf("stdout should contain successful log output, got %q", stdout)
	}
	if strings.Contains(stdout, "bad-ctx") {
		t.Errorf("stdout should not contain error context output, got %q", stdout)
	}
	if !strings.Contains(stderr, "bad-ctx") {
		t.Errorf("stderr should contain the error context name, got %q", stderr)
	}
	if !strings.Contains(stderr, "connection refused") {
		t.Errorf("stderr should contain the error message, got %q", stderr)
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

func TestStreamLinesConcurrentWriters(t *testing.T) {
	lineCount := 100

	var ctx1Input, ctx2Input strings.Builder
	for i := 0; i < lineCount; i++ {
		fmt.Fprintf(&ctx1Input, "ctx1-line-%d\n", i)
		fmt.Fprintf(&ctx2Input, "ctx2-line-%d\n", i)
	}

	r, w, _ := os.Pipe()
	var buf bytes.Buffer
	done := make(chan bool)
	go func() {
		io.Copy(&buf, r)
		done <- true
	}()

	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(2)
	go streamLines(&wg, &mu, strings.NewReader(ctx1Input.String()), "ctx1", "", w)
	go streamLines(&wg, &mu, strings.NewReader(ctx2Input.String()), "ctx2", "", w)
	wg.Wait()
	w.Close()
	<-done

	output := buf.String()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")

	if len(lines) != lineCount*2 {
		t.Fatalf("expected %d lines, got %d", lineCount*2, len(lines))
	}

	for i, line := range lines {
		hasCtx1 := strings.HasPrefix(line, "ctx1  ctx1-line-")
		hasCtx2 := strings.HasPrefix(line, "ctx2  ctx2-line-")
		if !hasCtx1 && !hasCtx2 {
			t.Errorf("line %d appears interleaved or malformed: %q", i, line)
		}
	}
}

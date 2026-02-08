package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestIsWatchMode(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no watch flag",
			args:     []string{"pods"},
			expected: false,
		},
		{
			name:     "short watch flag",
			args:     []string{"pods", "-w"},
			expected: true,
		},
		{
			name:     "long watch flag",
			args:     []string{"pods", "--watch"},
			expected: true,
		},
		{
			name:     "watch-only flag",
			args:     []string{"pods", "--watch-only"},
			expected: true,
		},
		{
			name:     "watch flag before resource",
			args:     []string{"-w", "pods"},
			expected: true,
		},
		{
			name:     "watch flag with other flags",
			args:     []string{"-n", "default", "-w", "pods"},
			expected: true,
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: false,
		},
		{
			name:     "similar but not watch flag",
			args:     []string{"-ww", "pods"},
			expected: false,
		},
		{
			name:     "output flag is not watch",
			args:     []string{"pods", "-o", "wide"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWatchMode(tt.args)
			if result != tt.expected {
				t.Errorf("isWatchMode(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestStreamLinesFilterHeader(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		coloredCtx    string
		padding       string
		contextHeader string
		expected      string
	}{
		{
			name:          "header and data lines",
			input:         "NAME    STATUS    AGE\npod1    Running   5m\npod2    Pending   3m\n",
			coloredCtx:    "ctx1",
			padding:       "",
			contextHeader: "CONTEXT",
			expected:      "CONTEXT  NAME    STATUS    AGE\nctx1  pod1    Running   5m\nctx1  pod2    Pending   3m\n",
		},
		{
			name:          "header only",
			input:         "NAME    STATUS    AGE\n",
			coloredCtx:    "ctx1",
			padding:       "",
			contextHeader: "CONTEXT",
			expected:      "CONTEXT  NAME    STATUS    AGE\n",
		},
		{
			name:          "empty input",
			input:         "",
			coloredCtx:    "ctx1",
			padding:       "",
			contextHeader: "CONTEXT",
			expected:      "",
		},
		{
			name:          "padding applied to data lines",
			input:         "NAME    STATUS\npod1    Running\n",
			coloredCtx:    "ctx1",
			padding:       "    ",
			contextHeader: "CONTEXT ",
			expected:      "CONTEXT   NAME    STATUS\nctx1      pod1    Running\n",
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
			var headerOnce sync.Once
			wg.Add(1)
			streamLinesFilterHeader(&wg, &mu, reader, tt.coloredCtx, tt.padding, tt.contextHeader, w, &headerOnce)
			wg.Wait()
			w.Close()
			<-done

			output := buf.String()
			if output != tt.expected {
				t.Errorf("streamLinesFilterHeader() output = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestStreamLinesFilterHeaderDeduplicatesAcrossContexts(t *testing.T) {
	ctx1Input := "NAME    STATUS\npod1    Running\n"
	ctx2Input := "NAME    STATUS\npod2    Pending\n"

	r, w, _ := os.Pipe()
	var buf bytes.Buffer
	done := make(chan bool)
	go func() {
		io.Copy(&buf, r)
		done <- true
	}()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var headerOnce sync.Once

	// Run sequentially to get deterministic output
	wg.Add(1)
	streamLinesFilterHeader(&wg, &mu, strings.NewReader(ctx1Input), "ctx1", "  ", "CONTEXT", w, &headerOnce)
	wg.Wait()

	wg.Add(1)
	streamLinesFilterHeader(&wg, &mu, strings.NewReader(ctx2Input), "ctx2", "  ", "CONTEXT", w, &headerOnce)
	wg.Wait()

	w.Close()
	<-done

	output := buf.String()

	headerCount := strings.Count(output, "CONTEXT")
	if headerCount != 1 {
		t.Errorf("expected header to appear exactly once, got %d times in %q", headerCount, output)
	}

	if !strings.Contains(output, "CONTEXT  NAME    STATUS") {
		t.Errorf("expected unified header line, got %q", output)
	}
	if !strings.Contains(output, "ctx1    pod1    Running") {
		t.Errorf("expected ctx1 data line, got %q", output)
	}
	if !strings.Contains(output, "ctx2    pod2    Pending") {
		t.Errorf("expected ctx2 data line, got %q", output)
	}

	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (1 header + 2 data), got %d: %q", len(lines), output)
	}
}

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

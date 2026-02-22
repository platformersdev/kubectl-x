package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

			assert.Equal(t, tt.expected, buf.String())
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

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	require.Len(t, lines, lineCount*2)

	for i, line := range lines {
		hasCtx1 := strings.HasPrefix(line, "ctx1  ctx1-line-")
		hasCtx2 := strings.HasPrefix(line, "ctx2  ctx2-line-")
		assert.True(t, hasCtx1 || hasCtx2, "line %d appears interleaved or malformed: %q", i, line)
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

			assert.Equal(t, tt.expected, buf.String())
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

	assert.Equal(t, 1, strings.Count(output, "CONTEXT"), "header should appear exactly once")
	assert.Contains(t, output, "CONTEXT  NAME    STATUS")
	assert.Contains(t, output, "ctx1    pod1    Running")
	assert.Contains(t, output, "ctx2    pod2    Pending")

	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	assert.Len(t, lines, 3, "expected 1 header + 2 data lines")
}

func captureStderr(fn func()) string {
	r, w, _ := os.Pipe()
	oldStderr := os.Stderr
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name      string
		started   float64
		completed float64
		total     int
		wantText  string
		wantWhite bool
		wantGray  bool
	}{
		{
			name:      "all pending",
			started:   0,
			completed: 0,
			total:     10,
			wantText:  "0/10 complete",
			wantGray:  true,
		},
		{
			name:      "some started none completed",
			started:   3,
			completed: 0,
			total:     10,
			wantText:  "0/10 complete",
			wantGray:  true,
		},
		{
			name:      "some completed some in progress",
			started:   6,
			completed: 3,
			total:     10,
			wantText:  "3/10 complete",
			wantWhite: true,
			wantGray:  true,
		},
		{
			name:      "all completed",
			started:   10,
			completed: 10,
			total:     10,
			wantText:  "10/10 complete",
			wantWhite: true,
		},
		{
			name:      "zero total",
			started:   0,
			completed: 0,
			total:     0,
			wantText:  "",
		},
		{
			name:      "fractional values mid-animation",
			started:   5.5,
			completed: 2.7,
			total:     10,
			wantText:  "2/10 complete",
			wantWhite: true,
			wantGray:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderProgressBar(tt.started, tt.completed, tt.total)
			if tt.total == 0 {
				assert.Empty(t, result)
				return
			}
			assert.Contains(t, result, tt.wantText)
			if tt.wantWhite {
				assert.Contains(t, result, colorWhite)
			}
			if tt.wantGray {
				assert.Contains(t, result, colorGray)
			}
		})
	}
}

func TestRenderProgressBarWidth(t *testing.T) {
	result := renderProgressBar(30, 30, 30)
	assert.Equal(t, strings.Count(result, "█"), progressBarWidth, "fully completed bar should have exactly progressBarWidth full blocks")
	assert.NotContains(t, result, "░")
}

func TestRenderProgressBarPartialBlocks(t *testing.T) {
	result := renderProgressBar(1, 0, 100)
	hasPartial := false
	for _, p := range partialBlocks[1:] {
		if strings.Contains(result, p) {
			hasPartial = true
			break
		}
	}
	assert.True(t, hasPartial, "small progress should produce a partial block character")
}

func TestLerp(t *testing.T) {
	result := lerp(0.0, 10.0)
	assert.Greater(t, result, 0.0)
	assert.Less(t, result, 10.0)

	result = lerp(9.96, 10.0)
	assert.Equal(t, 10.0, result, "should snap to target when close enough")
}

func TestClearProgress(t *testing.T) {
	output := captureStderr(func() {
		clearProgress()
	})

	assert.Contains(t, output, "\r\033[K")
}

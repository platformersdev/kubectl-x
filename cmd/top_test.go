package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestTopCmd(t *testing.T) {
	if topCmd == nil {
		t.Fatal("topCmd should not be nil")
	}
	if topCmd.Use != "top" {
		t.Errorf("topCmd.Use = %q, want %q", topCmd.Use, "top")
	}
	if !topCmd.DisableFlagParsing {
		t.Error("topCmd should have DisableFlagParsing enabled")
	}
}

func TestFormatTopOutput(t *testing.T) {
	tests := []struct {
		name     string
		results  []contextResult
		expected string
	}{
		{
			name: "single context nodes",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME     CPU(cores)   MEMORY(bytes)\nnode1    100m         500Mi",
					err:     nil,
				},
			},
			expected: "CONTEXT  NAME     CPU(cores)    MEMORY(bytes)\nctx1     node1    100m          500Mi\n",
		},
		{
			name: "single context pods",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    CPU(cores)   MEMORY(bytes)\npod1    10m          100Mi\npod2    20m          200Mi",
					err:     nil,
				},
			},
			expected: "CONTEXT  NAME    CPU(cores)    MEMORY(bytes)\nctx1     pod1    10m           100Mi\nctx1     pod2    20m           200Mi\n",
		},
		{
			name: "multiple contexts",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    CPU(cores)   MEMORY(bytes)\npod1    10m          100Mi",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "NAME    CPU(cores)   MEMORY(bytes)\npod2    20m          200Mi",
					err:     nil,
				},
			},
			expected: "CONTEXT  NAME    CPU(cores)    MEMORY(bytes)\nctx1     pod1    10m           100Mi\nctx2     pod2    20m           200Mi\n",
		},
		{
			name: "context with error",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    CPU(cores)   MEMORY(bytes)\npod1    10m          100Mi",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "error message",
					err:     fmt.Errorf("metrics not available"),
				},
			},
			expected: "CONTEXT  NAME    CPU(cores)    MEMORY(bytes)\nctx1     pod1    10m           100Mi\n",
		},
		{
			name: "context with empty output",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "NAME    CPU(cores)   MEMORY(bytes)\npod1    10m          100Mi",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "",
					err:     nil,
				},
			},
			expected: "CONTEXT  NAME    CPU(cores)    MEMORY(bytes)\nctx1     pod1    10m           100Mi\n",
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

func TestFormatTopOutputErrorsBeforeOutput(t *testing.T) {
	results := []contextResult{
		{
			context: "ctx1",
			output:  "NAME    CPU(cores)   MEMORY(bytes)\npod1    10m          100Mi",
			err:     nil,
		},
		{
			context: "ctx2",
			output:  "error message",
			err:     fmt.Errorf("metrics not available"),
		},
	}

	combined := captureOutputCombined(func() {
		formatDefaultOutput(results)
	})

	errIdx := strings.Index(combined, "Error:")
	normalIdx := strings.Index(combined, "pod1")

	if errIdx == -1 {
		t.Fatal("expected error message in combined output")
	}
	if normalIdx == -1 {
		t.Fatal("expected normal output in combined output")
	}
	if errIdx > normalIdx {
		t.Errorf("error (at index %d) should appear before normal output (at index %d)", errIdx, normalIdx)
	}
}

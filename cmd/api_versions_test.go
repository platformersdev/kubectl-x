package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestApiVersionsCmd(t *testing.T) {
	if apiVersionsCmd == nil {
		t.Fatal("apiVersionsCmd should not be nil")
	}
	if apiVersionsCmd.Use != "api-versions" {
		t.Errorf("apiVersionsCmd.Use = %q, want %q", apiVersionsCmd.Use, "api-versions")
	}
	if !apiVersionsCmd.DisableFlagParsing {
		t.Error("apiVersionsCmd should have DisableFlagParsing enabled")
	}
}

func TestFormatApiVersionsOutput(t *testing.T) {
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
					output:  "apps/v1\nv1",
					err:     nil,
				},
			},
			expected: "ctx1  apps/v1\nctx1  v1\n",
		},
		{
			name: "multiple contexts",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "apps/v1\nv1",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "apps/v1\nv1",
					err:     nil,
				},
			},
			expected: "ctx1  apps/v1\nctx1  v1\nctx2  apps/v1\nctx2  v1\n",
		},
		{
			name: "context with error is skipped",
			results: []contextResult{
				{
					context: "ctx1",
					output:  "apps/v1\nv1",
					err:     nil,
				},
				{
					context: "ctx2",
					output:  "error: connection refused",
					err:     fmt.Errorf("connection refused"),
				},
			},
			expected: "ctx1  apps/v1\nctx1  v1\n",
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

			err := formatOutput(tt.results, formatDefault, "api-versions")
			w.Close()
			<-done

			if err != nil {
				t.Errorf("formatOutput() error = %v, want nil", err)
			}

			output := stdout.String()
			if output != tt.expected {
				t.Errorf("formatOutput() output = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestFormatApiResourcesOutput(t *testing.T) {
	results := []contextResult{
		{
			context: "ctx1",
			output:  "NAME          SHORTNAMES   APIVERSION   NAMESPACED   KIND\nbindings                   v1           true         Binding\npods          po           v1           true         Pod",
			err:     nil,
		},
		{
			context: "ctx2",
			output:  "NAME          SHORTNAMES   APIVERSION   NAMESPACED   KIND\nbindings                   v1           true         Binding\npods          po           v1           true         Pod",
			err:     nil,
		},
	}

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

	err := formatOutput(results, formatDefault, "api-resources")
	w.Close()
	<-done

	if err != nil {
		t.Fatalf("formatOutput() error = %v, want nil", err)
	}

	output := stdout.String()

	if !strings.Contains(output, "CONTEXT") {
		t.Error("expected CONTEXT column in header")
	}
	if !strings.Contains(output, "SHORTNAMES") {
		t.Error("expected SHORTNAMES column in header")
	}
	if strings.Count(output, "SHORTNAMES") != 1 {
		t.Errorf("expected header to appear exactly once, got %d times", strings.Count(output, "SHORTNAMES"))
	}
	if !strings.Contains(output, "ctx1") {
		t.Error("expected ctx1 in output")
	}
	if !strings.Contains(output, "ctx2") {
		t.Error("expected ctx2 in output")
	}
	if !strings.Contains(output, "pods") {
		t.Error("expected pods in output")
	}
}

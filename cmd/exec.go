package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type contextResult struct {
	context string
	output  string
	err     error
}

func runCommand(subcommand string, extraArgs []string) error {
	contexts, err := getContexts()
	if err != nil {
		return fmt.Errorf("failed to get contexts: %w", err)
	}

	if len(contexts) == 0 {
		return fmt.Errorf("no contexts found in kubeconfig")
	}

	results := make([]contextResult, len(contexts))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, batchSize)

	for i, ctx := range contexts {
		wg.Add(1)
		go func(index int, context string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			output, err := runKubectlCommand(context, subcommand, extraArgs)
			results[index] = contextResult{
				context: context,
				output:  output,
				err:     err,
			}
		}(i, ctx)
	}

	wg.Wait()

	// Determine output format
	outputFormat := detectOutputFormat(extraArgs)

	// Format and print results
	return formatOutput(results, outputFormat, subcommand)
}

func runKubectlCommand(context, subcommand string, extraArgs []string) (string, error) {
	args := []string{"--context", context, subcommand}
	args = append(args, extraArgs...)

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// hasFollowFlag checks if the -f or --follow flag is present in the arguments
func hasFollowFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-f" || arg == "--follow" {
			return true
		}
	}
	return false
}

// runCommandStreaming runs a kubectl subcommand with streaming support
// It streams output from all contexts in parallel, prefixing each line with the context name
func runCommandStreaming(subcommand string, extraArgs []string) error {
	contexts, err := getContexts()
	if err != nil {
		return fmt.Errorf("failed to get contexts: %w", err)
	}

	if len(contexts) == 0 {
		return fmt.Errorf("no contexts found in kubeconfig")
	}

	// Calculate maximum context name width for consistent alignment
	maxContextWidth := 0
	for _, ctx := range contexts {
		if len(ctx) > maxContextWidth {
			maxContextWidth = len(ctx)
		}
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, batchSize)
	var mu sync.Mutex // Protects stdout writes

	// Helper function to format context name with padding
	formatContextPrefix := func(context string) string {
		coloredContext := colorizeContext(context)
		// Calculate padding needed (using actual context length, not colored length)
		padding := maxContextWidth - len(context)
		if padding > 0 {
			return coloredContext + strings.Repeat(" ", padding) + "  "
		}
		return coloredContext + "  "
	}

	for _, ctx := range contexts {
		wg.Add(1)
		go func(context string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			args := []string{"--context", context, subcommand}
			args = append(args, extraArgs...)

			cmd := exec.Command("kubectl", args...)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "%sFailed to create stdout pipe: %v\n", formatContextPrefix(context), err)
				mu.Unlock()
				return
			}

			stderr, err := cmd.StderrPipe()
			if err != nil {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "%sFailed to create stderr pipe: %v\n", formatContextPrefix(context), err)
				mu.Unlock()
				return
			}

			if err := cmd.Start(); err != nil {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "%sFailed to start command: %v\n", formatContextPrefix(context), err)
				mu.Unlock()
				return
			}

			// WaitGroup for stdout/stderr goroutines
			var streamWg sync.WaitGroup

			// Stream stdout
			streamWg.Add(1)
			go func() {
				defer streamWg.Done()
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					line := scanner.Text()
					mu.Lock()
					fmt.Printf("%s%s\n", formatContextPrefix(context), line)
					mu.Unlock()
				}
				if err := scanner.Err(); err != nil && err != io.EOF {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "%sError reading stdout: %v\n", formatContextPrefix(context), err)
					mu.Unlock()
				}
			}()

			// Stream stderr
			streamWg.Add(1)
			go func() {
				defer streamWg.Done()
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					line := scanner.Text()
					mu.Lock()
					fmt.Fprintf(os.Stderr, "%s%s\n", formatContextPrefix(context), line)
					mu.Unlock()
				}
				if err := scanner.Err(); err != nil && err != io.EOF {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "%sError reading stderr: %v\n", formatContextPrefix(context), err)
					mu.Unlock()
				}
			}()

			// Wait for command to complete
			if err := cmd.Wait(); err != nil {
				// Don't print error if it's just exit code 1 (common for logs)
				// Only print if it's a more serious error
				if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 1 {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "%sCommand exited with error: %v\n", formatContextPrefix(context), err)
					mu.Unlock()
				}
			}

			// Wait for streaming goroutines to finish
			streamWg.Wait()
		}(ctx)
	}

	wg.Wait()
	return nil
}

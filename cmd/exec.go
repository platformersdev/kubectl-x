package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"golang.org/x/term"
)

type contextResult struct {
	context string
	output  string
	err     error
}

func stderrIsTerminal() bool {
	return term.IsTerminal(int(os.Stderr.Fd()))
}

const progressBarWidth = 30

func renderProgressBar(started, completed, total int) string {
	if total == 0 {
		return ""
	}

	completedWidth := (completed * progressBarWidth) / total
	startedWidth := (started * progressBarWidth) / total
	inProgressWidth := startedWidth - completedWidth
	pendingWidth := progressBarWidth - completedWidth - inProgressWidth

	var bar strings.Builder
	bar.WriteString(colorWhite) // bright white for completed
	bar.WriteString(strings.Repeat("█", completedWidth))
	bar.WriteString(colorGray) // dark gray for in-progress
	bar.WriteString(strings.Repeat("█", inProgressWidth))
	bar.WriteString(colorGray)
	bar.WriteString(strings.Repeat("░", pendingWidth))
	bar.WriteString(colorReset)

	return fmt.Sprintf("\r\033[K %s %d/%d complete", bar.String(), completed, total)
}

func showProgress(started, completed *atomic.Int32, total int) {
	fmt.Fprint(os.Stderr, renderProgressBar(int(started.Load()), int(completed.Load()), total))
}

func clearProgress() {
	fmt.Fprintf(os.Stderr, "\r\033[K")
}

func runCommand(subcommand string, extraArgs []string) error {
	contexts, err := getContexts()
	if err != nil {
		return fmt.Errorf("failed to get contexts: %w", err)
	}

	if len(contexts) == 0 {
		return fmt.Errorf("no contexts found in kubeconfig")
	}

	showStatus := stderrIsTerminal()
	var started, completed atomic.Int32
	total := len(contexts)

	if showStatus {
		showProgress(&started, &completed, total)
	}

	results := make([]contextResult, len(contexts))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, batchSize)

	for i, ctx := range contexts {
		wg.Add(1)
		go func(index int, context string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			started.Add(1)
			if showStatus {
				showProgress(&started, &completed, total)
			}

			output, err := runKubectlCommand(context, subcommand, extraArgs)
			results[index] = contextResult{
				context: context,
				output:  output,
				err:     err,
			}
			completed.Add(1)
			if showStatus {
				showProgress(&started, &completed, total)
			}
		}(i, ctx)
	}

	wg.Wait()

	if showStatus {
		clearProgress()
	}

	outputFormat := detectOutputFormat(extraArgs)
	return formatOutput(results, outputFormat, subcommand)
}

func runKubectlCommand(context, subcommand string, extraArgs []string) (string, error) {
	args := []string{"--context", context, subcommand}
	args = append(args, extraArgs...)

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runStreamingCommand(subcommand string, extraArgs []string, filterHeaders bool) error {
	contexts, err := getContexts()
	if err != nil {
		return fmt.Errorf("failed to get contexts: %w", err)
	}

	if len(contexts) == 0 {
		return fmt.Errorf("no contexts found in kubeconfig")
	}

	maxWidth := 0
	for _, ctx := range contexts {
		if len(ctx) > maxWidth {
			maxWidth = len(ctx)
		}
	}
	if filterHeaders && maxWidth < len("CONTEXT") {
		maxWidth = len("CONTEXT")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	var mu sync.Mutex
	var wg sync.WaitGroup
	var cmds []*exec.Cmd
	var headerOnce sync.Once

	for _, ctx := range contexts {
		kubectlArgs := []string{"--context", ctx, subcommand}
		kubectlArgs = append(kubectlArgs, extraArgs...)

		cmd := exec.Command("kubectl", kubectlArgs...)
		cmds = append(cmds, cmd)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: failed to create stdout pipe: %v\n", ctx, err)
			continue
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: failed to create stderr pipe: %v\n", ctx, err)
			continue
		}

		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: failed to start: %v\n", ctx, err)
			continue
		}

		coloredCtx := colorizeContext(ctx)
		padding := strings.Repeat(" ", maxWidth-len(ctx))

		wg.Add(1)
		if filterHeaders {
			contextHeader := "CONTEXT" + strings.Repeat(" ", maxWidth-len("CONTEXT"))
			go streamLinesFilterHeader(&wg, &mu, stdout, coloredCtx, padding, contextHeader, os.Stdout, &headerOnce)
		} else {
			go streamLines(&wg, &mu, stdout, coloredCtx, padding, os.Stdout)
		}

		wg.Add(1)
		go streamLines(&wg, &mu, stderr, coloredCtx, padding, os.Stderr)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-sigChan:
		for _, cmd := range cmds {
			if cmd.Process != nil {
				cmd.Process.Signal(syscall.SIGTERM)
			}
		}
		for _, cmd := range cmds {
			cmd.Wait()
		}
	case <-done:
		for _, cmd := range cmds {
			cmd.Wait()
		}
	}

	return nil
}

func streamLines(wg *sync.WaitGroup, mu *sync.Mutex, reader io.Reader, coloredCtx, padding string, dest *os.File) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		mu.Lock()
		fmt.Fprintf(dest, "%s%s  %s\n", coloredCtx, padding, line)
		mu.Unlock()
	}
}

// streamLinesFilterHeader prints the first line (header) exactly once across
// all goroutines sharing the same headerOnce, then streams remaining lines
// with the context prefix.
func streamLinesFilterHeader(wg *sync.WaitGroup, mu *sync.Mutex, reader io.Reader, coloredCtx, padding, contextHeader string, dest *os.File, headerOnce *sync.Once) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	firstLine := true
	for scanner.Scan() {
		line := scanner.Text()
		if firstLine {
			firstLine = false
			headerOnce.Do(func() {
				mu.Lock()
				fmt.Fprintf(dest, "%s  %s\n", contextHeader, line)
				mu.Unlock()
			})
			continue
		}
		mu.Lock()
		fmt.Fprintf(dest, "%s%s  %s\n", coloredCtx, padding, line)
		mu.Unlock()
	}
}

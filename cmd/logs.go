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
	"syscall"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:                "logs",
	Short:              "Run kubectl logs against all contexts",
	Long:               `Run kubectl logs command against all contexts in parallel. Supports streaming with -f/--follow flag.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isFollowMode(args) {
			return runStreamingLogs(args)
		}
		return runLogsCommand(args)
	},
}

func isFollowMode(args []string) bool {
	for _, arg := range args {
		if arg == "-f" || arg == "--follow" {
			return true
		}
	}
	return false
}

func runLogsCommand(args []string) error {
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
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			output, err := runKubectlCommand(context, "logs", args)
			results[index] = contextResult{
				context: context,
				output:  output,
				err:     err,
			}
		}(i, ctx)
	}

	wg.Wait()

	return formatLogsOutput(results)
}

func runStreamingLogs(args []string) error {
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	var mu sync.Mutex
	var wg sync.WaitGroup
	var cmds []*exec.Cmd

	for _, ctx := range contexts {
		kubectlArgs := []string{"--context", ctx, "logs"}
		kubectlArgs = append(kubectlArgs, args...)

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
		go streamLines(&wg, &mu, stdout, coloredCtx, padding, os.Stdout)

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

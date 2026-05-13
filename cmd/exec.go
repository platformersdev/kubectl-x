package cmd

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

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

var partialBlocks = []string{" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

const lerpFactor = 0.15
const lerpSnap = 0.05

func terminalWidth() int {
	w, _, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

func lerp(display, target float64) float64 {
	display += (target - display) * lerpFactor
	if math.Abs(target-display) < lerpSnap {
		return target
	}
	return display
}

func renderProgressBar(displayStarted, displayCompleted float64, total int) string {
	if total == 0 {
		return ""
	}

	suffix := fmt.Sprintf(" %d/%d complete", int(displayCompleted), total)
	barWidth := terminalWidth() - 1 - len(suffix)
	if barWidth < 10 {
		barWidth = 10
	}

	cEighths := int(displayCompleted * float64(barWidth) * 8 / float64(total))
	sEighths := int(displayStarted * float64(barWidth) * 8 / float64(total))

	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		left := i * 8
		right := (i + 1) * 8

		switch {
		case right <= cEighths:
			bar.WriteString(colorWhite + "█")
		case left >= sEighths:
			bar.WriteString(colorGray + "░")
		case left >= cEighths && right <= sEighths:
			bar.WriteString(colorGray + "█")
		case left < cEighths:
			bar.WriteString(colorWhite + partialBlocks[cEighths-left])
		default:
			bar.WriteString(colorGray + partialBlocks[sEighths-left])
		}
	}
	bar.WriteString(colorReset)

	return fmt.Sprintf("\r\033[K %s%s", bar.String(), suffix)
}

func clearProgress() {
	fmt.Fprintf(os.Stderr, "\r\033[K")
}

type progressBar struct {
	started   atomic.Int32
	completed atomic.Int32
	total     int
	stop      chan struct{}
	done      chan struct{}
}

func newProgressBar(total int) *progressBar {
	p := &progressBar{
		total: total,
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go p.animate()
	return p
}

func (p *progressBar) animate() {
	defer close(p.done)
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()

	displayStarted := 0.0
	displayCompleted := 0.0

	for {
		select {
		case <-p.stop:
			clearProgress()
			return
		case <-ticker.C:
			targetStarted := float64(p.started.Load())
			targetCompleted := float64(p.completed.Load())

			displayStarted = lerp(displayStarted, targetStarted)
			displayCompleted = lerp(displayCompleted, targetCompleted)

			fmt.Fprint(os.Stderr, renderProgressBar(displayStarted, displayCompleted, p.total))
		}
	}
}

func (p *progressBar) finish() {
	close(p.stop)
	<-p.done
}

func hasSortBy(args []string) bool {
	for _, arg := range args {
		if arg == "--sort-by" || strings.HasPrefix(arg, "--sort-by=") {
			return true
		}
	}
	return false
}

func runCommand(subcommand string, extraArgs []string) error {
	if hasSortBy(extraArgs) {
		fmt.Fprintf(os.Stderr, "Warning: --sort-by sorts within each context independently and may not produce the expected global ordering. See https://github.com/platformersdev/kubectl-x/issues/29\n")
	}

	contexts, err := getContexts()
	if err != nil {
		return fmt.Errorf("failed to get contexts: %w", err)
	}

	if len(contexts) == 0 {
		return fmt.Errorf("no contexts found in kubeconfig")
	}

	showStatus := stderrIsTerminal()
	total := len(contexts)

	var progress *progressBar
	if showStatus {
		progress = newProgressBar(total)
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

			if progress != nil {
				progress.started.Add(1)
			}

			output, err := runKubectlCommand(context, subcommand, extraArgs)
			results[index] = contextResult{
				context: context,
				output:  output,
				err:     err,
			}

			if progress != nil {
				progress.completed.Add(1)
			}
		}(i, ctx)
	}

	wg.Wait()

	if progress != nil {
		progress.finish()
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

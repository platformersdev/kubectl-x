package main

import (
	"fmt"
	"os"

	"github.com/platformersdev/kubectl-multi_context/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

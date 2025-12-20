package cmd

import (
	"github.com/spf13/cobra"
)

var batchSize int = 25

var rootCmd = &cobra.Command{
	Use:              "kubectl multi-context",
	Short:            "Run kubectl commands against every context in kubeconfig",
	Long:             `kubectl multi-context executes commands against all contexts in your kubeconfig file in parallel.`,
	TraverseChildren: true, // this lets us use root-level flags, but still allow subcommands to disable flag parsing
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().IntVarP(&batchSize, "batch-size", "b", 25, "Number of contexts to process in parallel")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(getCmd)
}

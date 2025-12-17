package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kubectl multi-context",
	Short: "Run kubectl commands against every context in kubeconfig",
	Long:  `kubectl multi-context executes commands against all contexts in your kubeconfig file in parallel.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(getCmd)
}

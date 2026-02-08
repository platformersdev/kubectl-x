package cmd

import (
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:                "logs",
	Short:              "Run kubectl logs against all contexts",
	Long:               `Run kubectl logs command against all contexts in parallel. Supports streaming with -f flag.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use streaming mode if -f or --follow flag is present
		if hasFollowFlag(args) {
			return runCommandStreaming("logs", args)
		}
		// Otherwise use the standard batch mode
		return runCommand("logs", args)
	},
}

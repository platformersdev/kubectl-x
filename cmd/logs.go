package cmd

import (
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:                "logs",
	Short:              "Run kubectl logs against all contexts",
	Long:               `Run kubectl logs command against all contexts in parallel. Supports streaming with -f/--follow flag.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isFollowMode(args) {
			return runStreamingCommand("logs", args)
		}
		return runCommand("logs", args)
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

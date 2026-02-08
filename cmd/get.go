package cmd

import (
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:                "get",
	Short:              "Run kubectl get against all contexts",
	Long:               `Run kubectl get command against all contexts in parallel. Supports streaming with -w/--watch flag.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isWatchMode(args) {
			return runStreamingCommand("get", args, true)
		}
		return runCommand("get", args)
	},
}

func isWatchMode(args []string) bool {
	for _, arg := range args {
		if arg == "-w" || arg == "--watch" || arg == "--watch-only" {
			return true
		}
	}
	return false
}

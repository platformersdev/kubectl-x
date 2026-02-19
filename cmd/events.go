package cmd

import (
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:                "events",
	Short:              "Run kubectl events against all contexts",
	Long:               `Run kubectl events command against all contexts in parallel. Supports streaming with -w/--watch flag.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isWatchMode(args) {
			return runStreamingCommand("events", args, false)
		}
		return runCommand("events", args)
	},
}

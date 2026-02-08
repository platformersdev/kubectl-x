package cmd

import (
	"github.com/spf13/cobra"
)

var waitCmd = &cobra.Command{
	Use:                "wait",
	Short:              "Run kubectl wait against all contexts",
	Long:               `Run kubectl wait command against all contexts in parallel.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCommand("wait", args)
	},
}

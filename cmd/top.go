package cmd

import (
	"github.com/spf13/cobra"
)

var topCmd = &cobra.Command{
	Use:                "top",
	Short:              "Run kubectl top against all contexts",
	Long:               `Run kubectl top command against all contexts in parallel.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCommand("top", args)
	},
}

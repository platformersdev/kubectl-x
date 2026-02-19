package cmd

import (
	"github.com/spf13/cobra"
)

var apiVersionsCmd = &cobra.Command{
	Use:                "api-versions",
	Short:              "Run kubectl api-versions against all contexts",
	Long:               `Run kubectl api-versions command against all contexts in parallel.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCommand("api-versions", args)
	},
}

package cmd

import (
	"github.com/spf13/cobra"
)

var apiResourcesCmd = &cobra.Command{
	Use:                "api-resources",
	Short:              "Run kubectl api-resources against all contexts",
	Long:               `Run kubectl api-resources command against all contexts in parallel.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCommand("api-resources", args)
	},
}

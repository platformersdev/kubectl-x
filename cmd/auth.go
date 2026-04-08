package cmd

import (
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:                "auth",
	Short:              "Run kubectl auth subcommands against all contexts",
	Long:               `Run kubectl auth subcommands (e.g. whoami, can-i) against all contexts in parallel.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCommand("auth", args)
	},
}

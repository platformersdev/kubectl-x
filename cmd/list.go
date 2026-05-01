package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all matching contexts",
	Long:  `List all contexts from kubeconfig, optionally filtered by --include and --exclude.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList()
	},
}

func runList() error {
	contexts, err := getContexts()
	if err != nil {
		return err
	}
	for _, ctx := range contexts {
		fmt.Println(ctx)
	}
	return nil
}

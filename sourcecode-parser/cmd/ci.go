package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "CI mode - Python DSL implementation in progress",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("CI command not yet implemented in new architecture")
	},
}

func init() {
	rootCmd.AddCommand(ciCmd)
}

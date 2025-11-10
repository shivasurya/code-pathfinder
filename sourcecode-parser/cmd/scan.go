package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan mode - Python DSL implementation in progress",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("Scan command not yet implemented in new architecture")
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

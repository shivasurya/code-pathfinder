package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query mode - Python DSL implementation in progress",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("Query command not yet implemented in new architecture")
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
}

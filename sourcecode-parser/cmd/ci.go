package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Scan a project for vulnerabilities with ruleset in ci mode",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("CI mode not implemented yet. Coming soon")
	},
}

func init() {
	rootCmd.AddCommand(ciCmd)
}

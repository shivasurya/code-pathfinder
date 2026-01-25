package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and commit information",
	Run: func(_ *cobra.Command, _ []string) {
		// Version is a debug command - no analytics tracking
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

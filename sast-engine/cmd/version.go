package cmd

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"

	"github.com/spf13/cobra"
)

var (
	Version   = "0.0.24"
	GitCommit = "HEAD"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and commit information",
	Run: func(_ *cobra.Command, _ []string) {
		analytics.ReportEvent(analytics.VersionCommand)
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

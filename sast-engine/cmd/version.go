package cmd

import (
	"fmt"
	"os"

	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and commit information",
	Run: func(cmd *cobra.Command, _ []string) {
		// Display banner unless --no-banner is set
		noBanner, _ := cmd.Parent().PersistentFlags().GetBool("no-banner")
		if !noBanner {
			output.PrintBanner(os.Stderr, Version, output.DefaultBannerOptions())
		}

		// Version info
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

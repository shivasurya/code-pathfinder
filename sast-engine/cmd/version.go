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
		// Display banner
		noBanner, _ := cmd.Parent().PersistentFlags().GetBool("no-banner")
		logger := output.NewLogger(output.VerbosityDefault)
		if output.ShouldShowBanner(logger.IsTTY(), noBanner) {
			output.PrintBanner(logger.GetWriter(), Version, output.DefaultBannerOptions())
		} else if logger.IsTTY() && !noBanner {
			fmt.Fprintln(os.Stderr, output.GetCompactBanner(Version))
			fmt.Fprintln(os.Stderr)
		}

		// Version is a debug command - no analytics tracking
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

package cmd

import (
	"fmt"
	"os"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/spf13/cobra"
)

var (
	verboseFlag bool
	Version     = "1.2.2"
	GitCommit   = "HEAD"
)

var rootCmd = &cobra.Command{
	Use:   "pathfinder",
	Short: "AI-Native Static Code Analysis | Graph-First Engine | Privacy-First",
	Long:  `Code Pathfinder - AI-Native static code analysis with graph-first engine.

Combines structural analysis (call graphs, dataflow, taint tracking) with AI to understand
real exploit paths. Supports Python, Docker, and docker-compose with language-agnostic queries.

Learn more: https://codepathfinder.dev`,
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		disableMetrics, _ := cmd.Flags().GetBool("disable-metrics") //nolint:all
		verboseFlag, _ = cmd.Flags().GetBool("verbose")             //nolint:all
		analytics.LoadEnvFile()
		analytics.Init(disableMetrics)
		analytics.SetVersion(Version)
		if verboseFlag {
			graph.EnableVerboseLogging()
		}

		// Show banner for help command
		if cmd.Name() == "help" || (len(os.Args) == 1 || (len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h"))) {
			noBanner, _ := cmd.Flags().GetBool("no-banner")
			logger := output.NewLogger(output.VerbosityDefault)
			if output.ShouldShowBanner(logger.IsTTY(), noBanner) {
				output.PrintBanner(logger.GetWriter(), Version, output.DefaultBannerOptions())
			} else if logger.IsTTY() && !noBanner {
				fmt.Fprintln(os.Stderr, output.GetCompactBanner(Version))
				fmt.Fprintln(os.Stderr)
			}
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().Bool("disable-metrics", false, "Disable metrics collection")
	rootCmd.PersistentFlags().Bool("verbose", false, "Verbose output")
	rootCmd.PersistentFlags().Bool("no-banner", false, "Disable startup banner")
}

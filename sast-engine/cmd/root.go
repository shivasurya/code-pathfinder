package cmd

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/spf13/cobra"
)

var verboseFlag bool

var rootCmd = &cobra.Command{
	Use:   "pathfinder",
	Short: "Code Pathfinder - A query language for structural search on source code",
	Long:  `Code Pathfinder is designed for identifying vulnerabilities in source code.`,
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		disableMetrics, _ := cmd.Flags().GetBool("disable-metrics") //nolint:all
		verboseFlag, _ = cmd.Flags().GetBool("verbose")             //nolint:all
		analytics.LoadEnvFile()
		analytics.Init(disableMetrics)
		if verboseFlag {
			graph.EnableVerboseLogging()
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().Bool("disable-metrics", false, "Disable metrics collection")
	rootCmd.PersistentFlags().Bool("verbose", false, "Verbose output")
}

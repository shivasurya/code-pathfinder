package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
	"github.com/spf13/cobra"
)

var (
	verboseFlag            bool
	Version                = "2.0.2"
	GitCommit              = "HEAD"
	updateCheckManifestURL string // empty = use updatecheck default; overridable in tests
)

var rootCmd = &cobra.Command{
	Use:   "pathfinder",
	Short: "Static Code Analysis | Graph-First Engine | Privacy-First",
	Long:  `Code Pathfinder - Static code analysis with graph-first engine.

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

		// Update check: best-effort 800 ms fetch against the CDN manifest.
		// shouldSkipUpdateCheck is the fetch gate; shouldShowNotice is the render gate.
		// They are kept separate so PR-04 can fire analytics even when the banner is
		// suppressed by --no-banner or a non-TTY environment.
		if !shouldSkipUpdateCheck(cmd) {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			logger := output.NewLogger(output.VerbosityDefault)
			notice := updatecheck.Check(ctx, Version, "cli", updatecheck.Options{
				ManifestURL: updateCheckManifestURL,
				Logger:      logger,
			})
			noBanner, _ := cmd.Flags().GetBool("no-banner")
			if notice != nil && shouldShowNotice(logger.IsTTY(), noBanner) {
				if notice.Upgrade != nil {
					output.PrintUpdateNotice(os.Stderr, notice.Upgrade)
				}
				if notice.Announcement != nil {
					output.PrintAnnouncement(os.Stderr, notice.Announcement)
				}
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
	rootCmd.PersistentFlags().Bool("no-update-check", false,
		"Disable check for newer pathfinder versions")
}

// shouldSkipUpdateCheck returns true when the update-check fetch should be
// skipped entirely. This is the *fetch* gate — it prevents any outbound HTTP
// request. CI environments intentionally receive the update check so teams
// stay informed about newer versions.
func shouldSkipUpdateCheck(cmd *cobra.Command) bool {
	if v, _ := cmd.Flags().GetBool("no-update-check"); v {
		return true
	}
	if os.Getenv("PATHFINDER_NO_UPDATE_CHECK") != "" {
		return true
	}
	return false
}

// shouldShowNotice returns true when the update-check result should be rendered
// to the user. This is the *render* gate — it is separate from the fetch gate
// so that PR-04 analytics can record stale versions even when the banner is
// suppressed by --no-banner or a non-TTY output stream.
//
// isTTY and noBanner are passed as plain booleans so the function is
// independently testable without needing a real terminal.
func shouldShowNotice(isTTY, noBanner bool) bool {
	return isTTY && !noBanner
}

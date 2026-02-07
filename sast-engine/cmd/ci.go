package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/shivasurya/code-pathfinder/sast-engine/diff"
	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/github"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/spf13/cobra"
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "CI mode with SARIF, JSON, or CSV output for CI/CD integration",
	Long: `CI mode for integrating security scans into CI/CD pipelines.

Outputs results in SARIF, JSON, or CSV format for consumption by CI tools.

Examples:
  # Generate SARIF report with single rules file
  pathfinder ci --rules rules/owasp_top10.py --project . --output sarif > results.sarif

  # Generate SARIF report with rules directory
  pathfinder ci --rules rules/ --project . --output sarif > results.sarif

  # Generate JSON report
  pathfinder ci --rules rules/owasp_top10.py --project . --output json > results.json

  # Generate CSV report
  pathfinder ci --rules rules/owasp_top10.py --project . --output csv > results.csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
		startTime := time.Now()
		rulesPath, _ := cmd.Flags().GetString("rules")
		projectPath, _ := cmd.Flags().GetString("project")
		outputFormat, _ := cmd.Flags().GetString("output")
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")
		failOnStr, _ := cmd.Flags().GetString("fail-on")
		skipTests, _ := cmd.Flags().GetBool("skip-tests")
		baseRef, _ := cmd.Flags().GetString("base")
		headRef, _ := cmd.Flags().GetString("head")
		noDiff, _ := cmd.Flags().GetBool("no-diff")

		// GitHub PR commenting flags.
		prOpts := prCommentOptions{
			Comment: false,
			Inline:  false,
		}
		prOpts.Token, _ = cmd.Flags().GetString("github-token")
		prOpts.Repo, _ = cmd.Flags().GetString("github-repo")
		prOpts.PRNumber, _ = cmd.Flags().GetInt("github-pr")
		prOpts.Comment, _ = cmd.Flags().GetBool("pr-comment")
		prOpts.Inline, _ = cmd.Flags().GetBool("pr-inline")

		// Track CI started event (no PII, just metadata)
		analytics.ReportEventWithProperties(analytics.CIStarted, map[string]interface{}{
			"output_format": outputFormat,
			"skip_tests":    skipTests,
		})

		// Setup logger with appropriate verbosity
		verbosity := output.VerbosityDefault
		if debug {
			verbosity = output.VerbosityDebug
		} else if verbose {
			verbosity = output.VerbosityVerbose
		}
		logger := output.NewLogger(verbosity)

		// Display banner if appropriate
		noBanner, _ := cmd.Flags().GetBool("no-banner")
		if output.ShouldShowBanner(logger.IsTTY(), noBanner) {
			output.PrintBanner(logger.GetWriter(), Version, output.DefaultBannerOptions())
		} else if logger.IsTTY() && !noBanner {
			fmt.Fprintln(logger.GetWriter(), output.GetCompactBanner(Version))
		}

		// Parse and validate --fail-on severities
		failOn := output.ParseFailOn(failOnStr)
		if len(failOn) > 0 {
			if err := output.ValidateSeverities(failOn); err != nil {
				return err
			}
		}

		if rulesPath == "" {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]interface{}{
				"error_type": "validation",
				"phase":      "initialization",
			})
			return fmt.Errorf("--rules flag is required")
		}

		if projectPath == "" {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]interface{}{
				"error_type": "validation",
				"phase":      "initialization",
			})
			return fmt.Errorf("--project flag is required")
		}

		if outputFormat != "sarif" && outputFormat != "json" && outputFormat != "csv" {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]interface{}{
				"error_type": "validation",
				"phase":      "initialization",
			})
			return fmt.Errorf("--output must be 'sarif', 'json', or 'csv'")
		}

		// Validate PR commenting flags early.
		if err := prOpts.validate(); err != nil {
			return err
		}

		// Diff-aware scanning (on by default in CI mode).
		var changedFiles []string
		diffEnabled := !noDiff
		if diffEnabled {
			if baseRef == "" {
				baseRef = resolveBaseRef()
			}
			if baseRef == "" {
				logger.Progress("No baseline ref detected, running full scan")
				diffEnabled = false
			}
		}
		if diffEnabled {
			if err := diff.ValidateGitRef(projectPath, baseRef); err != nil {
				logger.Warning("Invalid base ref %q: %v (running full scan)", baseRef, err)
				diffEnabled = false
			}
		}
		if diffEnabled {
			files, err := computeChangedFiles(baseRef, headRef, projectPath, logger)
			if err != nil {
				logger.Warning("Failed to compute changed files: %v (showing all findings)", err)
				diffEnabled = false
			} else {
				changedFiles = files
			}
		}

		// Build code graph (AST)
		codeGraph := graph.Initialize(projectPath, &graph.ProgressCallbacks{
			OnStart: func(totalFiles int) {
				logger.StartProgress("Building code graph", totalFiles)
			},
			OnProgress: func() {
				logger.UpdateProgress(1)
			},
		})
		logger.FinishProgress()
		if len(codeGraph.Nodes) == 0 {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]interface{}{
				"error_type": "empty_project",
				"phase":      "graph_building",
			})
			return fmt.Errorf("no source files found in project")
		}
		logger.Statistic("Code graph built: %d nodes", len(codeGraph.Nodes))

		// Build module registry
		logger.StartProgress("Building module registry", -1)
		moduleRegistry, err := registry.BuildModuleRegistry(projectPath, skipTests)
		logger.FinishProgress()
		if err != nil {
			logger.Warning("failed to build module registry: %v", err)
			moduleRegistry = core.NewModuleRegistry()
		}
		if skipTests {
			logger.Debug("Skipping test files (use --skip-tests=false to include)")
		}

		// Build callgraph
		logger.StartProgress("Building callgraph", -1)
		cg, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectPath, logger)
		logger.FinishProgress()
		if err != nil {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]interface{}{
				"error_type": "callgraph_build",
				"phase":      "graph_building",
			})
			return fmt.Errorf("failed to build callgraph: %w", err)
		}
		logger.Statistic("Callgraph built: %d functions, %d call sites",
			len(cg.Functions), countTotalCallSites(cg))

		// Load Python DSL rules
		logger.StartProgress("Loading rules", -1)
		loader := dsl.NewRuleLoader(rulesPath)
		rules, err := loader.LoadRules(logger)
		logger.FinishProgress()
		if err != nil {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]interface{}{
				"error_type": "rule_loading",
				"phase":      "rule_loading",
			})
			return fmt.Errorf("failed to load rules: %w", err)
		}
		logger.Statistic("Loaded %d rules", len(rules))

		// Execute rules against callgraph
		logger.Progress("Running security scan...")

		// Create enricher for adding context to detections
		enricher := output.NewEnricher(cg, &output.OutputOptions{
			ProjectRoot:  projectPath,
			ContextLines: 3,
		})

		// Execute all rules and collect enriched detections
		var allEnriched []*dsl.EnrichedDetection
		allDetections := make(map[string][]dsl.DataflowDetection) // For SARIF compatibility
		var scanErrors []string
		hadErrors := false

		logger.StartProgress("Executing rules", len(rules))
		for _, rule := range rules {
			detections, err := loader.ExecuteRule(&rule, cg)
			if err != nil {
				errMsg := fmt.Sprintf("Error executing rule %s: %v", rule.Rule.ID, err)
				logger.Warning("%s", errMsg)
				scanErrors = append(scanErrors, errMsg)
				hadErrors = true
				logger.UpdateProgress(1)
				continue
			}

			if len(detections) > 0 {
				allDetections[rule.Rule.ID] = detections
				enriched, _ := enricher.EnrichAll(detections, rule)
				allEnriched = append(allEnriched, enriched...)
			}
			logger.UpdateProgress(1)
		}
		logger.FinishProgress()

		// Apply diff filter when diff-aware mode is active.
		if diffEnabled && len(changedFiles) > 0 {
			allEnriched = applyDiffFilter(allEnriched, changedFiles, logger)
		}

		logger.Statistic("Scan complete. Found %d vulnerabilities", len(allEnriched))
		logger.Progress("Generating %s output...", outputFormat)

		// Generate output
		switch outputFormat {
		case "sarif":
			scanInfo := output.ScanInfo{
				Target:        projectPath,
				RulesExecuted: len(rules),
				Errors:        scanErrors,
			}
			formatter := output.NewSARIFFormatter(nil)
			if err := formatter.Format(allEnriched, scanInfo); err != nil {
				return fmt.Errorf("failed to format SARIF output: %w", err)
			}
		case "json":
			summary := output.BuildSummary(allEnriched, len(rules))
			scanInfo := output.ScanInfo{
				Target:        projectPath,
				RulesExecuted: len(rules),
				Errors:        scanErrors,
			}
			formatter := output.NewJSONFormatter(nil)
			if err := formatter.Format(allEnriched, summary, scanInfo); err != nil {
				return fmt.Errorf("failed to format JSON output: %w", err)
			}
		case "csv":
			formatter := output.NewCSVFormatter(nil)
			if err := formatter.Format(allEnriched); err != nil {
				return fmt.Errorf("failed to format CSV output: %w", err)
			}
		default:
			return fmt.Errorf("unknown output format: %s", outputFormat)
		}

		// Post PR comments if configured.
		if prOpts.enabled() {
			metrics := github.ScanMetrics{
				FilesScanned:  len(codeGraph.Nodes),
				RulesExecuted: len(rules),
			}
			if err := postPRComments(prOpts, allEnriched, metrics, logger); err != nil {
				logger.Warning("Failed to post PR comments: %v", err)
			}
		}

		// Determine exit code based on findings and --fail-on flag
		exitCode := output.DetermineExitCode(allEnriched, failOn, hadErrors)

		// Track CI completion with results (no PII, just counts and metadata)
		severityBreakdown := make(map[string]int)
		for _, det := range allEnriched {
			severityBreakdown[det.Rule.Severity]++
		}

		analytics.ReportEventWithProperties(analytics.CICompleted, map[string]interface{}{
			"duration_ms":         time.Since(startTime).Milliseconds(),
			"rules_count":         len(rules),
			"findings_count":      len(allEnriched),
			"diff_aware":          diffEnabled,
			"diff_changed_files":  len(changedFiles),
			"severity_critical": severityBreakdown["critical"],
			"severity_high":     severityBreakdown["high"],
			"severity_medium":   severityBreakdown["medium"],
			"severity_low":      severityBreakdown["low"],
			"output_format":     outputFormat,
			"exit_code":         int(exitCode),
			"had_errors":        hadErrors,
		})

		if exitCode != output.ExitCodeSuccess {
			osExit(int(exitCode))
		}

		return nil
	},
}



// Variable to allow mocking os.Exit in tests.
var osExit = os.Exit

func init() {
	rootCmd.AddCommand(ciCmd)
	ciCmd.Flags().StringP("rules", "r", "", "Path to Python DSL rules file or directory (required)")
	ciCmd.Flags().StringP("project", "p", "", "Path to project directory to scan (required)")
	ciCmd.Flags().StringP("output", "o", "sarif", "Output format: sarif or json (default: sarif)")
	ciCmd.Flags().BoolP("verbose", "v", false, "Show statistics and timing information")
	ciCmd.Flags().Bool("debug", false, "Show detailed debug diagnostics with file-level progress and timestamps")
	ciCmd.Flags().String("fail-on", "", "Fail with exit code 1 if findings match severities (e.g., critical,high)")
	ciCmd.Flags().Bool("skip-tests", true, "Skip test files (test_*.py, *_test.py, conftest.py, etc.)")
	ciCmd.Flags().String("base", "", "Base git ref for diff-aware scanning (auto-detected in CI)")
	ciCmd.Flags().String("head", "HEAD", "Head git ref for diff-aware scanning")
	ciCmd.Flags().Bool("no-diff", false, "Disable diff-aware scanning (scan all files)")
	ciCmd.Flags().String("github-token", "", "GitHub API token for posting PR comments")
	ciCmd.Flags().String("github-repo", "", "GitHub repository in owner/repo format")
	ciCmd.Flags().Int("github-pr", 0, "Pull request number for posting comments")
	ciCmd.Flags().Bool("pr-comment", false, "Post summary comment on the pull request")
	ciCmd.Flags().Bool("pr-inline", false, "Post inline review comments for critical/high findings")
	ciCmd.MarkFlagRequired("rules")
	ciCmd.MarkFlagRequired("project")
}

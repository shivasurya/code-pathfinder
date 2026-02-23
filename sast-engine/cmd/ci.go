package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/shivasurya/code-pathfinder/sast-engine/diff"
	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/github"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/spf13/cobra"
)

// prFlags holds the CLI flags for PR commenting.
type prFlags struct {
	Token    string
	Repo     string // "owner/repo" format
	PRNumber int
	Comment  bool
	Inline   bool
}

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

  # Use remote rulesets
  pathfinder ci --ruleset python/django --ruleset docker/security --project . --output sarif

  # Write output to file
  pathfinder ci --ruleset docker/security --project . --output sarif --output-file results.sarif

  # Generate JSON report
  pathfinder ci --rules rules/owasp_top10.py --project . --output json > results.json

  # Generate CSV report
  pathfinder ci --rules rules/owasp_top10.py --project . --output csv > results.csv

  # Post PR comments on GitHub
  pathfinder ci --ruleset python/django --project . --output sarif \
    --github-token $GITHUB_TOKEN --github-repo owner/repo --github-pr 42 \
    --pr-comment --pr-inline`,
	RunE: func(cmd *cobra.Command, args []string) error {
		startTime := time.Now()
		rulesPath, _ := cmd.Flags().GetString("rules")
		rulesetSpecs, _ := cmd.Flags().GetStringArray("ruleset")
		refreshRules, _ := cmd.Flags().GetBool("refresh-rules")
		projectPath, _ := cmd.Flags().GetString("project")
		outputFormat, _ := cmd.Flags().GetString("output")
		outputFile, _ := cmd.Flags().GetString("output-file")
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")
		failOnStr, _ := cmd.Flags().GetString("fail-on")
		skipTests, _ := cmd.Flags().GetBool("skip-tests")
		baseRef, _ := cmd.Flags().GetString("base")
		headRef, _ := cmd.Flags().GetString("head")
		noDiff, _ := cmd.Flags().GetBool("no-diff")

		// GitHub PR commenting flags.
		var prOpts prFlags
		prOpts.Token, _ = cmd.Flags().GetString("github-token")
		prOpts.Repo, _ = cmd.Flags().GetString("github-repo")
		prOpts.PRNumber, _ = cmd.Flags().GetInt("github-pr")
		prOpts.Comment, _ = cmd.Flags().GetBool("pr-comment")
		prOpts.Inline, _ = cmd.Flags().GetBool("pr-inline")

		// Track CI started event (no PII, just metadata)
		analytics.ReportEventWithProperties(analytics.CIStarted, map[string]any{
			"output_format":     outputFormat,
			"skip_tests":        skipTests,
			"has_local_rules":   rulesPath != "",
			"has_remote_rules":  len(rulesetSpecs) > 0,
			"remote_rule_count": len(rulesetSpecs),
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

		if rulesPath == "" && len(rulesetSpecs) == 0 {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]any{
				"error_type": "validation",
				"phase":      "initialization",
			})
			return fmt.Errorf("either --rules or --ruleset flag is required")
		}

		if projectPath == "" {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]any{
				"error_type": "validation",
				"phase":      "initialization",
			})
			return fmt.Errorf("--project flag is required")
		}

		if outputFormat != "sarif" && outputFormat != "json" && outputFormat != "csv" {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]any{
				"error_type": "validation",
				"phase":      "initialization",
			})
			return fmt.Errorf("--output must be 'sarif', 'json', or 'csv'")
		}

		// Validate PR commenting flags early.
		if prOpts.Comment || prOpts.Inline {
			if prOpts.Token == "" {
				return fmt.Errorf("--github-token is required for PR commenting")
			}
			if prOpts.Repo == "" {
				return fmt.Errorf("--github-repo is required for PR commenting")
			}
			if prOpts.PRNumber <= 0 {
				return fmt.Errorf("--github-pr must be a positive number")
			}
			if _, _, err := github.ParseRepo(prOpts.Repo); err != nil {
				return err
			}
		}

		// Handle remote ruleset downloads and merge with local rules.
		finalRulesPath, tempDir, err := prepareRules(rulesPath, rulesetSpecs, refreshRules, logger)
		if err != nil {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]any{
				"error_type": "rule_preparation",
				"phase":      "initialization",
			})
			return fmt.Errorf("failed to prepare rules: %w", err)
		}
		if tempDir != "" {
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					logger.Warning("Failed to clean up temporary directory: %v", err)
				}
			}()
		}
		rulesPath = finalRulesPath

		// Diff-aware scanning (on by default in CI mode).
		var changedFiles []string
		diffEnabled := !noDiff
		if diffEnabled {
			if baseRef == "" {
				baseRef = diff.ResolveBaseRef()
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
			files, err := diff.ComputeChangedFiles(baseRef, headRef, projectPath)
			if err != nil {
				logger.Warning("Failed to compute changed files: %v (showing all findings)", err)
				diffEnabled = false
			} else {
				changedFiles = files
				logger.Progress("Changed files: %d", len(changedFiles))
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
			logger.Progress("No source files found in project")
		} else {
			logger.Statistic("Code graph built: %d nodes", len(codeGraph.Nodes))
		}

		// Execute container rules if Docker/Compose files are present.
		loader := dsl.NewRuleLoader(rulesPath)
		var containerDetections []*dsl.EnrichedDetection
		var containerRulesCount int
		dockerFiles, composeFiles := extractContainerFiles(codeGraph)
		if len(dockerFiles) > 0 || len(composeFiles) > 0 {
			logger.Progress("Found %d Dockerfile(s) and %d docker-compose file(s)", len(dockerFiles), len(composeFiles))
			logger.Progress("Loading container rules...")
			containerRulesJSON, err := loader.LoadContainerRules(logger)
			if err == nil {
				logger.Progress("Executing container rules...")
				containerDetections = executeContainerRules(containerRulesJSON, dockerFiles, composeFiles, projectPath, logger)
				containerRulesCount = countContainerRules(containerRulesJSON)
				if len(containerDetections) > 0 {
					logger.Statistic("Container scan found %d issue(s)", len(containerDetections))
				} else {
					logger.Progress("No container issues detected")
				}
			} else {
				logger.Debug("Container rule loading failed: %v", err)
			}
		}

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
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]any{
				"error_type": "callgraph_build",
				"phase":      "graph_building",
			})
			return fmt.Errorf("failed to build callgraph: %w", err)
		}
		logger.Statistic("Callgraph built: %d functions, %d call sites",
			len(cg.Functions), countTotalCallSites(cg))

		// Build Go call graph if go.mod exists
		goModPath := filepath.Join(projectPath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			logger.Debug("Detected go.mod, building Go call graph...")

			goRegistry, err := resolution.BuildGoModuleRegistry(projectPath)
			if err != nil {
				logger.Warning("Failed to build Go module registry: %v", err)
			} else {
				// Initialize Go type inference engine for Phase 2 type tracking
				goTypeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)

				goCG, err := builder.BuildGoCallGraph(codeGraph, goRegistry, goTypeEngine)
				if err != nil {
					logger.Warning("Failed to build Go call graph: %v", err)
				} else {
					builder.MergeCallGraphs(cg, goCG)
					logger.Statistic("Go call graph merged: %d functions, %d call sites",
						len(goCG.Functions), countTotalCallSites(goCG))
				}
			}
		}

		// Load Python DSL rules
		logger.StartProgress("Loading rules", -1)
		rules, err := loader.LoadRules(logger)
		logger.FinishProgress()
		if err != nil {
			analytics.ReportEventWithProperties(analytics.CIFailed, map[string]any{
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

		// Merge container detections with code analysis detections.
		allEnriched = append(allEnriched, containerDetections...)

		// Apply diff filter when diff-aware mode is active.
		if diffEnabled && len(changedFiles) > 0 {
			totalBefore := len(allEnriched)
			diffFilter := output.NewDiffFilter(changedFiles)
			allEnriched = diffFilter.Filter(allEnriched)
			logger.Progress("Diff filter: %d/%d findings in changed files", len(allEnriched), totalBefore)
		}

		// Total rules = code analysis rules loaded + container rules loaded.
		totalRules := len(rules) + containerRulesCount

		// Count unique source files. When diff-aware, only count changed files.
		var filesScanned int
		if diffEnabled && len(changedFiles) > 0 {
			filesScanned = len(changedFiles)
		} else {
			uniqueFiles := make(map[string]bool)
			for _, node := range codeGraph.Nodes {
				if node.File != "" {
					uniqueFiles[node.File] = true
				}
			}
			filesScanned = len(uniqueFiles)
		}

		logger.Statistic("Scan complete. Found %d vulnerabilities", len(allEnriched))
		logger.Progress("Generating %s output...", outputFormat)

		// Setup output writer (file or stdout).
		var outputWriter *os.File
		if outputFile != "" {
			outputWriter, err = os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer outputWriter.Close()
			logger.Progress("Writing output to %s", outputFile)
		}

		// Generate output.
		switch outputFormat {
		case "sarif":
			scanInfo := output.ScanInfo{
				Target:        projectPath,
				RulesExecuted: totalRules,
				Errors:        scanErrors,
			}
			var formatter *output.SARIFFormatter
			if outputWriter != nil {
				formatter = output.NewSARIFFormatterWithWriter(outputWriter, nil)
			} else {
				formatter = output.NewSARIFFormatter(nil)
			}
			if err := formatter.Format(allEnriched, scanInfo); err != nil {
				return fmt.Errorf("failed to format SARIF output: %w", err)
			}
		case "json":
			summary := output.BuildSummary(allEnriched, totalRules)
			scanInfo := output.ScanInfo{
				Target:        projectPath,
				RulesExecuted: totalRules,
				Errors:        scanErrors,
			}
			var formatter *output.JSONFormatter
			if outputWriter != nil {
				formatter = output.NewJSONFormatterWithWriter(outputWriter, nil)
			} else {
				formatter = output.NewJSONFormatter(nil)
			}
			if err := formatter.Format(allEnriched, summary, scanInfo); err != nil {
				return fmt.Errorf("failed to format JSON output: %w", err)
			}
		case "csv":
			var formatter *output.CSVFormatter
			if outputWriter != nil {
				formatter = output.NewCSVFormatterWithWriter(outputWriter, nil)
			} else {
				formatter = output.NewCSVFormatter(nil)
			}
			if err := formatter.Format(allEnriched); err != nil {
				return fmt.Errorf("failed to format CSV output: %w", err)
			}
		default:
			return fmt.Errorf("unknown output format: %s", outputFormat)
		}

		if outputWriter != nil {
			logger.Progress("Successfully wrote results to %s", outputFile)
		}

		// Post PR comments if configured.
		if prOpts.Comment || prOpts.Inline {
			owner, repo, _ := github.ParseRepo(prOpts.Repo) // Already validated.
			client := github.NewClient(prOpts.Token, owner, repo)
			ghOpts := github.PRCommentOptions{
				PRNumber: prOpts.PRNumber,
				Comment:  prOpts.Comment,
				Inline:   prOpts.Inline,
			}
			metrics := github.ScanMetrics{
				FilesScanned:  filesScanned,
				RulesExecuted: totalRules,
			}
			if err := github.PostPRComments(client, ghOpts, allEnriched, metrics, logger.Progress); err != nil {
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

		analytics.ReportEventWithProperties(analytics.CICompleted, map[string]any{
			"duration_ms":        time.Since(startTime).Milliseconds(),
			"rules_count":        totalRules,
			"findings_count":     len(allEnriched),
			"diff_aware":         diffEnabled,
			"diff_changed_files": len(changedFiles),
			"severity_critical":  severityBreakdown["critical"],
			"severity_high":      severityBreakdown["high"],
			"severity_medium":    severityBreakdown["medium"],
			"severity_low":       severityBreakdown["low"],
			"output_format":      outputFormat,
			"exit_code":          int(exitCode),
			"had_errors":         hadErrors,
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
	ciCmd.Flags().StringP("rules", "r", "", "Path to Python DSL rules file or directory")
	ciCmd.Flags().StringArray("ruleset", []string{}, "Ruleset bundle (e.g., docker/security) or individual rule ID (e.g., docker/DOCKER-BP-007). Can be specified multiple times.")
	ciCmd.Flags().Bool("refresh-rules", false, "Force refresh of cached rulesets")
	ciCmd.Flags().StringP("project", "p", "", "Path to project directory to scan (required)")
	ciCmd.Flags().StringP("output", "o", "sarif", "Output format: sarif, json, or csv (default: sarif)")
	ciCmd.Flags().StringP("output-file", "f", "", "Write output to file instead of stdout")
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
	ciCmd.MarkFlagRequired("project")
}

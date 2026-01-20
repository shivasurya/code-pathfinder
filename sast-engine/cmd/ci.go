package cmd

import (
	"fmt"
	"os"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
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
		rulesPath, _ := cmd.Flags().GetString("rules")
		projectPath, _ := cmd.Flags().GetString("project")
		outputFormat, _ := cmd.Flags().GetString("output")
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")
		failOnStr, _ := cmd.Flags().GetString("fail-on")
		skipTests, _ := cmd.Flags().GetBool("skip-tests")

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
			return fmt.Errorf("--rules flag is required")
		}

		if projectPath == "" {
			return fmt.Errorf("--project flag is required")
		}

		if outputFormat != "sarif" && outputFormat != "json" && outputFormat != "csv" {
			return fmt.Errorf("--output must be 'sarif', 'json', or 'csv'")
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

		// Determine exit code based on findings and --fail-on flag
		exitCode := output.DetermineExitCode(allEnriched, failOn, hadErrors)
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
	ciCmd.MarkFlagRequired("rules")
	ciCmd.MarkFlagRequired("project")
}

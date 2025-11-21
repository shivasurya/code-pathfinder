package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	sarif "github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/dsl"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/output"
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

		// Setup logger with appropriate verbosity
		verbosity := output.VerbosityDefault
		if debug {
			verbosity = output.VerbosityDebug
		} else if verbose {
			verbosity = output.VerbosityVerbose
		}
		logger := output.NewLogger(verbosity)

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
		logger.Progress("Building code graph from %s...", projectPath)
		codeGraph := graph.Initialize(projectPath)
		if len(codeGraph.Nodes) == 0 {
			return fmt.Errorf("no source files found in project")
		}
		logger.Statistic("Code graph built: %d nodes", len(codeGraph.Nodes))

		// Build module registry
		logger.Progress("Building module registry...")
		moduleRegistry, err := registry.BuildModuleRegistry(projectPath)
		if err != nil {
			logger.Warning("failed to build module registry: %v", err)
			moduleRegistry = core.NewModuleRegistry()
		}

		// Build callgraph
		logger.Progress("Building callgraph...")
		cg, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectPath, logger)
		if err != nil {
			return fmt.Errorf("failed to build callgraph: %w", err)
		}
		logger.Statistic("Callgraph built: %d functions, %d call sites",
			len(cg.Functions), countTotalCallSites(cg))

		// Load Python DSL rules
		logger.Progress("Loading rules from %s...", rulesPath)
		loader := dsl.NewRuleLoader(rulesPath)
		rules, err := loader.LoadRules()
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

		for _, rule := range rules {
			detections, err := loader.ExecuteRule(&rule, cg)
			if err != nil {
				errMsg := fmt.Sprintf("Error executing rule %s: %v", rule.Rule.ID, err)
				logger.Warning("%s", errMsg)
				scanErrors = append(scanErrors, errMsg)
				continue
			}

			if len(detections) > 0 {
				allDetections[rule.Rule.ID] = detections
				enriched, _ := enricher.EnrichAll(detections, rule)
				allEnriched = append(allEnriched, enriched...)
			}
		}

		logger.Statistic("Scan complete. Found %d vulnerabilities", len(allEnriched))
		logger.Progress("Generating %s output...", outputFormat)

		// Generate output
		switch outputFormat {
		case "sarif":
			return generateSARIFOutput(rules, allDetections)
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
			if len(allEnriched) > 0 {
				osExit(1)
			}
			return nil
		case "csv":
			formatter := output.NewCSVFormatter(nil)
			if err := formatter.Format(allEnriched); err != nil {
				return fmt.Errorf("failed to format CSV output: %w", err)
			}
			if len(allEnriched) > 0 {
				osExit(1)
			}
			return nil
		default:
			return fmt.Errorf("unknown output format: %s", outputFormat)
		}
	},
}

func generateSARIFOutput(rules []dsl.RuleIR, allDetections map[string][]dsl.DataflowDetection) error {
	report, err := sarif.New(sarif.Version210)
	if err != nil {
		return fmt.Errorf("failed to create SARIF report: %w", err)
	}

	run := sarif.NewRunWithInformationURI("Code Pathfinder", "https://github.com/shivasurya/code-pathfinder")

	// Add all rules to the run
	for _, rule := range rules {
		// Create full description with CWE and OWASP info
		fullDesc := rule.Rule.Description
		if rule.Rule.CWE != "" || rule.Rule.OWASP != "" {
			fullDesc += " ("
			if rule.Rule.CWE != "" {
				fullDesc += rule.Rule.CWE
			}
			if rule.Rule.OWASP != "" {
				if rule.Rule.CWE != "" {
					fullDesc += ", "
				}
				fullDesc += rule.Rule.OWASP
			}
			fullDesc += ")"
		}

		sarifRule := run.AddRule(rule.Rule.ID).
			WithDescription(fullDesc).
			WithName(rule.Rule.Name)

		// Map severity to SARIF level
		level := "warning"
		switch rule.Rule.Severity {
		case "critical", "high":
			level = "error"
		case "medium":
			level = "warning"
		case "low":
			level = "note"
		}
		sarifRule.WithDefaultConfiguration(sarif.NewReportingConfiguration().WithLevel(level))
	}

	// Add detections as results
	for _, rule := range rules {
		detections, ok := allDetections[rule.Rule.ID]
		if !ok {
			continue
		}

		for _, detection := range detections {
			// Create detailed message
			message := fmt.Sprintf("%s in %s", rule.Rule.Description, detection.FunctionFQN)
			if detection.SinkCall != "" {
				message += fmt.Sprintf(" (sink: %s, confidence: %.0f%%)", detection.SinkCall, detection.Confidence*100)
			}

			result := run.CreateResultForRule(rule.Rule.ID).
				WithMessage(sarif.NewTextMessage(message))

			// Add location
			if detection.FunctionFQN != "" {
				location := sarif.NewLocation().
					WithPhysicalLocation(
						sarif.NewPhysicalLocation().
							WithRegion(
								sarif.NewRegion().
									WithStartLine(detection.SinkLine).
									WithEndLine(detection.SinkLine),
							),
					)

				result.AddLocation(location)
			}

			// Note: Additional detection info (functionFQN, sinkCall, etc.) is included in the message
			// SARIF v2 spec doesn't have a straightforward way to add custom properties to results
		}
	}

	report.AddRun(run)

	// Write to stdout
	sarifJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF: %w", err)
	}

	fmt.Println(string(sarifJSON))
	return nil
}


// Variable to allow mocking os.Exit in tests.
var osExit = os.Exit

func init() {
	rootCmd.AddCommand(ciCmd)
	ciCmd.Flags().StringP("rules", "r", "", "Path to Python DSL rules file or directory (required)")
	ciCmd.Flags().StringP("project", "p", "", "Path to project directory to scan (required)")
	ciCmd.Flags().StringP("output", "o", "sarif", "Output format: sarif or json (default: sarif)")
	ciCmd.Flags().BoolP("verbose", "v", false, "Show progress and statistics")
	ciCmd.Flags().Bool("debug", false, "Show debug diagnostics with timestamps")
	ciCmd.MarkFlagRequired("rules")
	ciCmd.MarkFlagRequired("project")
}

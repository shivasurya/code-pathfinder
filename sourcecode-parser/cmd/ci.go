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
	Short: "CI mode with SARIF or JSON output for CI/CD integration",
	Long: `CI mode for integrating security scans into CI/CD pipelines.

Outputs results in SARIF or JSON format for consumption by CI tools.

Examples:
  # Generate SARIF report with single rules file
  pathfinder ci --rules rules/owasp_top10.py --project . --output sarif > results.sarif

  # Generate SARIF report with rules directory
  pathfinder ci --rules rules/ --project . --output sarif > results.sarif

  # Generate JSON report
  pathfinder ci --rules rules/owasp_top10.py --project . --output json > results.json`,
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

		if outputFormat != "sarif" && outputFormat != "json" {
			return fmt.Errorf("--output must be 'sarif' or 'json'")
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
		cg, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectPath)
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
		allDetections := make(map[string][]dsl.DataflowDetection)
		totalDetections := 0
		for _, rule := range rules {
			detections, err := loader.ExecuteRule(&rule, cg)
			if err != nil {
				logger.Error("executing rule %s: %v", rule.Rule.ID, err)
				continue
			}

			if len(detections) > 0 {
				allDetections[rule.Rule.ID] = detections
				totalDetections += len(detections)
			}
		}

		logger.Statistic("Scan complete. Found %d vulnerabilities", totalDetections)
		logger.Progress("Generating %s output...", outputFormat)

		// Generate output
		if outputFormat == "sarif" {
			return generateSARIFOutput(rules, allDetections)
		}
		return generateJSONOutput(rules, allDetections)
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

func generateJSONOutput(rules []dsl.RuleIR, allDetections map[string][]dsl.DataflowDetection) error {
	output := make(map[string]interface{})
	output["tool"] = "Code Pathfinder"
	output["version"] = Version

	results := []map[string]interface{}{}
	for _, rule := range rules {
		detections, ok := allDetections[rule.Rule.ID]
		if !ok {
			continue
		}

		for _, detection := range detections {
			result := map[string]interface{}{
				"ruleId":      rule.Rule.ID,
				"ruleName":    rule.Rule.Name,
				"severity":    rule.Rule.Severity,
				"cwe":         rule.Rule.CWE,
				"owasp":       rule.Rule.OWASP,
				"description": rule.Rule.Description,
				"functionFQN": detection.FunctionFQN,
				"sinkLine":    detection.SinkLine,
				"sinkCall":    detection.SinkCall,
				"scope":       detection.Scope,
				"confidence":  detection.Confidence,
			}

			if detection.SourceLine > 0 {
				result["sourceLine"] = detection.SourceLine
			}

			if detection.TaintedVar != "" {
				result["taintedVar"] = detection.TaintedVar
			}

			results = append(results, result)
		}
	}

	output["results"] = results
	output["summary"] = map[string]interface{}{
		"totalVulnerabilities": len(results),
		"rulesExecuted":        len(rules),
	}

	jsonOutput, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonOutput))

	// Exit with error code if vulnerabilities found
	if len(results) > 0 {
		osExit(1)
	}

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

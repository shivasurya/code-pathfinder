package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/executor"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/docker"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan code for security vulnerabilities using Python DSL rules",
	Long: `Scan codebase using Python DSL security rules.

Examples:
  # Scan with a single rules file
  pathfinder scan --rules rules/owasp_top10.py --project /path/to/project

  # Scan with a directory of rules
  pathfinder scan --rules rules/ --project /path/to/project

  # Scan with custom rules and output to JSON file
  pathfinder scan --rules my_rules.py --project . --output json --output-file results.json

  # Scan with SARIF output for CI/CD integration
  pathfinder scan --rules rules/ --project . --output sarif --output-file results.sarif

  # Scan and print JSON to stdout (for piping)
  pathfinder scan --rules rules/ --project . --output json | jq .`,
	// Note: The main RunE logic is covered by integration tests in exit_code_integration_test.go
	// Unit testing cobra commands requires complex mocking of file systems, graph building, etc.
	// Integration tests provide better coverage for the full execution path.
	RunE: func(cmd *cobra.Command, args []string) error {
		rulesPath, _ := cmd.Flags().GetString("rules")
		projectPath, _ := cmd.Flags().GetString("project")
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")
		failOnStr, _ := cmd.Flags().GetString("fail-on")
		outputFormat, _ := cmd.Flags().GetString("output")
		outputFile, _ := cmd.Flags().GetString("output-file")

		// Setup logger with appropriate verbosity
		verbosity := output.VerbosityDefault
		if debug {
			verbosity = output.VerbosityDebug
		} else if verbose {
			verbosity = output.VerbosityVerbose
		}
		logger := output.NewLogger(verbosity)

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

		if outputFormat != "" && outputFormat != "text" && outputFormat != "json" && outputFormat != "sarif" && outputFormat != "csv" {
			return fmt.Errorf("--output must be 'text', 'json', 'sarif', or 'csv'")
		}

		// Convert project path to absolute path to ensure consistency
		absProjectPath, err := filepath.Abs(projectPath)
		if err != nil {
			return fmt.Errorf("failed to resolve project path: %w", err)
		}
		projectPath = absProjectPath

		// Create rule loader (used for both container and code analysis rules)
		loader := dsl.NewRuleLoader(rulesPath)

		// Step 1: Build code graph (AST)
		logger.Progress("Building code graph from %s...", projectPath)
		codeGraph := graph.Initialize(projectPath)
		if len(codeGraph.Nodes) == 0 {
			return fmt.Errorf("no source files found in project")
		}
		logger.Statistic("Code graph built: %d nodes", len(codeGraph.Nodes))

		// Step 1.5: Execute container rules if Docker/Compose files are present
		var containerDetections []*dsl.EnrichedDetection
		dockerFiles, composeFiles := extractContainerFiles(codeGraph)
		if len(dockerFiles) > 0 || len(composeFiles) > 0 {
			logger.Progress("Found %d Dockerfile(s) and %d docker-compose file(s)", len(dockerFiles), len(composeFiles))

			// Load container rules from the same rules path (runtime generation)
			logger.Progress("Loading container rules...")
			containerRulesJSON, err := loader.LoadContainerRules()
			if err != nil {
				logger.Warning("No container rules found: %v", err)
			} else {
				logger.Progress("Executing container rules...")
				containerDetections = executeContainerRules(containerRulesJSON, dockerFiles, composeFiles, projectPath, logger)
				if len(containerDetections) > 0 {
					logger.Statistic("Container scan found %d issue(s)", len(containerDetections))
				} else {
					logger.Progress("No container issues detected")
				}
			}
		}

		// Step 2: Build module registry
		logger.Progress("Building module registry...")
		moduleRegistry, err := registry.BuildModuleRegistry(projectPath)
		if err != nil {
			logger.Warning("failed to build module registry: %v", err)
			// Create empty registry as fallback
			moduleRegistry = core.NewModuleRegistry()
		}

		// Step 3: Build callgraph
		logger.Progress("Building callgraph...")
		cg, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectPath, logger)
		if err != nil {
			return fmt.Errorf("failed to build callgraph: %w", err)
		}
		logger.Statistic("Callgraph built: %d functions, %d call sites",
			len(cg.Functions), countTotalCallSites(cg))

		// Step 4: Load Python DSL rules
		logger.Progress("Loading rules from %s...", rulesPath)
		rules, err := loader.LoadRules()
		if err != nil {
			return fmt.Errorf("failed to load rules: %w", err)
		}
		logger.Statistic("Loaded %d rules", len(rules))

		// Step 5: Execute rules against callgraph
		logger.Progress("Running security scan...")

		// Create enricher for adding context to detections
		enricher := output.NewEnricher(cg, &output.OutputOptions{
			ProjectRoot:  projectPath,
			ContextLines: 3,
			Verbosity:    verbosity,
		})

		// Execute all rules and collect enriched detections
		var allEnriched []*dsl.EnrichedDetection
		var scanErrors bool
		for _, rule := range rules {
			detections, err := loader.ExecuteRule(&rule, cg)
			if err != nil {
				logger.Warning("Error executing rule %s: %v", rule.Rule.ID, err)
				scanErrors = true
				continue
			}

			if len(detections) > 0 {
				enriched, _ := enricher.EnrichAll(detections, rule)
				allEnriched = append(allEnriched, enriched...)
			}
		}

		// Merge container detections with code analysis detections
		allEnriched = append(allEnriched, containerDetections...)

		// Step 6: Format and display results
		// Count unique rule IDs from all detections (includes both code and container rules)
		uniqueRules := make(map[string]bool)
		for _, det := range allEnriched {
			uniqueRules[det.Rule.ID] = true
		}
		summary := output.BuildSummary(allEnriched, len(uniqueRules))

		// Default to text format if not specified
		if outputFormat == "" {
			outputFormat = "text"
		}

		logger.Progress("Generating %s output...", outputFormat)

		// Setup output writer (file or stdout)
		var outputWriter *os.File
		if outputFile != "" {
			var err error
			outputWriter, err = os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer outputWriter.Close()
			logger.Progress("Writing output to %s", outputFile)
		}

		// Generate output based on format
		switch outputFormat {
		case "text":
			formatter := output.NewTextFormatter(&output.OutputOptions{
				Verbosity: verbosity,
			}, logger)
			if err := formatter.Format(allEnriched, summary); err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}
		case "json":
			scanInfo := output.ScanInfo{
				Target:        projectPath,
				RulesExecuted: len(uniqueRules),
				Errors:        []string{},
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
		case "sarif":
			scanInfo := output.ScanInfo{
				Target:        projectPath,
				RulesExecuted: len(uniqueRules),
				Errors:        []string{},
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

		// Determine exit code based on findings and --fail-on flag
		exitCode := output.DetermineExitCode(allEnriched, failOn, scanErrors)
		if exitCode != output.ExitCodeSuccess {
			os.Exit(int(exitCode))
		}

		return nil
	},
}

func countTotalCallSites(cg *core.CallGraph) int {
	total := 0
	for _, sites := range cg.CallSites {
		total += len(sites)
	}
	return total
}

// extractContainerFiles extracts unique Docker and docker-compose file paths from CodeGraph.
func extractContainerFiles(codeGraph *graph.CodeGraph) (dockerFiles []string, composeFiles []string) {
	dockerFileSet := make(map[string]bool)
	composeFileSet := make(map[string]bool)

	for _, node := range codeGraph.Nodes {
		if node.Type == "dockerfile_instruction" {
			dockerFileSet[node.File] = true
		} else if node.Type == "compose_service" {
			composeFileSet[node.File] = true
		}
	}

	for file := range dockerFileSet {
		dockerFiles = append(dockerFiles, file)
	}
	for file := range composeFileSet {
		composeFiles = append(composeFiles, file)
	}

	return dockerFiles, composeFiles
}

// executeContainerRules executes container security rules and returns enriched detections.
func executeContainerRules(
	rulesJSON []byte,
	dockerFiles []string,
	composeFiles []string,
	projectPath string,
	logger *output.Logger,
) []*dsl.EnrichedDetection {
	// Create executor and load rules
	exec := &executor.ContainerRuleExecutor{}
	if err := exec.LoadRules(rulesJSON); err != nil {
		logger.Warning("Failed to parse container rules: %v", err)
		return nil
	}

	var allMatches []executor.RuleMatch

	// Execute rules on Dockerfiles
	for _, dockerFilePath := range dockerFiles {
		parser := docker.NewDockerfileParser()
		dockerGraph, err := parser.ParseFile(dockerFilePath)
		if err != nil {
			logger.Warning("Failed to parse Dockerfile %s: %v", dockerFilePath, err)
			continue
		}

		matches := exec.ExecuteDockerfile(dockerGraph)
		allMatches = append(allMatches, matches...)
	}

	// Execute rules on docker-compose files
	for _, composeFilePath := range composeFiles {
		composeGraph, err := graph.ParseDockerCompose(composeFilePath)
		if err != nil {
			logger.Warning("Failed to parse docker-compose %s: %v", composeFilePath, err)
			continue
		}

		matches := exec.ExecuteCompose(composeGraph)
		allMatches = append(allMatches, matches...)
	}

	// Convert RuleMatch to EnrichedDetection
	enriched := make([]*dsl.EnrichedDetection, 0, len(allMatches))
	for _, match := range allMatches {
		// Make file path relative to project root
		relPath, err := filepath.Rel(projectPath, match.FilePath)
		if err != nil {
			relPath = match.FilePath
		}

		// Build description with service name if present (compose rules)
		description := match.Message
		if match.ServiceName != "" {
			description = fmt.Sprintf("[Service: %s] %s", match.ServiceName, match.Message)
		}

		// Parse CWE into slice format
		cweList := []string{}
		if match.CWE != "" {
			cweList = []string{match.CWE}
		}

		// Generate code snippet
		snippet := generateCodeSnippet(match.FilePath, match.LineNumber, 3)

		detection := &dsl.EnrichedDetection{
			Detection: dsl.DataflowDetection{
				FunctionFQN: match.FilePath, // Use file path as function identifier for container rules
				SinkLine:    match.LineNumber,
				Confidence:  1.0, // Container rules are deterministic
				Scope:       "file",
			},
			Location: dsl.LocationInfo{
				FilePath: match.FilePath,
				RelPath:  relPath,
				Line:     match.LineNumber,
			},
			Snippet: snippet,
			Rule: dsl.RuleMetadata{
				ID:          match.RuleID,
				Name:        match.RuleName,
				Severity:    strings.ToLower(match.Severity), // Normalize to lowercase for formatter
				Description: description,
				CWE:         cweList,
			},
			DetectionType: dsl.DetectionTypePattern,
		}

		enriched = append(enriched, detection)
	}

	return enriched
}

// generateCodeSnippet creates a code snippet with context lines around the target line.
func generateCodeSnippet(filePath string, lineNumber int, contextLines int) dsl.CodeSnippet {
	// Read file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		return dsl.CodeSnippet{}
	}

	lines := splitLines(string(content))
	if lineNumber < 1 || lineNumber > len(lines) {
		return dsl.CodeSnippet{}
	}

	// Calculate start and end lines (1-indexed)
	startLine := lineNumber - contextLines
	if startLine < 1 {
		startLine = 1
	}
	endLine := lineNumber + contextLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// Build snippet lines
	var snippetLines []dsl.SnippetLine
	for i := startLine; i <= endLine; i++ {
		snippetLines = append(snippetLines, dsl.SnippetLine{
			Number:      i,
			Content:     lines[i-1], // lines is 0-indexed
			IsHighlight: i == lineNumber,
		})
	}

	return dsl.CodeSnippet{
		Lines:         snippetLines,
		StartLine:     startLine,
		HighlightLine: lineNumber,
	}
}

// splitLines splits content into lines preserving empty lines.
func splitLines(content string) []string {
	if content == "" {
		return []string{}
	}
	// Split by newline but preserve empty lines
	lines := []string{}
	currentLine := ""
	for _, ch := range content {
		if ch == '\n' {
			lines = append(lines, currentLine)
			currentLine = ""
		} else if ch != '\r' { // Skip carriage returns
			currentLine += string(ch)
		}
	}
	// Add last line if not empty or if content doesn't end with newline
	if currentLine != "" || len(content) > 0 && content[len(content)-1] != '\n' {
		lines = append(lines, currentLine)
	}
	return lines
}

// printDetections outputs detections in simple format (used by query command).
func printDetections(rule dsl.RuleIR, detections []dsl.DataflowDetection) {
	fmt.Printf("\n[%s] %s (%s)\n", rule.Rule.Severity, rule.Rule.ID, rule.Rule.Name)
	fmt.Printf("  CWE: %s | OWASP: %s\n", rule.Rule.CWE, rule.Rule.OWASP)
	fmt.Printf("  %s\n", rule.Rule.Description)

	for _, detection := range detections {
		fmt.Printf("\n  → %s:%d\n", detection.FunctionFQN, detection.SinkLine)
		if detection.SourceLine > 0 {
			fmt.Printf("    Source: line %d\n", detection.SourceLine)
		}
		if detection.SinkCall != "" {
			fmt.Printf("    Sink: %s (line %d)\n", detection.SinkCall, detection.SinkLine)
		}
		if detection.TaintedVar != "" {
			fmt.Printf("    Tainted variable: %s\n", detection.TaintedVar)
		}
		fmt.Printf("    Confidence: %.0f%%\n", detection.Confidence*100)
		fmt.Printf("    Scope: %s\n", detection.Scope)
	}
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringP("rules", "r", "", "Path to Python DSL rules file or directory (required)")
	scanCmd.Flags().StringP("project", "p", "", "Path to project directory to scan (required)")
	scanCmd.Flags().StringP("output", "o", "text", "Output format: text, json, sarif, or csv (default: text)")
	scanCmd.Flags().StringP("output-file", "f", "", "Write output to file instead of stdout")
	scanCmd.Flags().BoolP("verbose", "v", false, "Show progress and statistics")
	scanCmd.Flags().Bool("debug", false, "Show debug diagnostics with timestamps")
	scanCmd.Flags().String("fail-on", "", "Fail with exit code 1 if findings match severities (e.g., critical,high)")
	scanCmd.MarkFlagRequired("rules")
	scanCmd.MarkFlagRequired("project")
}

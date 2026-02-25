package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/shivasurya/code-pathfinder/sast-engine/diff"
	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/executor"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/docker"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/shivasurya/code-pathfinder/sast-engine/ruleset"
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

  # Scan with a remote ruleset bundle
  pathfinder scan --ruleset docker/security --project /path/to/project

  # Scan with an individual rule by ID
  pathfinder scan --ruleset docker/DOCKER-BP-007 --project /path/to/project

  # Scan with multiple individual rules
  pathfinder scan --ruleset docker/DOCKER-BP-007 --ruleset docker/DOCKER-SEC-001 --project .

  # Mix bundles, individual rules, and local rules
  pathfinder scan --rules rules/ --ruleset docker/security --ruleset python/PYTHON-SEC-042 --project .

  # Output to JSON file
  pathfinder scan --ruleset docker/security --project . --output json --output-file results.json

  # SARIF output for CI/CD integration
  pathfinder scan --ruleset docker/security --project . --output sarif --output-file results.sarif`,
	// Note: The main RunE logic is covered by integration tests in exit_code_integration_test.go
	// Unit testing cobra commands requires complex mocking of file systems, graph building, etc.
	// Integration tests provide better coverage for the full execution path.
	RunE: func(cmd *cobra.Command, args []string) error {
		startTime := time.Now()
		rulesPath, _ := cmd.Flags().GetString("rules")
		rulesetSpecs, _ := cmd.Flags().GetStringArray("ruleset")
		refreshRules, _ := cmd.Flags().GetBool("refresh-rules")
		projectPath, _ := cmd.Flags().GetString("project")
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")
		failOnStr, _ := cmd.Flags().GetString("fail-on")
		outputFormat, _ := cmd.Flags().GetString("output")
		outputFile, _ := cmd.Flags().GetString("output-file")
		skipTests, _ := cmd.Flags().GetBool("skip-tests")
		diffAware, _ := cmd.Flags().GetBool("diff-aware")
		baseRef, _ := cmd.Flags().GetString("base")
		headRef, _ := cmd.Flags().GetString("head")

		// Track scan started event (no PII, just metadata)
		analytics.ReportEventWithProperties(analytics.ScanStarted, map[string]any{
			"output_format":     outputFormat,
			"has_local_rules":   rulesPath != "",
			"has_remote_rules":  len(rulesetSpecs) > 0,
			"remote_rule_count": len(rulesetSpecs),
			"skip_tests":        skipTests,
		})

		// Validate that at least one rule source is provided
		if len(rulesetSpecs) == 0 && rulesPath == "" {
			analytics.ReportEventWithProperties(analytics.ScanFailed, map[string]any{
				"error_type": "validation",
				"phase":      "initialization",
			})
			return fmt.Errorf("either --rules or --ruleset flag is required")
		}

		if projectPath == "" {
			analytics.ReportEventWithProperties(analytics.ScanFailed, map[string]any{
				"error_type": "validation",
				"phase":      "initialization",
			})
			return fmt.Errorf("--project flag is required")
		}

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

		// Handle remote ruleset downloads and merge with local rules
		finalRulesPath, tempDir, err := prepareRules(rulesPath, rulesetSpecs, refreshRules, logger)
		if err != nil {
			analytics.ReportEventWithProperties(analytics.ScanFailed, map[string]any{
				"error_type": "rule_preparation",
				"phase":      "initialization",
			})
			return fmt.Errorf("failed to prepare rules: %w", err)
		}
		// Clean up temporary directory if created
		if tempDir != "" {
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					logger.Warning("Failed to clean up temporary directory: %v", err)
				}
			}()
		}

		// Use the prepared rules path for scanning
		rulesPath = finalRulesPath

		if outputFormat != "" && outputFormat != "text" && outputFormat != "json" && outputFormat != "sarif" && outputFormat != "csv" {
			return fmt.Errorf("--output must be 'text', 'json', 'sarif', or 'csv'")
		}

		// Convert project path to absolute path to ensure consistency
		absProjectPath, err := filepath.Abs(projectPath)
		if err != nil {
			return fmt.Errorf("failed to resolve project path: %w", err)
		}
		projectPath = absProjectPath

		// Diff-aware scanning (opt-in for scan command).
		var changedFiles []string
		if diffAware {
			if baseRef == "" {
				return fmt.Errorf("--base flag is required when --diff-aware is enabled")
			}
			if err := diff.ValidateGitRef(projectPath, baseRef); err != nil {
				return fmt.Errorf("invalid base ref %q: %w", baseRef, err)
			}
			files, err := diff.ComputeChangedFiles(baseRef, headRef, projectPath)
			if err != nil {
				return fmt.Errorf("failed to compute changed files: %w", err)
			}
			changedFiles = files
			logger.Progress("Changed files: %d", len(changedFiles))
		}

		// Create rule loader (used for both container and code analysis rules)
		loader := dsl.NewRuleLoader(rulesPath)

		// Step 1: Build code graph (AST)
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
			analytics.ReportEventWithProperties(analytics.ScanFailed, map[string]any{
				"error_type": "empty_project",
				"phase":      "graph_building",
			})
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
			containerRulesJSON, err := loader.LoadContainerRules(logger)
			if err == nil {
				logger.Progress("Executing container rules...")
				containerDetections = executeContainerRules(containerRulesJSON, dockerFiles, composeFiles, projectPath, logger)
				if len(containerDetections) > 0 {
					logger.Statistic("Container scan found %d issue(s)", len(containerDetections))
				} else {
					logger.Progress("No container issues detected")
				}
			} else {
				// Container rule loading failed - log for debugging
				logger.Debug("Container rule loading failed: %v", err)
			}
		}

		// Step 2: Build module registry
		logger.StartProgress("Building module registry", -1)
		moduleRegistry, err := registry.BuildModuleRegistry(projectPath, skipTests)
		logger.FinishProgress()
		if err != nil {
			logger.Warning("failed to build module registry: %v", err)
			// Create empty registry as fallback
			moduleRegistry = core.NewModuleRegistry()
		}
		if skipTests {
			logger.Debug("Skipping test files (use --skip-tests=false to include)")
		}

		// Step 3: Build callgraph
		logger.StartProgress("Building callgraph", -1)
		cg, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectPath, logger)
		logger.FinishProgress()
		if err != nil {
			analytics.ReportEventWithProperties(analytics.ScanFailed, map[string]any{
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
				// Initialize Go stdlib loader and type inference engine
				builder.InitGoStdlibLoader(goRegistry, projectPath, logger)
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

		// Step 4: Load Python DSL rules
		logger.StartProgress("Loading rules", -1)
		rules, err := loader.LoadRules(logger)
		logger.FinishProgress()
		if err != nil {
			analytics.ReportEventWithProperties(analytics.ScanFailed, map[string]any{
				"error_type": "rule_loading",
				"phase":      "rule_loading",
			})
			return fmt.Errorf("failed to load rules: %w", err)
		}
		logger.Statistic("Loaded %d rules", len(rules))

		// Validate that at least one type of rule was loaded
		if len(rules) == 0 && len(containerDetections) == 0 {
			analytics.ReportEventWithProperties(analytics.ScanFailed, map[string]any{
				"error_type": "no_rules",
				"phase":      "rule_loading",
			})
			return fmt.Errorf("no rules loaded: file contains neither code analysis rules (@rule) nor container rules (@dockerfile_rule/@compose_rule)")
		}

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
		logger.StartProgress("Executing rules", len(rules))
		for _, rule := range rules {
			detections, err := loader.ExecuteRule(&rule, cg)
			if err != nil {
				logger.Warning("Error executing rule %s: %v", rule.Rule.ID, err)
				scanErrors = true
				logger.UpdateProgress(1)
				continue
			}

			if len(detections) > 0 {
				enriched, _ := enricher.EnrichAll(detections, rule)
				allEnriched = append(allEnriched, enriched...)
			}
			logger.UpdateProgress(1)
		}
		logger.FinishProgress()

		// Merge container detections with code analysis detections
		allEnriched = append(allEnriched, containerDetections...)

		// Apply diff filter when diff-aware mode is active.
		if diffAware && len(changedFiles) > 0 {
			totalBefore := len(allEnriched)
			diffFilter := output.NewDiffFilter(changedFiles)
			allEnriched = diffFilter.Filter(allEnriched)
			logger.Progress("Diff filter: %d/%d findings in changed files", len(allEnriched), totalBefore)
		}

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
				Version:       Version,
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
				Version:       Version,
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

		// Track scan completion with results (no PII, just counts and metadata)
		severityBreakdown := make(map[string]int)
		for _, det := range allEnriched {
			severityBreakdown[det.Rule.Severity]++
		}

		analytics.ReportEventWithProperties(analytics.ScanCompleted, map[string]any{
			"duration_ms":        time.Since(startTime).Milliseconds(),
			"rules_count":        len(uniqueRules),
			"findings_count":     len(allEnriched),
			"diff_aware":         diffAware,
			"diff_changed_files": len(changedFiles),
			"severity_critical":  severityBreakdown["critical"],
			"severity_high":      severityBreakdown["high"],
			"severity_medium":    severityBreakdown["medium"],
			"severity_low":       severityBreakdown["low"],
			"output_format":      outputFormat,
			"exit_code":          int(exitCode),
			"had_errors":         scanErrors,
		})

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

// countContainerRules parses the container rules JSON IR and returns the total rule count.
func countContainerRules(jsonIR []byte) int {
	var ir struct {
		Dockerfile []json.RawMessage `json:"dockerfile"`
		Compose    []json.RawMessage `json:"compose"`
	}
	if err := json.Unmarshal(jsonIR, &ir); err != nil {
		return 0
	}
	return len(ir.Dockerfile) + len(ir.Compose)
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
	startLine := max(lineNumber-contextLines, 1)
	endLine := min(lineNumber+contextLines, len(lines))

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

// findRulesDirectory locates the rules directory for resolving rule IDs.
// Looks in current directory, parent directories, and common locations.
func findRulesDirectory() string {
	// Check common locations
	candidates := []string{
		"rules",       // Current directory
		"../rules",    // Parent directory
		"../../rules", // Grandparent
		filepath.Join(os.Getenv("HOME"), ".local", "share", "code-pathfinder", "rules"),
		"/usr/local/share/code-pathfinder/rules",
		"/opt/code-pathfinder/rules",
	}

	for _, dir := range candidates {
		if absDir, err := filepath.Abs(dir); err == nil {
			if stat, err := os.Stat(absDir); err == nil && stat.IsDir() {
				return absDir
			}
		}
	}

	// Fallback to current directory + rules
	pwd, _ := os.Getwd()
	return filepath.Join(pwd, "rules")
}

// prepareRules downloads remote rulesets, resolves rule IDs, and merges with local rules if needed.
// Returns: (finalRulesPath, tempDirToCleanup, error).
func prepareRules(localRulesPath string, rulesetSpecs []string, refresh bool, logger *output.Logger) (string, string, error) {
	// Case 1: Only local rules - use directly
	if len(rulesetSpecs) == 0 {
		return localRulesPath, "", nil
	}

	// Separate ruleset specs into bundles and individual rule IDs
	var bundleSpecs []string
	var ruleIDSpecs []string

	for _, spec := range rulesetSpecs {
		parts := strings.Split(spec, "/")
		if len(parts) == 2 && ruleset.IsRuleID(parts[1]) {
			// This is a rule ID (e.g., docker/DOCKER-BP-007)
			ruleIDSpecs = append(ruleIDSpecs, spec)
		} else {
			// This is a bundle (e.g., docker/security) or category expansion (e.g., docker/all)
			bundleSpecs = append(bundleSpecs, spec)
		}
	}

	// Expand "category/all" specs to individual bundle specs
	if len(bundleSpecs) > 0 {
		manifestLoader := ruleset.NewManifestLoader("https://assets.codepathfinder.dev/rules", getCacheDir())
		expanded, err := expandBundleSpecs(bundleSpecs, manifestLoader, logger)
		if err != nil {
			return "", "", err
		}
		bundleSpecs = expanded
	}

	// Download remote bundles
	var downloadedPaths []string
	if len(bundleSpecs) > 0 {
		config := &ruleset.DownloadConfig{
			BaseURL:       "https://assets.codepathfinder.dev/rules",
			CacheDir:      getCacheDir(),
			CacheTTL:      24 * time.Hour,
			ManifestTTL:   1 * time.Hour,
			HTTPTimeout:   30 * time.Second,
			RetryAttempts: 3,
		}

		downloader, err := ruleset.NewDownloader(config)
		if err != nil {
			return "", "", fmt.Errorf("failed to create downloader: %w", err)
		}

		downloadedPaths = make([]string, 0, len(bundleSpecs))
		for _, spec := range bundleSpecs {
			if refresh {
				logger.Progress("Refreshing ruleset cache for %s...", spec)
				if err := downloader.RefreshCache(spec); err != nil {
					logger.Warning("Failed to invalidate cache for %s: %v", spec, err)
				}
			}

			path, err := downloader.Download(spec)
			if err != nil {
				return "", "", fmt.Errorf("failed to download ruleset %s: %w", spec, err)
			}
			downloadedPaths = append(downloadedPaths, path)
			logger.Progress("Downloaded ruleset: %s", spec)
		}
	}

	// Resolve individual rule IDs to file paths
	var resolvedRulePaths []string
	if len(ruleIDSpecs) > 0 {
		rulesBaseDir := findRulesDirectory()
		finder := ruleset.NewRuleFinder(rulesBaseDir)

		for _, spec := range ruleIDSpecs {
			ruleSpec, err := ruleset.ParseRuleSpec(spec)
			if err != nil {
				return "", "", fmt.Errorf("invalid rule spec %s: %w", spec, err)
			}

			if err := ruleSpec.Validate(); err != nil {
				return "", "", fmt.Errorf("invalid rule spec %s: %w", spec, err)
			}

			filePath, err := finder.FindRuleFile(ruleSpec)
			if err != nil {
				return "", "", fmt.Errorf("failed to find rule %s: %w", spec, err)
			}

			resolvedRulePaths = append(resolvedRulePaths, filePath)
			logger.Progress("Resolved rule %s → %s", spec, filepath.Base(filePath))
		}
	}

	// Calculate total sources
	totalSources := len(downloadedPaths) + len(resolvedRulePaths) + boolToInt(localRulesPath != "")

	// Case 2: Single source - use directly
	if totalSources == 1 {
		if localRulesPath != "" {
			return localRulesPath, "", nil
		}
		if len(downloadedPaths) == 1 {
			return downloadedPaths[0], "", nil
		}
		// Single resolved rule file - create temp dir with just that file
		tempDir, err := os.MkdirTemp("", "pathfinder-rules-*")
		if err != nil {
			return "", "", fmt.Errorf("failed to create temp directory: %w", err)
		}
		if err := copyFile(resolvedRulePaths[0], filepath.Join(tempDir, filepath.Base(resolvedRulePaths[0]))); err != nil {
			os.RemoveAll(tempDir)
			return "", "", fmt.Errorf("failed to copy rule file: %w", err)
		}
		return tempDir, tempDir, nil
	}

	// Case 3: Multiple sources - need to merge
	tempDir, err := os.MkdirTemp("", "pathfinder-rules-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	logger.Progress("Merging %d rule source(s)...", totalSources)

	// Copy local rules if provided
	if localRulesPath != "" {
		if err := copyRules(localRulesPath, tempDir, "local"); err != nil {
			os.RemoveAll(tempDir)
			return "", "", fmt.Errorf("failed to copy local rules: %w", err)
		}
	}

	// Copy downloaded bundles
	for i, path := range downloadedPaths {
		destName := fmt.Sprintf("remote-%d", i)
		if err := copyRules(path, tempDir, destName); err != nil {
			os.RemoveAll(tempDir)
			return "", "", fmt.Errorf("failed to copy remote ruleset: %w", err)
		}
	}

	// Copy individual resolved rule files
	for i, filePath := range resolvedRulePaths {
		destName := fmt.Sprintf("rule-%d", i)
		destPath := filepath.Join(tempDir, destName)
		if err := os.MkdirAll(destPath, 0755); err != nil {
			os.RemoveAll(tempDir)
			return "", "", fmt.Errorf("failed to create directory: %w", err)
		}
		destFile := filepath.Join(destPath, filepath.Base(filePath))
		if err := copyFile(filePath, destFile); err != nil {
			os.RemoveAll(tempDir)
			return "", "", fmt.Errorf("failed to copy rule file %s: %w", filePath, err)
		}
	}

	logger.Progress("Merged %d rule source(s)", totalSources)
	return tempDir, tempDir, nil
}

// copyRules copies Python rule files from src to dest/subdir.
func copyRules(src, dest, subdir string) error {
	destDir := filepath.Join(dest, subdir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Check if src is a file or directory
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	if srcInfo.IsDir() {
		// Copy all .py files from directory
		entries, err := os.ReadDir(src)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".py") {
				continue
			}

			srcFile := filepath.Join(src, entry.Name())
			destFile := filepath.Join(destDir, entry.Name())
			if err := copyFile(srcFile, destFile); err != nil {
				return fmt.Errorf("failed to copy %s: %w", entry.Name(), err)
			}
		}
	} else {
		// Single file - copy directly
		destFile := filepath.Join(destDir, filepath.Base(src))
		if err := copyFile(src, destFile); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
	}

	return nil
}

// expandBundleSpecs expands "category/all" specs into individual bundle specs.
// This function is extracted for testability with mock manifest providers.
func expandBundleSpecs(bundleSpecs []string, manifestProvider ruleset.ManifestProvider, logger *output.Logger) ([]string, error) {
	expandedBundleSpecs := make([]string, 0, len(bundleSpecs))

	for _, spec := range bundleSpecs {
		parsed, err := ruleset.ParseSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid ruleset spec %s: %w", spec, err)
		}

		// Check if this is a category expansion (bundle == "*")
		if parsed.Bundle == "*" {
			// Load category manifest to get all bundle names
			manifest, err := manifestProvider.LoadCategoryManifest(parsed.Category)
			if err != nil {
				return nil, fmt.Errorf("failed to load manifest for category %s: %w", parsed.Category, err)
			}

			// Expand to all bundles in category
			bundleNames := manifest.GetAllBundleNames()
			if len(bundleNames) == 0 {
				logger.Warning("Category %s has no bundles", parsed.Category)
				continue
			}

			logger.Progress("Expanding %s/all to %d bundles: %v", parsed.Category, len(bundleNames), bundleNames)

			for _, bundleName := range bundleNames {
				expandedBundleSpecs = append(expandedBundleSpecs, fmt.Sprintf("%s/%s", parsed.Category, bundleName))
			}
		} else {
			// Regular bundle spec, keep as-is
			expandedBundleSpecs = append(expandedBundleSpecs, spec)
		}
	}

	return expandedBundleSpecs, nil
}

// copyFile copies a single file from src to dest.
func copyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Close()
}

// boolToInt converts bool to int (0 or 1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// getCacheDir returns platform-specific cache directory.
func getCacheDir() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	return filepath.Join(cacheDir, "code-pathfinder", "rules")
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringP("rules", "r", "", "Path to Python DSL rules file or directory")
	scanCmd.Flags().StringArray("ruleset", []string{}, "Ruleset bundle (e.g., docker/security) or individual rule ID (e.g., docker/DOCKER-BP-007). Can be specified multiple times.")
	scanCmd.Flags().Bool("refresh-rules", false, "Force refresh of cached rulesets")
	scanCmd.Flags().StringP("project", "p", "", "Path to project directory to scan (required)")
	scanCmd.Flags().StringP("output", "o", "text", "Output format: text, json, sarif, or csv (default: text)")
	scanCmd.Flags().StringP("output-file", "f", "", "Write output to file instead of stdout")
	scanCmd.Flags().BoolP("verbose", "v", false, "Show statistics and timing information")
	scanCmd.Flags().Bool("debug", false, "Show detailed debug diagnostics with file-level progress and timestamps")
	scanCmd.Flags().String("fail-on", "", "Fail with exit code 1 if findings match severities (e.g., critical,high)")
	scanCmd.Flags().Bool("skip-tests", true, "Skip test files (test_*.py, *_test.py, conftest.py, etc.)")
	scanCmd.Flags().Bool("diff-aware", false, "Enable diff-aware scanning (only report findings in changed files)")
	scanCmd.Flags().String("base", "", "Base git ref for diff-aware scanning (required with --diff-aware)")
	scanCmd.Flags().String("head", "HEAD", "Head git ref for diff-aware scanning")
	scanCmd.MarkFlagRequired("project")
}

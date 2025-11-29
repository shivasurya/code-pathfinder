package cmd

import (
	"fmt"
	"sort"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/spf13/cobra"
)

// Note: callgraph.InitializeCallGraph is used from the callgraph root package integration

var resolutionReportCmd = &cobra.Command{
	Use:   "resolution-report",
	Short: "Generate a diagnostic report on call resolution statistics",
	Long: `Analyze the call graph and generate a detailed report showing:
  - Overall resolution statistics (resolved vs unresolved)
  - Breakdown by failure category
  - Top unresolved patterns with occurrence counts

This helps identify why calls are not being resolved and prioritize
improvements to the resolution logic.`,
	Run: func(cmd *cobra.Command, _ []string) {
		projectInput := cmd.Flag("project").Value.String()

		if projectInput == "" {
			fmt.Println("Error: --project flag is required")
			return
		}

		fmt.Println("Building code graph...")
		codeGraph := graph.Initialize(projectInput)

		fmt.Println("Building call graph...")
		cg, registry, _, err := callgraph.InitializeCallGraph(codeGraph, projectInput, output.NewLogger(output.VerbosityDefault))
		if err != nil {
			fmt.Printf("Error building call graph: %v\n", err)
			return
		}

		fmt.Printf("\nResolution Report for %s\n", projectInput)
		fmt.Println("===============================================")

		// Collect statistics
		stats := aggregateResolutionStatistics(cg)

		// Print overall statistics
		printOverallStatistics(stats)
		fmt.Println()

		// Phase 2: Print type inference statistics
		if stats.TypeInferenceResolved > 0 {
			printTypeInferenceStatistics(stats)
			fmt.Println()
		}

		// Print stdlib registry statistics
		if stats.StdlibResolved > 0 {
			printStdlibStatistics(stats)
			fmt.Println()
		}

		// Print failure breakdown
		printFailureBreakdown(stats)
		fmt.Println()

		// Print top unresolved patterns
		printTopUnresolvedPatterns(stats, 20)
		fmt.Println()

		fmt.Printf("Module registry: %d modules\n", len(registry.Modules))
	},
}

// resolutionStatistics holds aggregated statistics about call resolution.
type resolutionStatistics struct {
	TotalCalls       int
	ResolvedCalls    int
	UnresolvedCalls  int
	FailuresByReason map[string]int                // Category -> count
	PatternCounts    map[string]int                // Target pattern -> count
	FrameworkCounts  map[string]int                // Framework prefix -> count (for external_framework category)
	UnresolvedByFQN  map[string]core.CallSite // For detailed inspection

	// Phase 2: Type inference statistics
	TypeInferenceResolved  int            // Calls resolved via type inference
	ResolvedByTraditional  int            // Calls resolved via traditional methods
	TypesBySource          map[string]int // TypeInfo.Source -> count
	BuiltinTypeResolved    int            // Resolved to builtin types
	ClassTypeResolved      int            // Resolved to project classes
	ConfidenceSum          float64        // Sum of confidence scores for averaging
	ConfidenceDistribution map[string]int // Confidence ranges -> count

	// Stdlib registry statistics
	StdlibResolved      int            // Calls resolved to stdlib
	StdlibByModule      map[string]int // Module name -> count (e.g., "os" -> 45)
	StdlibByType        map[string]int // Type -> count (function, class, constant, attribute)
	StdlibViaAnnotation int            // Resolved via type annotations
	StdlibViaInference  int            // Resolved via type inference
	StdlibViaBuiltin    int            // Resolved via builtin registry
}

// aggregateResolutionStatistics analyzes the call graph and collects statistics.
func aggregateResolutionStatistics(cg *core.CallGraph) *resolutionStatistics {
	stats := &resolutionStatistics{
		FailuresByReason:       make(map[string]int),
		PatternCounts:          make(map[string]int),
		FrameworkCounts:        make(map[string]int),
		UnresolvedByFQN:        make(map[string]core.CallSite),
		TypesBySource:          make(map[string]int),
		ConfidenceDistribution: make(map[string]int),
		StdlibByModule:         make(map[string]int),
		StdlibByType:           make(map[string]int),
	}

	// Iterate through all call sites
	for _, callSites := range cg.CallSites {
		for _, site := range callSites {
			stats.TotalCalls++

			if site.Resolved {
				stats.ResolvedCalls++

				// Track stdlib resolutions
				if isStdlibResolution(site.TargetFQN) {
					stats.StdlibResolved++

					// Extract module name (first component before dot)
					moduleName := extractModuleName(site.TargetFQN)
					if moduleName != "" {
						stats.StdlibByModule[moduleName]++
					}

					// Determine resolution source
					switch {
					case site.TypeSource == "annotation" || site.TypeSource == "stdlib_annotation":
						stats.StdlibViaAnnotation++
					case site.ResolvedViaTypeInference:
						stats.StdlibViaInference++
					case site.TypeSource == "builtin" || site.TypeSource == "stdlib_builtin":
						stats.StdlibViaBuiltin++
					}

					// Track type (function, class, etc.)
					resType := determineStdlibType(site.TargetFQN)
					stats.StdlibByType[resType]++
				}

				// Phase 2: Track type inference resolutions
				if site.ResolvedViaTypeInference {
					stats.TypeInferenceResolved++
					stats.ConfidenceSum += float64(site.TypeConfidence)

					// Track by source
					if site.TypeSource != "" {
						stats.TypesBySource[site.TypeSource]++
					}

					// Track builtin vs class
					if containsString(site.InferredType, "builtins.") {
						stats.BuiltinTypeResolved++
					} else {
						stats.ClassTypeResolved++
					}

					// Track confidence distribution
					conf := site.TypeConfidence
					switch {
					case conf >= 0.9:
						stats.ConfidenceDistribution["0.9-1.0 (high)"]++
					case conf >= 0.7:
						stats.ConfidenceDistribution["0.7-0.9 (medium-high)"]++
					case conf >= 0.5:
						stats.ConfidenceDistribution["0.5-0.7 (medium)"]++
					default:
						stats.ConfidenceDistribution["0.0-0.5 (low)"]++
					}
				} else {
					stats.ResolvedByTraditional++
				}
			} else {
				stats.UnresolvedCalls++

				// Count by failure reason
				if site.FailureReason != "" {
					stats.FailuresByReason[site.FailureReason]++
				} else {
					stats.FailuresByReason["uncategorized"]++
				}

				// Count pattern occurrences
				stats.PatternCounts[site.Target]++

				// For external frameworks, track which framework
				if site.FailureReason == "external_framework" {
					// Extract framework prefix (first component before dot)
					for idx := 0; idx < len(site.TargetFQN); idx++ {
						if site.TargetFQN[idx] == '.' {
							framework := site.TargetFQN[:idx]
							stats.FrameworkCounts[framework]++
							break
						}
					}
				}

				// Store for detailed inspection
				stats.UnresolvedByFQN[site.TargetFQN] = site
			}
		}
	}

	return stats
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// printOverallStatistics prints the overall resolution statistics.
func printOverallStatistics(stats *resolutionStatistics) {
	fmt.Println("Overall Statistics:")
	fmt.Printf("  Total calls:       %d\n", stats.TotalCalls)
	fmt.Printf("  Resolved:          %d (%.1f%%)\n",
		stats.ResolvedCalls,
		percentage(stats.ResolvedCalls, stats.TotalCalls))
	fmt.Printf("  Unresolved:        %d (%.1f%%)\n",
		stats.UnresolvedCalls,
		percentage(stats.UnresolvedCalls, stats.TotalCalls))
}

// printTypeInferenceStatistics prints Phase 2 type inference statistics.
func printTypeInferenceStatistics(stats *resolutionStatistics) {
	fmt.Println("Type Inference Statistics:")
	fmt.Printf("  Resolved via type inference:  %d (%.1f%% of resolved)\n",
		stats.TypeInferenceResolved,
		percentage(stats.TypeInferenceResolved, stats.ResolvedCalls))
	fmt.Printf("  Resolved via traditional:     %d (%.1f%% of resolved)\n",
		stats.ResolvedByTraditional,
		percentage(stats.ResolvedByTraditional, stats.ResolvedCalls))
	fmt.Println()

	// Type breakdown
	fmt.Printf("  Type breakdown:\n")
	fmt.Printf("    Builtin types:  %d (%.1f%%)\n",
		stats.BuiltinTypeResolved,
		percentage(stats.BuiltinTypeResolved, stats.TypeInferenceResolved))
	fmt.Printf("    Class types:    %d (%.1f%%)\n",
		stats.ClassTypeResolved,
		percentage(stats.ClassTypeResolved, stats.TypeInferenceResolved))
	fmt.Println()

	// Average confidence
	avgConfidence := 0.0
	if stats.TypeInferenceResolved > 0 {
		avgConfidence = stats.ConfidenceSum / float64(stats.TypeInferenceResolved)
	}
	fmt.Printf("  Average confidence: %.2f\n", avgConfidence)
	fmt.Println()

	// Confidence distribution
	if len(stats.ConfidenceDistribution) > 0 {
		fmt.Printf("  Confidence distribution:\n")
		// Sort by key for consistent output
		keys := []string{"0.9-1.0 (high)", "0.7-0.9 (medium-high)", "0.5-0.7 (medium)", "0.0-0.5 (low)"}
		for _, key := range keys {
			if count, ok := stats.ConfidenceDistribution[key]; ok {
				fmt.Printf("    %-20s %d (%.1f%%)\n",
					key+":",
					count,
					percentage(count, stats.TypeInferenceResolved))
			}
		}
		fmt.Println()
	}

	// By inference source
	if len(stats.TypesBySource) > 0 {
		fmt.Printf("  By inference source:\n")
		// Sort sources by count (descending)
		type sourceCount struct {
			source string
			count  int
		}
		sources := make([]sourceCount, 0, len(stats.TypesBySource))
		for source, count := range stats.TypesBySource {
			sources = append(sources, sourceCount{source, count})
		}
		sort.Slice(sources, func(i, j int) bool {
			return sources[i].count > sources[j].count
		})

		for _, sc := range sources {
			fmt.Printf("    %-30s %d (%.1f%%)\n",
				sc.source+":",
				sc.count,
				percentage(sc.count, stats.TypeInferenceResolved))
		}
	}
}

// printFailureBreakdown prints the breakdown of failures by category.
func printFailureBreakdown(stats *resolutionStatistics) {
	fmt.Println("Failure Breakdown:")

	// Sort categories by count (descending)
	type categoryCount struct {
		category string
		count    int
	}
	categories := make([]categoryCount, 0, len(stats.FailuresByReason))
	for cat, count := range stats.FailuresByReason {
		categories = append(categories, categoryCount{cat, count})
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].count > categories[j].count
	})

	// Print each category
	for _, cc := range categories {
		fmt.Printf("  %-20s %d (%.1f%%)\n",
			cc.category+":",
			cc.count,
			percentage(cc.count, stats.TotalCalls))

		// For external frameworks, show framework breakdown
		if cc.category == "external_framework" && len(stats.FrameworkCounts) > 0 {
			// Sort frameworks by count
			type frameworkCount struct {
				framework string
				count     int
			}
			var frameworks []frameworkCount
			for fw, count := range stats.FrameworkCounts {
				frameworks = append(frameworks, frameworkCount{fw, count})
			}
			sort.Slice(frameworks, func(i, j int) bool {
				return frameworks[i].count > frameworks[j].count
			})

			// Print top 5 frameworks
			for i, fc := range frameworks {
				if i >= 5 {
					break
				}
				fmt.Printf("    %s.*: %d\n", fc.framework, fc.count)
			}
		}
	}
}

// printTopUnresolvedPatterns prints the most common unresolved patterns.
func printTopUnresolvedPatterns(stats *resolutionStatistics, topN int) {
	fmt.Printf("Top %d Unresolved Patterns:\n", topN)

	// Sort patterns by count (descending)
	type patternCount struct {
		pattern string
		count   int
	}
	patterns := make([]patternCount, 0, len(stats.PatternCounts))
	for pattern, count := range stats.PatternCounts {
		patterns = append(patterns, patternCount{pattern, count})
	}
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].count > patterns[j].count
	})

	// Print top N patterns
	for i, pc := range patterns {
		if i >= topN {
			break
		}
		fmt.Printf("  %2d. %-40s %d occurrences\n", i+1, pc.pattern, pc.count)
	}
}

// percentage calculates the percentage of part out of total.
func percentage(part, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(part) * 100.0 / float64(total)
}

// isStdlibResolution checks if a FQN resolves to Python stdlib.
func isStdlibResolution(fqn string) bool {
	// List of common stdlib modules
	stdlibModules := []string{
		"os.", "sys.", "pathlib.", "re.", "json.", "time.", "datetime.",
		"collections.", "itertools.", "functools.", "math.", "random.",
		"subprocess.", "threading.", "multiprocessing.", "asyncio.",
		"logging.", "argparse.", "unittest.", "sqlite3.", "csv.",
		"xml.", "html.", "urllib.", "http.", "email.", "socket.",
		"io.", "tempfile.", "shutil.", "glob.", "pickle.", "base64.",
		"hashlib.", "hmac.", "secrets.", "struct.", "codecs.", "typing.",
		"abc.", "contextlib.", "warnings.", "traceback.", "inspect.",
		"ast.", "dis.", "zipfile.", "tarfile.", "gzip.", "bz2.",
	}

	for _, mod := range stdlibModules {
		if len(fqn) >= len(mod) && fqn[:len(mod)] == mod {
			return true
		}
	}

	return false
}

// extractModuleName extracts the top-level module name from a FQN.
// Example: "os.path.join" -> "os".
func extractModuleName(fqn string) string {
	for i := 0; i < len(fqn); i++ {
		if fqn[i] == '.' {
			return fqn[:i]
		}
	}
	return fqn
}

// determineStdlibType determines if the target is a function, class, method, etc.
func determineStdlibType(fqn string) string {
	// Split FQN by dots
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(fqn); i++ {
		if fqn[i] == '.' {
			parts = append(parts, fqn[start:i])
			start = i + 1
		}
	}
	if start < len(fqn) {
		parts = append(parts, fqn[start:])
	}

	if len(parts) == 0 {
		return "unknown"
	}

	// Last component
	lastPart := parts[len(parts)-1]

	// Class names typically start with uppercase
	if len(lastPart) > 0 && lastPart[0] >= 'A' && lastPart[0] <= 'Z' {
		return "class"
	}

	// If there are multiple parts and second-to-last is uppercase, likely a method
	if len(parts) >= 2 {
		secondLast := parts[len(parts)-2]
		if len(secondLast) > 0 && secondLast[0] >= 'A' && secondLast[0] <= 'Z' {
			return "method"
		}
	}

	// Constants are all uppercase
	isConstant := true
	for i := 0; i < len(lastPart); i++ {
		if lastPart[i] >= 'a' && lastPart[i] <= 'z' {
			isConstant = false
			break
		}
	}
	if isConstant && len(lastPart) > 1 {
		return "constant"
	}

	// Default to function
	return "function"
}

// printStdlibStatistics prints Python stdlib registry statistics.
func printStdlibStatistics(stats *resolutionStatistics) {
	fmt.Println("Stdlib Registry Statistics:")
	fmt.Printf("  Total stdlib resolutions:  %d (%.1f%% of resolved)\n",
		stats.StdlibResolved,
		percentage(stats.StdlibResolved, stats.ResolvedCalls))
	fmt.Println()

	// Resolution source breakdown
	if stats.StdlibViaAnnotation > 0 || stats.StdlibViaInference > 0 || stats.StdlibViaBuiltin > 0 {
		fmt.Printf("  Resolution source:\n")
		if stats.StdlibViaAnnotation > 0 {
			fmt.Printf("    Type annotations:  %d (%.1f%%)\n",
				stats.StdlibViaAnnotation,
				percentage(stats.StdlibViaAnnotation, stats.StdlibResolved))
		}
		if stats.StdlibViaInference > 0 {
			fmt.Printf("    Type inference:    %d (%.1f%%)\n",
				stats.StdlibViaInference,
				percentage(stats.StdlibViaInference, stats.StdlibResolved))
		}
		if stats.StdlibViaBuiltin > 0 {
			fmt.Printf("    Builtin registry:  %d (%.1f%%)\n",
				stats.StdlibViaBuiltin,
				percentage(stats.StdlibViaBuiltin, stats.StdlibResolved))
		}
		fmt.Println()
	}

	// By type (function, class, etc.)
	if len(stats.StdlibByType) > 0 {
		fmt.Printf("  By type:\n")
		// Sort by count
		type typeCount struct {
			typeName string
			count    int
		}
		types := make([]typeCount, 0, len(stats.StdlibByType))
		for t, count := range stats.StdlibByType {
			types = append(types, typeCount{t, count})
		}
		sort.Slice(types, func(i, j int) bool {
			return types[i].count > types[j].count
		})

		for _, tc := range types {
			fmt.Printf("    %-15s %d (%.1f%%)\n",
				tc.typeName+":",
				tc.count,
				percentage(tc.count, stats.StdlibResolved))
		}
		fmt.Println()
	}

	// Top modules
	if len(stats.StdlibByModule) > 0 {
		fmt.Printf("  Top 10 modules:\n")
		// Sort by count
		type moduleCount struct {
			module string
			count  int
		}
		modules := make([]moduleCount, 0, len(stats.StdlibByModule))
		for mod, count := range stats.StdlibByModule {
			modules = append(modules, moduleCount{mod, count})
		}
		sort.Slice(modules, func(i, j int) bool {
			return modules[i].count > modules[j].count
		})

		// Print top 10
		for i, mc := range modules {
			if i >= 10 {
				break
			}
			fmt.Printf("    %2d. %-15s %d calls\n", i+1, mc.module, mc.count)
		}
	}
}

func init() {
	rootCmd.AddCommand(resolutionReportCmd)
	resolutionReportCmd.Flags().StringP("project", "p", "", "Project root directory")
	resolutionReportCmd.MarkFlagRequired("project")
}

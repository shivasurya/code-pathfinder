package cmd

import (
	"fmt"
	"sort"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
	"github.com/spf13/cobra"
)

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
		cg, registry, _, err := callgraph.InitializeCallGraph(codeGraph, projectInput)
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
	UnresolvedByFQN  map[string]callgraph.CallSite // For detailed inspection
}

// aggregateResolutionStatistics analyzes the call graph and collects statistics.
func aggregateResolutionStatistics(cg *callgraph.CallGraph) *resolutionStatistics {
	stats := &resolutionStatistics{
		FailuresByReason: make(map[string]int),
		PatternCounts:    make(map[string]int),
		FrameworkCounts:  make(map[string]int),
		UnresolvedByFQN:  make(map[string]callgraph.CallSite),
	}

	// Iterate through all call sites
	for _, callSites := range cg.CallSites {
		for _, site := range callSites {
			stats.TotalCalls++

			if site.Resolved {
				stats.ResolvedCalls++
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

func init() {
	rootCmd.AddCommand(resolutionReportCmd)
	resolutionReportCmd.Flags().StringP("project", "p", "", "Project root directory")
	resolutionReportCmd.MarkFlagRequired("project")
}

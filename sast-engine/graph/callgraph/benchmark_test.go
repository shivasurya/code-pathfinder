package callgraph

import (
	"os"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// Benchmark project paths
// These paths are used for performance testing against real-world codebases.
const (
	// Small project: ~5 Python files, simple imports.
	smallProjectPath = "../test-fixtures/python/simple_project"

	// Medium project: label-studio (~1000 Python files, complex imports).
	mediumProjectPath = "/Users/shiva/src/label-studio/label_studio"

	// Large project: salt (~10,000 Python files, very complex imports).
	largeProjectPath = "/Users/shiva/src/shivasurya/salt/salt"
)

// BenchmarkBuildModuleRegistry_Small measures module registry performance on a small codebase.
// Target: <10ms
//
// This benchmark tests Pass 1 of the 3-pass algorithm on a minimal project.
// It measures the overhead of directory walking and module path conversion.
func BenchmarkBuildModuleRegistry_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		moduleRegistry, err := registry.BuildModuleRegistry(smallProjectPath, false)
		if err != nil {
			b.Fatalf("Failed to build module registry: %v", err)
		}
		if len(moduleRegistry.Modules) == 0 {
			b.Fatal("Expected modules to be registered")
		}
	}
}

// BenchmarkBuildModuleRegistry_Medium measures module registry performance on a medium codebase.
// Target: <500ms
//
// This benchmark tests Pass 1 against label-studio, a real-world Django application.
// It stresses the directory walking and file filtering logic.
func BenchmarkBuildModuleRegistry_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		moduleRegistry, err := registry.BuildModuleRegistry(mediumProjectPath, false)
		if err != nil {
			b.Fatalf("Failed to build module registry: %v", err)
		}
		if len(moduleRegistry.Modules) == 0 {
			b.Fatal("Expected modules to be registered")
		}
	}
}

// BenchmarkBuildModuleRegistry_Large measures module registry performance on a large codebase.
// Target: <2s
//
// This benchmark tests Pass 1 against salt, a massive Python project with thousands of modules.
// It validates that the algorithm scales to production-sized codebases.
func BenchmarkBuildModuleRegistry_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		moduleRegistry, err := registry.BuildModuleRegistry(largeProjectPath, false)
		if err != nil {
			b.Fatalf("Failed to build module registry: %v", err)
		}
		if len(moduleRegistry.Modules) == 0 {
			b.Fatal("Expected modules to be registered")
		}
	}
}

// BenchmarkExtractImports_Small measures import extraction performance on a small project.
// Target: <20ms
//
// This benchmark tests Pass 2A (import extraction) using tree-sitter parsing.
// It measures parser initialization and AST traversal overhead.
func BenchmarkExtractImports_Small(b *testing.B) {
	// Pre-build registry to isolate import extraction performance
	moduleRegistry, err := registry.BuildModuleRegistry(smallProjectPath, false)
	if err != nil {
		b.Fatalf("Failed to build module registry: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Extract imports from all files in the small project
		for modulePath, filePath := range moduleRegistry.Modules {
			sourceCode, readErr := os.ReadFile(filePath)
			if readErr != nil {
				b.Fatalf("Failed to read file %s: %v", filePath, readErr)
			}

			_, extractErr := resolution.ExtractImports(filePath, sourceCode, moduleRegistry)
			if extractErr != nil {
				b.Fatalf("Failed to extract imports from %s: %v", modulePath, extractErr)
			}
		}
	}
}

// BenchmarkExtractImports_Medium measures import extraction performance on a medium project.
// Target: <2s
//
// This benchmark tests Pass 2A against label-studio's import patterns.
// It validates that tree-sitter parsing scales to production projects.
func BenchmarkExtractImports_Medium(b *testing.B) {
	// Pre-build registry to isolate import extraction performance
	moduleRegistry, err := registry.BuildModuleRegistry(mediumProjectPath, false)
	if err != nil {
		b.Fatalf("Failed to build module registry: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Extract imports from all files in the medium project
		for _, filePath := range moduleRegistry.Modules {
			sourceCode, readErr := os.ReadFile(filePath)
			if readErr != nil {
				// Skip files that can't be read (permissions, etc.)
				continue
			}

			_, extractErr := resolution.ExtractImports(filePath, sourceCode, moduleRegistry)
			if extractErr != nil {
				// Skip files with parse errors (syntax errors, etc.)
				continue
			}
		}
	}
}

// BenchmarkExtractCallSites_Small measures call site extraction performance on a small project.
// Target: <30ms
//
// This benchmark tests Pass 2B (call site extraction) using tree-sitter.
// It measures the overhead of finding all function/method calls in the AST.
func BenchmarkExtractCallSites_Small(b *testing.B) {
	// Pre-build registry and import maps
	moduleRegistry, err := registry.BuildModuleRegistry(smallProjectPath, false)
	if err != nil {
		b.Fatalf("Failed to build module registry: %v", err)
	}

	// Build import maps for all files
	importMaps := make(map[string]*core.ImportMap)
	for modulePath, filePath := range moduleRegistry.Modules {
		sourceCode, readErr := os.ReadFile(filePath)
		if readErr != nil {
			b.Fatalf("Failed to read file %s: %v", filePath, readErr)
		}

		importMap, extractErr := resolution.ExtractImports(filePath, sourceCode, moduleRegistry)
		if extractErr != nil {
			b.Fatalf("Failed to extract imports from %s: %v", modulePath, extractErr)
		}
		importMaps[filePath] = importMap
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Extract call sites from all files
		for _, filePath := range moduleRegistry.Modules {
			sourceCode, readErr := os.ReadFile(filePath)
			if readErr != nil {
				b.Fatalf("Failed to read file %s: %v", filePath, readErr)
			}

			importMap := importMaps[filePath]
			_, extractErr := resolution.ExtractCallSites(filePath, sourceCode, importMap)
			if extractErr != nil {
				b.Fatalf("Failed to extract call sites from %s: %v", filePath, extractErr)
			}
		}
	}
}

// BenchmarkBuildCallGraph_Small measures end-to-end call graph construction on a small project.
// Target: <100ms
//
// This benchmark tests the complete 3-pass algorithm:
//  - Pass 1: Module registry
//  - Pass 2A: Import extraction
//  - Pass 2B: Call site extraction
//  - Pass 3: Call graph construction
//
// Note: Currently skipped because BuildCallGraph expects codeGraph to have functions pre-indexed
// which requires full AST parsing. Use BenchmarkInitializeCallGraph_Small instead which
// includes the full pipeline.
func BenchmarkBuildCallGraph_Small(b *testing.B) {
	b.Skip("Skipping: BuildCallGraph requires codeGraph with pre-indexed functions")
}

// BenchmarkBuildCallGraph_Medium measures end-to-end call graph construction on a medium project.
// Target: <5s
//
// This benchmark validates that the 3-pass algorithm scales to label-studio.
//
// Note: Currently skipped. Use BenchmarkInitializeCallGraph_Medium instead.
func BenchmarkBuildCallGraph_Medium(b *testing.B) {
	b.Skip("Skipping: BuildCallGraph requires codeGraph with pre-indexed functions")
}

// BenchmarkBuildCallGraph_Large measures end-to-end call graph construction on a large project.
// Target: <30s
//
// This benchmark validates that the 3-pass algorithm can handle salt's complexity.
//
// Note: Currently skipped. Use BenchmarkInitializeCallGraph_Large instead (when enabled).
func BenchmarkBuildCallGraph_Large(b *testing.B) {
	b.Skip("Skipping: BuildCallGraph requires codeGraph with pre-indexed functions")
}

// BenchmarkInitializeCallGraph_Small measures the full initialization pipeline on a small project.
// Target: <150ms
//
// This benchmark tests InitializeCallGraph(), which includes:
//  - Code graph initialization (AST parsing)
//  - Module registry building
//  - Call graph construction
//  - Pattern registry loading
func BenchmarkInitializeCallGraph_Small(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Include full pipeline: graph initialization + call graph initialization
		codeGraph := graph.Initialize(smallProjectPath, nil)
		callGraph, registry, patternRegistry, err := InitializeCallGraph(codeGraph, smallProjectPath, output.NewLogger(output.VerbosityDefault))
		if err != nil {
			b.Fatalf("Failed to initialize call graph: %v", err)
		}

		// Validate results (don't fail if Functions is empty since it depends on graph.Initialize)
		if len(registry.Modules) == 0 {
			b.Fatal("Expected modules to be registered")
		}
		if len(patternRegistry.Patterns) == 0 {
			b.Fatal("Expected patterns to be loaded")
		}
		_ = callGraph // Use callGraph to avoid unused variable
	}
}

// BenchmarkInitializeCallGraph_Medium measures the full initialization pipeline on a medium project.
// Target: <10s
//
// Note: Disabled by default due to long runtime. Enable manually to test medium project performance.
func BenchmarkInitializeCallGraph_Medium(b *testing.B) {
	b.Skip("Skipping: Medium project benchmarks take >10s, enable manually")

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		codeGraph := graph.Initialize(mediumProjectPath, nil)
		callGraph, registry, patternRegistry, err := InitializeCallGraph(codeGraph, mediumProjectPath, output.NewLogger(output.VerbosityDefault))
		if err != nil {
			b.Fatalf("Failed to initialize call graph: %v", err)
		}

		if len(registry.Modules) == 0 {
			b.Fatal("Expected modules to be registered")
		}
		if len(patternRegistry.Patterns) == 0 {
			b.Fatal("Expected patterns to be loaded")
		}
		_ = callGraph
	}
}

// BenchmarkPatternMatching_Small measures security pattern analysis performance on a small project.
// Target: <50ms
//
// This benchmark tests the pattern matching engine against a small call graph.
func BenchmarkPatternMatching_Small(b *testing.B) {
	// Pre-build call graph and pattern registry
	codeGraph := graph.Initialize(smallProjectPath, nil)
	callGraph, registry, patternRegistry, err := InitializeCallGraph(codeGraph, smallProjectPath, output.NewLogger(output.VerbosityDefault))
	if err != nil {
		b.Fatalf("Failed to initialize call graph: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		matches := AnalyzePatterns(callGraph, patternRegistry)
		_ = matches // Use matches to avoid compiler optimization
	}

	_ = registry // Silence unused variable warning
}

// BenchmarkPatternMatching_Medium measures security pattern analysis performance on a medium project.
// Target: <2s
//
// This benchmark validates that pattern matching scales to label-studio's call graph.
func BenchmarkPatternMatching_Medium(b *testing.B) {
	// Pre-build call graph and pattern registry
	codeGraph := graph.Initialize(mediumProjectPath, nil)
	callGraph, registry, patternRegistry, err := InitializeCallGraph(codeGraph, mediumProjectPath, output.NewLogger(output.VerbosityDefault))
	if err != nil {
		b.Fatalf("Failed to initialize call graph: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		matches := AnalyzePatterns(callGraph, patternRegistry)
		_ = matches
	}

	_ = registry
}

// BenchmarkResolveCallTarget measures call target resolution performance.
// Target: <1µs per call
//
// This benchmark tests the hot path for resolving function calls to FQNs.
// It's critical for overall performance since it's called for every call site.
//
// Note: Skipped because resolveCallTarget is now a private function in the builder package.
func BenchmarkResolveCallTarget(b *testing.B) {
	b.Skip("Skipping: resolveCallTarget is now a private function in builder package")
}

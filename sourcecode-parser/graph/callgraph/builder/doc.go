// Package builder provides call graph construction orchestration.
//
// This package ties together all components to build a complete call graph:
//   - Module registry (registry package)
//   - Type inference (resolution package)
//   - Import resolution (resolution package)
//   - Call site extraction (extraction package)
//   - Advanced resolution (resolution package)
//   - Pattern detection (patterns package)
//   - Taint analysis (analysis/taint package)
//
// # Basic Usage
//
//	// Build from existing code graph
//	callGraph, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectRoot)
//
// # Call Resolution Strategy
//
// The builder uses a multi-strategy approach to resolve function calls:
//  1. Direct import resolution
//  2. Method chaining with type inference
//  3. Self-attribute resolution (self.attr.method)
//  4. Type inference for variable.method() calls
//  5. ORM pattern detection (Django, SQLAlchemy)
//  6. Framework detection (known external frameworks)
//  7. Standard library resolution via remote CDN
//
// Each strategy is tried in order until one succeeds.
//
// # Multi-Pass Architecture
//
// The builder performs multiple passes over the codebase:
//
//  Pass 1: Index all function definitions
//  Pass 2: Extract return types from all functions
//  Pass 3: Extract variable assignments and type bindings
//  Pass 4: Extract class attributes
//  Pass 5: Resolve call sites and build call graph edges
//  Pass 6: Generate taint summaries for security analysis
//
// This multi-pass approach ensures that all necessary type information
// is collected before attempting to resolve call sites.
//
// # Caching
//
// The builder uses ImportMapCache to avoid re-parsing imports from
// the same file multiple times, significantly improving performance.
//
// # Thread Safety
//
// All exported functions in this package are thread-safe. The ImportMapCache
// uses a read-write mutex to allow concurrent reads while ensuring safe writes.
package builder

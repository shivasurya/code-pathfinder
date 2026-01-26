# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Build Commands

### Building the Binary
```bash
cd sast-engine
gradle buildGo
```
The binary is output to `build/go/pathfinder`. The build automatically cleans previous builds first.

### Running Tests
```bash
gradle testGo          # Run all Go tests
go test ./...          # Direct Go test command
go test -v ./graph/... # Run tests for specific package with verbose output
```

### Linting
```bash
gradle lintGo
# or directly:
golangci-lint run
```

### Running the Binary
```bash
# MCP server mode (for AI coding assistants)
./build/go/pathfinder serve --project <path>
./build/go/pathfinder serve --http --address :8080 --project <path>

# Scan mode (using Python DSL rules)
./build/go/pathfinder scan --project <path> --ruleset <path_to_rules>

# CI mode (loads rules from remote/local, outputs SARIF/JSON/CSV)
./build/go/pathfinder ci --project <path> --ruleset cpf/java --output sarif

# Resolution diagnostics
./build/go/pathfinder resolution-report --project <path>

# Taint analysis diagnostics
./build/go/pathfinder diagnose --project <path>
```

### Running a Single Test
```bash
go test -v -run TestBuildCallGraph ./graph/callgraph/builder/
go test -v -run TestTypeInference ./graph/callgraph/resolution/
```

## High-Level Architecture

Code Pathfinder is a multi-stage analysis pipeline:

```
Source Files (.py, .java, Dockerfile, docker-compose.yml)
    ↓
Tree-Sitter AST Parsing (parallel workers)
    ↓
Code Graph (Functions, Classes, Call Edges)
    ↓
Type Inference Engine (bidirectional, return types, variable assignments)
    ↓
Call Graph Builder (5-pass algorithm)
    ↓
MCP Server / Python DSL Rules
    ↓
Output Formats (JSON, SARIF, CSV, Text)
```

### Core Packages

**sast-engine/graph/** - Code graph construction and management
- `initialize.go`: Multi-threaded file parsing with parallel workers
- `parser.go`: AST traversal orchestrator (language-agnostic entry point)
- `parser_java.go`: Java-specific node parsing
- `parser_python.go`: Python-specific node parsing
- `utils.go`: SHA256-based ID generation, file operations

**sast-engine/graph/callgraph/** - Call graph and type inference
- `builder/builder.go`: 5-pass call graph construction algorithm
- `resolution/inference.go`: Type inference engine with bidirectional inference
- `resolution/return_type.go`: Return type extraction from AST
- `extraction/variables.go`: Variable assignment type tracking
- `registry/attribute.go`: Class attribute registry
- `registry/builtin.go`: Python builtin types registry

**sast-engine/cmd/** - CLI interface
- `serve.go`: MCP server for AI coding assistants
- `scan.go`: Scan project against local ruleset
- `ci.go`: CI/CD integration with rule loading from codepathfinder.dev
- `resolution_report.go`: Call resolution diagnostics
- `diagnose.go`: Taint analysis diagnostics

**sast-engine/model/** - AST data models
- `stmt.go`: Statement models (if/while/for/blocks)
- `expr.go`: Expression models
- `location.go`: Source location tracking for lazy loading

**sast-engine/analytics/** - Optional PostHog telemetry

## Critical Design Patterns

### Node ID Generation
All node IDs are deterministic SHA256 hashes to ensure consistency across runs:
```go
// Methods: method:<name>-<params>-<file>:<line>:<col>
GenerateMethodID("method:methodName", []string{params}, filepath)

// Expressions: <type>+<content>
GenerateSha256(exprType + node.Content(sourceCode))
```

This enables:
- Consistent results despite multi-threaded parsing
- Deduplication of identical constructs
- Reliable linking between method invocations and declarations

### Lazy Loading with SourceLocation
Nodes store `StartByte` and `EndByte` offsets instead of full code snippets:
```go
type Node struct {
    SourceLocation *SourceLocation // File path + byte offsets
}

func (n *Node) GetCodeSnippet() string {
    content := readFile(n.SourceLocation.File)
    return string(content[StartByte:EndByte])
}
```

This reduces memory usage from ~2.32 GB to ~2.18 GB for large codebases (27k+ methods). Code snippets are read on-demand, leveraging OS page caching for performance.

### Cartesian Product Query Optimization
Multi-entity queries (e.g., "find method md calling method target") generate exhaustive combinations:
```go
// Single entity: O(n) linear filtering
// Two entities: O(n²) pairwise matching with early pruning

for _, lhsNode := range typeIndex[selectList[0].Entity] {
    for _, rhsNode := range typeIndex[selectList[1].Entity] {
        if FilterEntities([]*Node{lhsNode, rhsNode}, expression) {
            validPairs = append(validPairs, []*Node{lhsNode, rhsNode})
        }
    }
}
```

**Performance tip**: Limit multi-entity queries to related types (e.g., method + invocation) to avoid exponential explosion.

### Worker Pool Concurrency
File parsing uses 5 concurrent workers to balance parallelism with overhead:
```go
// In initialize.go
numWorkers := 5
for i := 0; i < numWorkers; i++ {
    go worker(i + 1)
}
```

Each worker has its own tree-sitter parser instance to avoid thread-safety issues.

### Object Pooling
Environment maps are pooled to reduce GC pressure during query evaluation:
```go
var envMapPool = sync.Pool{
    New: func() interface{} {
        return make(map[string]interface{}, 10)
    },
}
```

Used in `query.go` during expression evaluation for thousands of nodes.

## Language Support

### CGO Dependency Requirement
This project **requires CGO** due to `go-tree-sitter` C bindings. Build fails with `CGO_ENABLED=0`. This affects:
- Cross-compilation (requires platform-specific CGO toolchains)
- Release automation (cannot use pure Go cross-compile)
- Docker-based builds recommended for releases

### Adding a New Language

1. **Add tree-sitter language package** to `go.mod`:
   ```go
   require github.com/smacker/go-tree-sitter/rust v0.0.0-...
   ```

2. **Update file extension mapping** in `graph/initialize.go`:
   ```go
   case ".rs":
       parser.SetLanguage(rust.GetLanguage())
   ```

3. **Create language-specific parser** file (e.g., `graph/parser_rust.go`):
   ```go
   func parseRustFunctionDefinition(node *sitter.Node, ...) *Node {
       // Extract Rust-specific AST details
   }
   ```

4. **Add node type handlers** in `graph/parser.go`:
   ```go
   case "function_item":
       if isRustSourceFile {
           currentContext = parseRustFunctionDefinition(...)
       }
   ```

5. **Extend query environment** in `graph/query.go`:
   ```go
   case "function_item":
       return map[string]interface{}{
           "getName": func() string { return node.Name },
           // ... other Rust-specific methods
       }
   ```

### Java vs Python Parsing Differences

**Java** (parser_java.go):
- Full method invocation tracking with parameter resolution
- Class inheritance and interface implementation
- Field declarations with visibility modifiers
- JavaDoc parsing with structured tags
- Annotation support

**Python** (parser_python.go):
- Function definitions with argument tracking
- Class definitions with inheritance
- Variable assignments (no type information)
- Simplified compared to Java (no invocation linking yet)

## Python DSL Rules (v1.0.0+)

Code Pathfinder uses **Python DSL** for writing security rules. Rules are Python functions that query the call graph using the MCP interface.

### Example Rule
```python
from pathfinder import Rule, Severity

@Rule(
    id="CUSTOM-001",
    name="Detect SQL Injection Risk",
    severity=Severity.HIGH,
    description="Methods that process payments should not directly execute SQL queries"
)
def detect_sql_injection(graph):
    """Find payment methods calling SQL execution."""
    payment_methods = graph.find_symbol(name="processPayment", type="method")

    for method in payment_methods:
        callees = graph.get_callees(function=method.fqn)
        for callee in callees:
            if "executeQuery" in callee.target_fqn or "execute" in callee.target_fqn:
                yield {
                    "file": method.file,
                    "line": callee.call_line,
                    "message": f"SQL execution in payment method: {method.fqn}"
                }
```

### Available MCP Tools in Rules
- `find_symbol(name, type)` - Find symbols by name and type
- `get_callees(function)` - Get functions called by a function
- `get_callers(function)` - Get functions that call a function
- `get_call_details(caller, callee)` - Get specific call site details
- `find_module(name)` - Find modules by name
- `list_modules()` - List all modules

### Running Rules
```bash
# Single rule file
pathfinder scan --rules rules/my_rule.py --project /path/to/project

# Directory of rules
pathfinder scan --rules rules/ --project /path/to/project

# Remote ruleset bundle
pathfinder scan --ruleset docker/security --project /path/to/project
```

## Testing Patterns

### Table-Driven Tests
Most test files use table-driven testing with `testify/assert`:
```go
tests := []struct {
    name     string
    input    X
    expected Y
}{
    {name: "test case 1", input: ..., expected: ...},
    {name: "test case 2", input: ..., expected: ...},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := functionUnderTest(tt.input)
        assert.Equal(t, tt.expected, result)
    })
}
```

### Test Organization
- `graph/callgraph/builder/builder_test.go` - Call graph builder tests
- `graph/callgraph/resolution/*_test.go` - Type inference and resolution tests
- `cmd/scan_test.go` - Scan command tests
- `cmd/ci_test.go` - CI command tests
- `cmd/resolution_report_test.go` - Resolution diagnostics tests

### Type Matching in Tests
When creating test nodes, ensure types match the Node struct:
```go
// Correct
LineNumber: uint32(10)

// Incorrect (compilation error)
LineNumber: 10
```

## Important Non-Obvious Relationships

### Method Invocation Linking
After AST parsing, method invocations are linked to declarations:
```go
// In graph.go
func (cg *CodeGraph) LinkMethodInvocations() {
    for _, invocation := range invocations {
        declaration := findMethodBySignature(invocation.Name, invocation.Parameters)
        if declaration != nil {
            declaration.HasAccess = true
            cg.AddEdge(invocation.ID, declaration.ID)
        }
    }
}
```

This enables queries like "find unused methods" by checking `hasAccess() == false`.

### Expression Environment Lazy Binding
Query expressions use `expr-lang` with method call syntax:
```go
// Query: md.getName() == "test"
// Compiled to: env["getName"]() == "test"

envMap := map[string]interface{}{
    "getName": func() string { return node.Name },
}
program := expr.Compile(expression, expr.Env(envMap))
expr.Run(program, envMap) // Returns bool
```

Methods are bound at runtime to actual node fields, enabling type-safe queries without reflection.

### SARIF Report Generation
CI mode generates SARIF reports for GitHub Advanced Security:
```go
// In cmd/ci.go
run := sarif.NewRunWithInformationURI("Code Pathfinder", "https://codepathfinder.dev")
result := run.CreateResultForRule(ruleID)
    .WithMessage(sarif.NewTextMessage(description))
    .AddLocation(sarif.NewLocationWithPhysicalLocation(
        sarif.NewPhysicalLocation().
            WithArtifactLocation(sarif.NewSimpleArtifactLocation(file)).
            WithRegion(sarif.NewSimpleRegion(lineNumber, lineNumber)),
    ))
```

SARIF output integrates with GitHub Code Scanning, VSCode, and other security platforms.

### Result Determinism
Results are sorted to ensure consistency across runs:
```go
// Sort by File → LineNumber → ID
sort.SliceStable(results, func(i, j int) bool {
    if nodeI.File != nodeJ.File {
        return nodeI.File < nodeJ.File
    }
    if nodeI.LineNumber != nodeJ.LineNumber {
        return nodeI.LineNumber < nodeJ.LineNumber
    }
    return nodeI.ID < nodeJ.ID
})
```

This counteracts non-determinism from multi-threaded parsing.

## Release and Versioning

### Version Management
Version is stored in `sast-engine/VERSION` and injected at build time:
```gradle
// In build.gradle
commandLine 'go', 'build', '-ldflags',
    "-X ...cmd.Version=${projectVersion} -X ...cmd.GitCommit=${gitCommit}"
```

Both `VERSION` file and `package.json` must be updated together when bumping versions.

### NPM Package Distribution
The npm package downloads pre-built binaries from GitHub releases:
```json
{
  "goBinary": {
    "url": "https://github.com/.../releases/download/v{{version}}/pathfinder-{{platform}}-{{arch}}.tar.gz"
  }
}
```

Releases must include binaries for linux-amd64, darwin-amd64, darwin-arm64, and windows-amd64.

## Performance Considerations

### Memory Usage
- Small codebase (<1k functions): ~100 MB
- Large codebase (27k functions): ~2.18 GB with lazy loading
- MCP server keeps index in memory for fast queries
- Call graph includes functions, edges, and type information

### Indexing Time
- Graph building: ~5 seconds for 27k functions (parallel workers)
- Return type extraction: Parallel across all files
- Variable assignment extraction: Parallel across all files
- Class attribute extraction: Parallel across all files

### Optimization Tips
1. Use `--verbose` flag to see indexing statistics
2. Large projects benefit from more CPU cores (PATHFINDER_MAX_WORKERS env var)
3. MCP queries are fast (index pre-built)
4. Python DSL rules run sequentially - keep rules focused

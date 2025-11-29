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
# Interactive query mode
./build/go/pathfinder query --project <path> --stdin

# CI mode (loads rules from remote/local)
./build/go/pathfinder ci --project <path> --ruleset cpf/java --output sarif

# Scan mode
./build/go/pathfinder scan --project <path> --ruleset <path_to_rules>

# With pagination
./build/go/pathfinder query --project <path> --page 1 --size 10
```

### Running a Single Test
```bash
go test -v -run TestPaginationSorting ./cmd/
```

## High-Level Architecture

Code Pathfinder is a multi-stage security analysis pipeline:

```
Source Files (.java, .py)
    ↓
Tree-Sitter AST Parsing (5 parallel workers)
    ↓
Code Graph (Nodes + Edges)
    ↓
Query Language (ANTLR parser)
    ↓
Query Engine (expr-lang evaluation)
    ↓
Output Formats (JSON, SARIF, Table)
```

### Core Packages

**sast-engine/graph/** - Code graph construction and management
- `initialize.go`: Multi-threaded file parsing with 5 workers
- `parser.go`: AST traversal orchestrator (language-agnostic entry point)
- `parser_java.go`: Java-specific node parsing
- `parser_python.go`: Python-specific node parsing
- `query.go`: Query execution engine with Cartesian product optimization
- `utils.go`: SHA256-based ID generation, file operations

**sast-engine/antlr/** - Query language parsing
- `Query.g4`: ANTLR grammar for PathFinder query language
- `listener_impl.go`: Semantic analysis of parsed queries

**sast-engine/cmd/** - CLI interface
- `query.go`: Interactive/batch query execution with pagination
- `ci.go`: CI/CD integration with rule loading from codepathfinder.dev
- `scan.go`: Scan project against local ruleset

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

## Query Language

### Syntax
```
FROM <entity_type> AS <alias> [, <entity_type> AS <alias>]
[WHERE <expression>]
SELECT <output_fields>
```

### Entity Types
- `method_declaration` - Java methods, Python functions
- `class_declaration` - Java/Python classes
- `variable_declaration` - Java fields, Python variables
- `method_invocation` - Java method calls
- `*_expression` - Binary operations (add_expression, eq_expression, etc.)
- `*_statement` - Control flow (if_statement, while_statement, etc.)

### Entity Environment Methods

Each entity type exposes specific methods in WHERE/SELECT clauses:

**method_declaration**:
- `getName()`, `getVisibility()`, `getReturnType()`
- `getArgumentTypes()`, `getArgumentName()`
- `getDoc()`, `getAnnotation()`, `hasAccess()`

**class_declaration**:
- `getName()`, `getSuperClass()`, `getInterface()`
- `getVisibility()`, `getDoc()`

**variable_declaration**:
- `getName()`, `getVisibility()`
- `getVariableDataType()`, `getVariableValue()`

**method_invocation**:
- `getName()`, `getAccessFromClass()`, `getAccessFromMethod()`

### Query Execution Flow
```
Query String
    ↓
ANTLR Parse → Query AST
    ↓
Generate Cartesian Product of Entity Types
    ↓
Build Environment Map (pooled)
    ↓
Compile Expression (expr-lang)
    ↓
Filter Entities (evaluate each combination)
    ↓
Sort Results (File → LineNumber → ID)
    ↓
Apply Pagination (if --page/--size specified)
    ↓
Format Output (json/table/sarif)
```

### Example Query
```
FROM method_declaration AS md, method_invocation AS mi
WHERE md.getName() == "processPayment" && mi.getName() == "executeQuery"
SELECT md, mi, "SQL injection risk in payment processing"
```

This finds methods named `processPayment` that invoke methods named `executeQuery`, potentially indicating SQL injection vulnerabilities.

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
- `graph/query_test.go` - Query execution tests
- `graph/initialize_test.go` - Graph initialization tests
- `antlr/listener_impl_test.go` - Query parsing tests
- `cmd/query_test.go` - CLI and pagination tests

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

### Pagination Determinism
Results are sorted **before** pagination to ensure consistency across runs:
```go
// In cmd/query.go
sort.SliceStable(pairs, func(i, j int) bool {
    // Sort by File → LineNumber → ID
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

## Query Performance Considerations

### Memory Usage
- Small codebase (<1k methods): ~100 MB
- Large codebase (27k methods): ~2.18 GB with lazy loading
- Pagination does NOT reduce memory (sorting requires all results in memory)
- Pagination reduces output size (37 MB → 4.7 KB for page size 10)

### Execution Time
- Graph building: ~5 seconds for 27k methods (5 workers)
- Query execution: <1 second for simple queries
- Multi-entity queries: O(n²) for 2 entities, can be slow for large graphs

### Optimization Tips
1. Use specific entity types in FROM clause (not wildcards)
2. Add WHERE conditions that filter early
3. Avoid multi-entity queries on unrelated types
4. Use pagination for large result sets (output size, not memory)
5. Run with `--verbose` to debug slow queries

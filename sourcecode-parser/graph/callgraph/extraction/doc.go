// Package extraction provides AST-based code extraction utilities for
// Python source code.
//
// This package uses tree-sitter to extract program statements from Python
// source files, converting AST nodes into structured Statement objects for
// analysis.
//
// Example:
//
//	statements, err := extraction.ExtractStatements(sourceCode, "myFunction")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, stmt := range statements {
//	    fmt.Printf("Statement type: %s\n", stmt.Type)
//	}
package extraction

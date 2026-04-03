package builder

import (
	"os"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
)

// ReadFileBytes reads a file and returns its contents as a byte slice.
// Helper function for reading source code.
//
// Parameters:
//   - filePath: path to the file (can be relative or absolute)
//
// Returns:
//   - File contents as byte slice
//   - error if file cannot be read
func ReadFileBytes(filePath string) ([]byte, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(absPath)
}

// FindFunctionAtLine searches for a function definition at the specified line number.
// Returns the tree-sitter node for the function, or nil if not found.
//
// This function recursively traverses the AST tree to find a function or method
// definition node at the given line number.
//
// Parameters:
//   - root: the root tree-sitter node to search from
//   - lineNumber: the line number to search for (1-indexed)
//
// Returns:
//   - tree-sitter node for the function definition, or nil if not found
func FindFunctionAtLine(root *sitter.Node, lineNumber uint32) *sitter.Node {
	if root == nil {
		return nil
	}

	// Check if this node is a function definition at the target line
	if (root.Type() == "function_definition" || root.Type() == "method_declaration") &&
		root.StartPoint().Row+1 == lineNumber {
		return root
	}

	// Recursively search children
	for i := 0; i < int(root.ChildCount()); i++ {
		if result := FindFunctionAtLine(root.Child(i), lineNumber); result != nil {
			return result
		}
	}

	return nil
}

// findGoNodeByByteRange finds a Go function or method node matching the given byte range.
// Analogous to FindFunctionAtLine (which searches Python by line number).
//
// Go nodes use SourceLocation{StartByte, EndByte} set by setGoSourceLocation
// in graph/parser_golang.go. This function matches on those byte offsets.
//
// Note: Go uses "function_declaration" (not Python's "function_definition")
// and "method_declaration" for methods with receivers.
//
// Parameters:
//   - root: the root tree-sitter node to search from
//   - startByte: expected StartByte of the function node
//   - endByte: expected EndByte of the function node
//
// Returns:
//   - tree-sitter node for the function/method, or nil if not found
func findGoNodeByByteRange(root *sitter.Node, startByte, endByte uint32) *sitter.Node {
	if root == nil {
		return nil
	}

	if (root.Type() == "function_declaration" || root.Type() == "method_declaration") &&
		root.StartByte() == startByte && root.EndByte() == endByte {
		return root
	}

	for i := 0; i < int(root.ChildCount()); i++ {
		if result := findGoNodeByByteRange(root.Child(i), startByte, endByte); result != nil {
			return result
		}
	}

	return nil
}

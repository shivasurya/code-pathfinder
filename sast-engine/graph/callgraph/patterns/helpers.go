package patterns

import (
	"os"

	sitter "github.com/smacker/go-tree-sitter"
)

// readFileBytes reads a file and returns its contents as bytes.
func readFileBytes(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// findFunctionAtLine finds a function node at a specific line number in the AST.
func findFunctionAtLine(root *sitter.Node, lineNumber uint32) *sitter.Node {
	if root == nil {
		return nil
	}

	// If this node is a function_definition and it starts at the target line
	if root.Type() == "function_definition" && root.StartPoint().Row+1 == lineNumber {
		return root
	}

	// Recursively search child nodes
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if result := findFunctionAtLine(child, lineNumber); result != nil {
			return result
		}
	}

	return nil
}

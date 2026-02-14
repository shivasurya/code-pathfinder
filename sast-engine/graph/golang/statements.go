package golang

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// ReturnInfo represents a parsed Go return_statement.
type ReturnInfo struct {
	Values     []string // expression strings
	LineNumber uint32
	StartByte  uint32
	EndByte    uint32
}

// ParseReturnStatement parses a Go return_statement node into a ReturnInfo.
// Example: return nil, err.
func ParseReturnStatement(node *sitter.Node, sourceCode []byte) *ReturnInfo {
	if node == nil || node.Type() != "return_statement" {
		return nil
	}

	info := &ReturnInfo{
		Values:     []string{},
		LineNumber: uint32(node.StartPoint().Row) + 1,
		StartByte:  node.StartByte(),
		EndByte:    node.EndByte(),
	}

	// Find expression_list child containing return values
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "expression_list" {
			// Iterate expressions in the list
			for j := 0; j < int(child.NamedChildCount()); j++ {
				expr := child.NamedChild(j)
				if expr != nil {
					info.Values = append(info.Values, expr.Content(sourceCode))
				}
			}
			break
		}
	}

	// Handle single return value (not wrapped in expression_list)
	if len(info.Values) == 0 {
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child != nil {
				info.Values = append(info.Values, child.Content(sourceCode))
			}
		}
	}

	return info
}

// ForInfo represents a parsed Go for_statement.
type ForInfo struct {
	IsRange    bool   // true for range, false for C-style
	Condition  string // for C-style
	Init       string // for C-style (first child of for_clause, NOT a field)
	Update     string // for C-style
	Left       string // for range (LHS variables)
	Right      string // for range (iterable)
	LineNumber uint32
	StartByte  uint32
	EndByte    uint32
}

// ParseForStatement parses a Go for_statement node into a ForInfo.
// Handles both C-style loops (for i := 0; i < 10; i++) and range loops (for _, v := range items).
func ParseForStatement(node *sitter.Node, sourceCode []byte) *ForInfo {
	if node == nil || node.Type() != "for_statement" {
		return nil
	}

	info := &ForInfo{
		LineNumber: uint32(node.StartPoint().Row) + 1,
		StartByte:  node.StartByte(),
		EndByte:    node.EndByte(),
	}

	// Find for_clause, range_clause, or direct condition (while-style)
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)

		switch child.Type() {
		case "for_clause":
			// C-style for loop
			info.IsRange = false

			// Get condition field
			condNode := child.ChildByFieldName("condition")
			if condNode != nil {
				info.Condition = condNode.Content(sourceCode)
			}

			// Get update field
			updNode := child.ChildByFieldName("update")
			if updNode != nil {
				info.Update = updNode.Content(sourceCode)
			}

			// Get init (first named child, NOT a field)
			initNode := child.NamedChild(0)
			if initNode != nil && initNode.Type() != "block" {
				info.Init = initNode.Content(sourceCode)
			}

			return info

		case "range_clause":
			// Range for loop
			info.IsRange = true

			// Get left field (LHS variables)
			leftNode := child.ChildByFieldName("left")
			if leftNode != nil {
				info.Left = leftNode.Content(sourceCode)
			}

			// Get right field (iterable)
			rightNode := child.ChildByFieldName("right")
			if rightNode != nil {
				info.Right = rightNode.Content(sourceCode)
			}

			return info

		case "block":
			// Skip block nodes
			continue

		default:
			// While-style loop: condition is direct child (not in for_clause)
			info.IsRange = false
			info.Init = child.Content(sourceCode)
			// For while-style loops, store condition in Init field
			return info
		}
	}

	return info
}

// IfInfo represents a parsed Go if_statement.
type IfInfo struct {
	Condition  string
	LineNumber uint32
	StartByte  uint32
	EndByte    uint32
}

// ParseIfStatement parses a Go if_statement node into an IfInfo.
// Example: if err != nil { return err }.
func ParseIfStatement(node *sitter.Node, sourceCode []byte) *IfInfo {
	if node == nil || node.Type() != "if_statement" {
		return nil
	}

	info := &IfInfo{
		LineNumber: uint32(node.StartPoint().Row) + 1,
		StartByte:  node.StartByte(),
		EndByte:    node.EndByte(),
	}

	// Get condition field
	condNode := node.ChildByFieldName("condition")
	if condNode != nil {
		info.Condition = condNode.Content(sourceCode)
	}

	return info
}

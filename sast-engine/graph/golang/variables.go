package golang

import sitter "github.com/smacker/go-tree-sitter"

// VarInfo holds extracted information from a Go variable/constant/assignment.
// Used by the dispatcher in parser_golang.go to create graph.Node instances.
type VarInfo struct {
	Name       string // "name", "Pi", "x"
	Value      string // RHS content for Node.VariableValue: `"Alice"`, "3.14", "x + 1"
	TypeName   string // explicit type annotation: "string", "int", "" if omitted
	Visibility string // "public" or "private"
	LineNumber uint32
	StartByte  uint32
	EndByte    uint32
	IsMulti    bool // true when part of multi-LHS assignment (x, y := ...)
}

// ParseVarDeclaration extracts variable information from a Go var_declaration node.
// Handles: var x int, var x = 1, var ( x int; y string ), var x, y int
func ParseVarDeclaration(node *sitter.Node, sourceCode []byte) []*VarInfo {
	var vars []*VarInfo

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)

		switch child.Type() {
		case "var_spec":
			vars = append(vars, extractVarSpec(child, sourceCode)...)
		case "var_spec_list":
			// Grouped var ( ... )
			for j := 0; j < int(child.NamedChildCount()); j++ {
				spec := child.NamedChild(j)
				if spec.Type() == "var_spec" {
					vars = append(vars, extractVarSpec(spec, sourceCode)...)
				}
			}
		}
	}

	return vars
}

// extractVarSpec extracts variable info from a single var_spec node.
// Handles multi-name declarations like "var x, y int".
func extractVarSpec(spec *sitter.Node, sourceCode []byte) []*VarInfo {
	var vars []*VarInfo

	// Collect all identifier names
	var names []string
	for i := 0; i < int(spec.NamedChildCount()); i++ {
		child := spec.NamedChild(i)
		if child.Type() == "identifier" {
			names = append(names, child.Content(sourceCode))
		}
	}

	// Get type and value
	typeName := ""
	typeNode := spec.ChildByFieldName("type")
	if typeNode != nil {
		typeName = typeNode.Content(sourceCode)
	}

	value := ""
	valueNode := spec.ChildByFieldName("value")
	if valueNode != nil {
		value = valueNode.Content(sourceCode)
	}

	// Create VarInfo for each name
	for _, name := range names {
		vars = append(vars, &VarInfo{
			Name:       name,
			Value:      value,
			TypeName:   typeName,
			Visibility: DetermineVisibility(name),
			LineNumber: spec.StartPoint().Row + 1,
			StartByte:  spec.StartByte(),
			EndByte:    spec.EndByte(),
			IsMulti:    false,
		})
	}

	return vars
}

// ParseShortVarDeclaration extracts variable information from a short_var_declaration node.
// Handles: name := "Alice", x, y := foo(), _, err := foo()
func ParseShortVarDeclaration(node *sitter.Node, sourceCode []byte) []*VarInfo {
	var vars []*VarInfo

	// Get left-hand side identifiers
	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return vars
	}

	var names []string
	totalCount := 0
	for i := 0; i < int(leftNode.NamedChildCount()); i++ {
		child := leftNode.NamedChild(i)
		if child.Type() == "identifier" {
			totalCount++
			name := child.Content(sourceCode)
			// Skip blank identifier
			if name != "_" {
				names = append(names, name)
			}
		}
	}

	// Get right-hand side value
	value := ""
	rightNode := node.ChildByFieldName("right")
	if rightNode != nil {
		value = rightNode.Content(sourceCode)
	}

	// Determine if multi-assignment
	isMulti := totalCount > 1

	// Create VarInfo for each non-blank identifier
	for _, name := range names {
		vars = append(vars, &VarInfo{
			Name:       name,
			Value:      value,
			TypeName:   "",
			Visibility: DetermineVisibility(name),
			LineNumber: node.StartPoint().Row + 1,
			StartByte:  node.StartByte(),
			EndByte:    node.EndByte(),
			IsMulti:    isMulti,
		})
	}

	return vars
}

// ParseConstDeclaration extracts constant information from a const_declaration node.
// Handles: const Pi = 3.14, const ( A = iota; B; C )
func ParseConstDeclaration(node *sitter.Node, sourceCode []byte) []*VarInfo {
	var consts []*VarInfo

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "const_spec" {
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			name := nameNode.Content(sourceCode)

			value := ""
			valueNode := child.ChildByFieldName("value")
			if valueNode != nil {
				value = valueNode.Content(sourceCode)
			}

			consts = append(consts, &VarInfo{
				Name:       name,
				Value:      value,
				TypeName:   "",
				Visibility: DetermineVisibility(name),
				LineNumber: child.StartPoint().Row + 1,
				StartByte:  child.StartByte(),
				EndByte:    child.EndByte(),
				IsMulti:    false,
			})
		}
	}

	return consts
}

// ParseAssignment extracts assignment information from an assignment_statement node.
// Handles: x = x + 1, x, y = 1, 2
func ParseAssignment(node *sitter.Node, sourceCode []byte) []*VarInfo {
	var vars []*VarInfo

	// Get left-hand side identifiers
	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return vars
	}

	var names []string
	for i := 0; i < int(leftNode.NamedChildCount()); i++ {
		child := leftNode.NamedChild(i)
		// Only process simple identifiers, skip subscript, selector_expression, etc.
		if child.Type() == "identifier" {
			names = append(names, child.Content(sourceCode))
		}
	}

	// Get right-hand side value
	value := ""
	rightNode := node.ChildByFieldName("right")
	if rightNode != nil {
		value = rightNode.Content(sourceCode)
	}

	// Determine if multi-assignment
	isMulti := len(names) > 1

	// Create VarInfo for each identifier
	for _, name := range names {
		vars = append(vars, &VarInfo{
			Name:       name,
			Value:      value,
			TypeName:   "",
			Visibility: DetermineVisibility(name),
			LineNumber: node.StartPoint().Row + 1,
			StartByte:  node.StartByte(),
			EndByte:    node.EndByte(),
			IsMulti:    isMulti,
		})
	}

	return vars
}

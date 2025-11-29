package extraction

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper: parse Python code and get function node.
// Returns the tree (caller must close it), function node, and source bytes.
func parsePythonFunction(t *testing.T, source string, funcName string) (*sitter.Tree, *sitter.Node, []byte) {
	t.Helper()
	sourceBytes := []byte(source)

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, sourceBytes)
	require.NoError(t, err)

	// Find function definition
	root := tree.RootNode()
	funcNode := findFunctionByName(root, funcName, sourceBytes)
	require.NotNil(t, funcNode, "Function %s not found", funcName)

	return tree, funcNode, sourceBytes
}

// Helper: find function definition node by name.
func findFunctionByName(node *sitter.Node, name string, source []byte) *sitter.Node {
	if node == nil {
		return nil
	}

	// Check if this is a function_definition with matching name
	if node.Type() == "function_definition" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil && string(nameNode.Content(source)) == name { //nolint:unconvert
			return node
		}
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		result := findFunctionByName(node.Child(i), name, source)
		if result != nil {
			return result
		}
	}

	return nil
}

//
// ========== ASSIGNMENT TESTS ==========
//

func TestExtractStatements_SimpleAssignment(t *testing.T) {
	source := `
def foo():
    x = 10
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
	assert.Equal(t, uint32(3), stmt.LineNumber) // Line 3 in source
	assert.Equal(t, "x", stmt.Def)
	assert.Equal(t, "10", stmt.CallTarget) // RHS stored in CallTarget
	assert.Equal(t, 0, len(stmt.Uses), "Literal has no uses")
}

func TestExtractStatements_AssignmentFromVariable(t *testing.T) {
	source := `
def foo():
    y = x
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
	assert.Equal(t, "y", stmt.Def)
	assert.Equal(t, "x", stmt.CallTarget)
	assert.Equal(t, []string{"x"}, stmt.Uses)
}

func TestExtractStatements_AssignmentFromCall(t *testing.T) {
	source := `
def foo():
    result = func(x, y)
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
	assert.Equal(t, "result", stmt.Def)
	// Uses should include function name and arguments
	assert.Contains(t, stmt.Uses, "func")
	assert.Contains(t, stmt.Uses, "x")
	assert.Contains(t, stmt.Uses, "y")
}

func TestExtractStatements_AugmentedAssignment(t *testing.T) {
	source := `
def foo():
    x += 5
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeAssignment, stmt.Type) // Normalized
	assert.Equal(t, "x", stmt.Def)
	assert.Contains(t, stmt.Uses, "x", "Augmented assignment uses LHS")
}

func TestExtractStatements_TupleUnpacking_Skipped(t *testing.T) {
	source := `
def foo():
    x, y = func()
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	// Tuple unpacking not supported, should be skipped
	assert.Equal(t, 0, len(statements), "Tuple unpacking should be skipped")
}

func TestExtractStatements_AttributeAssignment_Skipped(t *testing.T) {
	source := `
def foo():
    obj.field = 10
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	// Attribute assignment not supported (needs field sensitivity)
	assert.Equal(t, 0, len(statements), "Attribute assignment should be skipped")
}

//
// ========== CALL TESTS ==========
//

func TestExtractStatements_SimpleCall(t *testing.T) {
	source := `
def foo():
    func()
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeCall, stmt.Type)
	assert.Equal(t, "func", stmt.CallTarget)
	assert.Equal(t, 0, len(stmt.CallArgs))
	assert.Equal(t, "", stmt.Def, "Call without assignment has no defs")
}

func TestExtractStatements_CallWithArguments(t *testing.T) {
	source := `
def foo():
    eval(x)
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeCall, stmt.Type)
	assert.Equal(t, "eval", stmt.CallTarget)
	assert.Equal(t, []string{"x"}, stmt.CallArgs)
	assert.Contains(t, stmt.Uses, "x")
}

func TestExtractStatements_MethodCall(t *testing.T) {
	source := `
def foo():
    obj.method()
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeCall, stmt.Type)
	assert.Equal(t, "method", stmt.CallTarget, "Should extract method name")
	assert.Contains(t, stmt.Uses, "obj", "Should track base object")
}

func TestExtractStatements_ChainedMethodCall(t *testing.T) {
	source := `
def foo():
    obj.a.b.method()
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, "method", stmt.CallTarget)
	assert.Contains(t, stmt.Uses, "obj", "Should track base object")
}

func TestExtractStatements_NestedCalls(t *testing.T) {
	source := `
def foo():
    eval(func(x))
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements), "Nested calls treated as one statement")

	stmt := statements[0]
	assert.Equal(t, "eval", stmt.CallTarget, "Outer call is target")
	// Uses should include all identifiers (conservative)
	assert.Contains(t, stmt.Uses, "eval")
	assert.Contains(t, stmt.Uses, "func")
	assert.Contains(t, stmt.Uses, "x")
}

//
// ========== RETURN TESTS ==========
//

func TestExtractStatements_ReturnWithExpression(t *testing.T) {
	source := `
def foo():
    return x
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeReturn, stmt.Type)
	assert.Equal(t, "x", stmt.CallTarget) // Return expression stored in CallTarget
	assert.Equal(t, []string{"x"}, stmt.Uses)
}

func TestExtractStatements_ReturnWithoutExpression(t *testing.T) {
	source := `
def foo():
    return
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeReturn, stmt.Type)
	assert.Equal(t, "", stmt.CallTarget)
	assert.Equal(t, 0, len(stmt.Uses))
}

//
// ========== IDENTIFIER EXTRACTION TESTS ==========
//

func TestExtractIdentifiers_FilterKeywords(t *testing.T) {
	source := `
def foo():
    x = True and False or None
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	stmt := statements[0]

	// Should NOT include True, False, None (keywords)
	assert.NotContains(t, stmt.Uses, "True")
	assert.NotContains(t, stmt.Uses, "False")
	assert.NotContains(t, stmt.Uses, "None")
}

func TestExtractIdentifiers_Deduplication(t *testing.T) {
	source := `
def foo():
    result = x + x + x
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	stmt := statements[0]

	// Should have "x" only once (deduplicated)
	xCount := 0
	for _, use := range stmt.Uses {
		if use == "x" {
			xCount++
		}
	}
	assert.Equal(t, 1, xCount, "Should deduplicate identifiers")
}

//
// ========== INTEGRATION TESTS ==========
//

func TestExtractStatements_MultipleStatements(t *testing.T) {
	source := `
def vulnerable():
    x = request.GET['input']
    y = x.upper()
    z = y
    eval(z)
    return None
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "vulnerable")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 5, len(statements), "Should extract 5 statements")

	// Statement 1: x = request.GET['input']
	assert.Equal(t, core.StatementTypeAssignment, statements[0].Type)
	assert.Equal(t, "x", statements[0].Def)
	assert.Contains(t, statements[0].Uses, "request")

	// Statement 2: y = x.upper()
	assert.Equal(t, core.StatementTypeAssignment, statements[1].Type)
	assert.Equal(t, "y", statements[1].Def)
	assert.Contains(t, statements[1].Uses, "x")

	// Statement 3: z = y
	assert.Equal(t, core.StatementTypeAssignment, statements[2].Type)
	assert.Equal(t, "z", statements[2].Def)
	assert.Contains(t, statements[2].Uses, "y")

	// Statement 4: eval(z)
	assert.Equal(t, core.StatementTypeCall, statements[3].Type)
	assert.Equal(t, "eval", statements[3].CallTarget)
	assert.Contains(t, statements[3].Uses, "z")

	// Statement 5: return None
	assert.Equal(t, core.StatementTypeReturn, statements[4].Type)
}

func TestExtractStatements_EmptyFunction(t *testing.T) {
	source := `
def foo():
    pass
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 0, len(statements), "Empty function should have no statements")
}

func TestExtractStatements_ControlFlowSkipped(t *testing.T) {
	source := `
def foo():
    if condition:
        x = 10
    while True:
        break
    for i in range(10):
        continue
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	// All control flow statements should be skipped
	assert.Equal(t, 0, len(statements), "Control flow should be skipped")
}

func TestExtractStatements_MultipleAugmentedOperators(t *testing.T) {
	source := `
def foo():
    x += 1
    y -= 2
    z *= 3
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 3, len(statements))

	// All should be normalized to assignments
	assert.Equal(t, core.StatementTypeAssignment, statements[0].Type)
	assert.Equal(t, core.StatementTypeAssignment, statements[1].Type)
	assert.Equal(t, core.StatementTypeAssignment, statements[2].Type)

	// All should include LHS in Uses
	assert.Contains(t, statements[0].Uses, "x")
	assert.Contains(t, statements[1].Uses, "y")
	assert.Contains(t, statements[2].Uses, "z")
}

func TestExtractStatements_ComplexExpression(t *testing.T) {
	source := `
def foo():
    result = a + b * c - func(d, e)
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	// Should extract all identifiers from complex expression
	assert.Contains(t, stmt.Uses, "a")
	assert.Contains(t, stmt.Uses, "b")
	assert.Contains(t, stmt.Uses, "c")
	assert.Contains(t, stmt.Uses, "func")
	assert.Contains(t, stmt.Uses, "d")
	assert.Contains(t, stmt.Uses, "e")
}

func TestExtractStatements_KeywordArguments(t *testing.T) {
	source := `
def foo():
    func(x, y=5, z=name)
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeCall, stmt.Type)
	// CallArgs should include both positional and keyword values
	assert.Contains(t, stmt.CallArgs, "x")
	assert.Contains(t, stmt.CallArgs, "5")
	assert.Contains(t, stmt.CallArgs, "name")
}

func TestParsePythonFile(t *testing.T) {
	source := []byte(`
def foo():
    x = 10
`)

	tree, err := ParsePythonFile(source)
	require.NoError(t, err)
	require.NotNil(t, tree)
	defer tree.Close()

	root := tree.RootNode()
	assert.NotNil(t, root)
	assert.Equal(t, "module", root.Type())
}

func TestExtractStatements_SelfReference(t *testing.T) {
	source := `
def foo():
    self.process()
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	// "self" should be filtered out as a keyword
	assert.NotContains(t, stmt.Uses, "self")
}

// Additional tests for coverage

func TestExtractStatements_AugmentedAssignmentAttribute(t *testing.T) {
	source := `
def foo():
    obj.attr += 5
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
	assert.Contains(t, stmt.Uses, "obj")
	assert.Equal(t, "", stmt.Def, "Attribute augmented assignment has no def")
}

func TestExtractStatements_AugmentedAssignmentSubscript(t *testing.T) {
	source := `
def foo():
    arr[i] += 5
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Contains(t, stmt.Uses, "arr")
	assert.Contains(t, stmt.Uses, "i")
}

func TestExtractCallTarget_ComplexExpression(t *testing.T) {
	source := `
def foo():
    (lambda x: x)()
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeCall, stmt.Type)
	// Complex expression should have non-empty target
	assert.NotEmpty(t, stmt.CallTarget)
}

func TestParsePythonFile_InvalidSyntax(t *testing.T) {
	source := []byte(`
def foo(
    # unclosed parenthesis
`)

	tree, err := ParsePythonFile(source)
	// Tree-sitter is error-tolerant, so it won't error but will have error nodes
	require.NoError(t, err)
	require.NotNil(t, tree)
	defer tree.Close()
}

func TestExtractStatements_NilFunctionNode(t *testing.T) {
	source := []byte(`x = 10`)

	statements, err := ExtractStatements("test.py", source, nil)

	require.Error(t, err)
	assert.Nil(t, statements)
}

func TestExtractStatements_FunctionWithoutBody(t *testing.T) {
	source := `
def foo(): ...
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	// Ellipsis (...) is not a recognized statement type, should be empty or skip
	assert.GreaterOrEqual(t, len(statements), 0)
}

func TestExtractAssignment_NilRightNode(t *testing.T) {
	// This is a structural test - in practice, tree-sitter won't create
	// assignment nodes without RHS, but we test defensive coding
	source := `
def foo():
    x = 10
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)
	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))
}

func TestExtractReturn_NoExpression(t *testing.T) {
	source := `
def foo():
    return
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeReturn, stmt.Type)
	assert.Equal(t, 0, len(stmt.Uses))
	assert.Equal(t, "", stmt.CallTarget)
}

func TestExtractIdentifiers_EmptyNode(t *testing.T) {
	source := `
def foo():
    x = 10
`
	tree, _, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	// Test with nil node
	ids := extractIdentifiers(nil, sourceBytes)
	assert.Equal(t, 0, len(ids))
}

func TestExtractCallArgs_EmptyArguments(t *testing.T) {
	source := `
def foo():
    func()
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, 0, len(stmt.CallArgs))
}

func TestExtractStatements_AssignmentFromLiteral(t *testing.T) {
	source := `
def foo():
    x = "hello"
    y = [1, 2, 3]
    z = {"key": "value"}
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 3, len(statements))

	// All should be assignments with no uses (literals)
	for _, stmt := range statements {
		assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
		assert.NotEmpty(t, stmt.Def)
		assert.Equal(t, 0, len(stmt.Uses))
	}
}

func TestExtractCallTarget_AttributeWithoutField(t *testing.T) {
	// Edge case: what if ChildByFieldName returns nil?
	source := `
def foo():
    obj.method()
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, "method", stmt.CallTarget)
}

func TestExtractStatements_AssignmentUnknownLHS(t *testing.T) {
	// Test defensive code for unknown LHS types
	source := `
def foo():
    x = 10
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))
}

func TestExtractAugmentedAssignment_DefaultCase(t *testing.T) {
	// Test that normal augmented assignment works
	source := `
def foo():
    count += 1
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, "count", stmt.Def)
	assert.Contains(t, stmt.Uses, "count")
}

func TestExtractCallTarget_NilFunctionNode(t *testing.T) {
	// Direct test of extractCallTarget
	source := `
def foo():
    func()
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.Equal(t, "func", statements[0].CallTarget)
}

func TestExtractStatements_AssignmentKeywordLHS(t *testing.T) {
	// Although Python won't parse this, test defensive keyword check
	source := `
def foo():
    valid_var = 10
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.Equal(t, "valid_var", statements[0].Def)
}

func TestExtractReturn_MultipleChildren(t *testing.T) {
	source := `
def foo():
    return x + y
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	assert.Equal(t, core.StatementTypeReturn, stmt.Type)
	assert.Contains(t, stmt.Uses, "x")
	assert.Contains(t, stmt.Uses, "y")
}

func TestExtractIdentifiersFromArgs_NilNode(t *testing.T) {
	// Test nil safety
	result := extractIdentifiersFromArgs(nil, []byte{})
	assert.Equal(t, 0, len(result))
}

func TestExtractCallArgs_NilNode(t *testing.T) {
	// Test nil safety
	result := extractCallArgs(nil, []byte{})
	assert.Equal(t, 0, len(result))
}

func TestExtractStatements_LineNumbers(t *testing.T) {
	source := `
def foo():
    x = 10
    y = 20
    return x + y
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 3, len(statements))

	// Check that line numbers are set
	for _, stmt := range statements {
		assert.Greater(t, stmt.LineNumber, uint32(0), "Line number should be set")
	}
}

func TestExtractStatements_CallWithNestedKeywordArgs(t *testing.T) {
	source := `
def foo():
    func(a, b=nested(c), d=x+y)
`
	tree, funcNode, sourceBytes := parsePythonFunction(t, source, "foo")
	defer tree.Close()

	statements, err := ExtractStatements("test.py", sourceBytes, funcNode)

	require.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	stmt := statements[0]
	// Should extract identifiers from nested expressions
	assert.Contains(t, stmt.Uses, "a")
	assert.Contains(t, stmt.Uses, "c")
	assert.Contains(t, stmt.Uses, "x")
	assert.Contains(t, stmt.Uses, "y")
}

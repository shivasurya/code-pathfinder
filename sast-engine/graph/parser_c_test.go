package graph

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	clang "github.com/smacker/go-tree-sitter/c"
	cpplang "github.com/smacker/go-tree-sitter/cpp"
)

// parseCSnippetForTest parses C source for unit tests in this package.
// Lives here (rather than in graph/clike's testhelpers_test.go) because
// test-only symbols don't cross package boundaries.
func parseCSnippetForTest(t *testing.T, code string) (*sitter.Tree, *sitter.Node) {
	t.Helper()
	return parseSnippetForTest(t, code, false)
}

func parseSnippetForTest(t *testing.T, code string, isCpp bool) (*sitter.Tree, *sitter.Node) {
	t.Helper()
	parser := sitter.NewParser()
	if isCpp {
		parser.SetLanguage(cpplang.GetLanguage())
	} else {
		parser.SetLanguage(clang.GetLanguage())
	}
	defer parser.Close()
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return tree, tree.RootNode()
}

// findFirstNodeOfType performs a pre-order walk and returns the first
// descendant whose Type() matches nodeType. Returns nil when no match
// exists.
func findFirstNodeOfType(node *sitter.Node, nodeType string) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == nodeType {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if found := findFirstNodeOfType(node.Child(i), nodeType); found != nil {
			return found
		}
	}
	return nil
}

// TestParseCEndToEnd parses testdata/c/ as a complete project via Initialize
// and asserts that every node type the C parser is responsible for —
// function definitions, forward declarations, structs, enums, typedefs,
// variable declarations, includes, and call expressions — produces graph
// nodes with the right Type, Name, Language, and (where applicable)
// Metadata flags.
//
// The test is intentionally written against the public API surface
// (Initialize → CodeGraph) rather than against individual parse functions,
// so it doubles as a regression suite for the dispatch wiring in
// parser.go. If a future refactor moves parse functions into a subpackage
// or changes the dispatch order, this test will catch behavioural drift.
func TestParseCEndToEnd(t *testing.T) {
	graph := Initialize("testdata/c", nil)
	if graph == nil {
		t.Fatal("Initialize returned nil")
	}

	nodes := collectByType(graph)

	t.Run("function_definitions", func(t *testing.T) {
		fns := nodes[nodeTypeFunctionDefinition]
		// Definitions: fast, add (definition), process — 3 with bodies in
		// example.c. The forward declaration of add and the two prototypes
		// in buffer.h add three body-less function_definition nodes (every
		// function-shaped node tree-sitter sees becomes a function_definition,
		// and parser.go dispatches via fileExt before this point — so
		// declaration-shaped functions in headers also flow through here).
		if len(fns) < 3 {
			t.Fatalf("expected at least 3 function_definition nodes, got %d", len(fns))
		}

		gotByName := map[string]*Node{}
		for _, n := range fns {
			gotByName[n.Name] = n
		}

		// fast(int x) — definition, has body, has static+inline qualifiers
		fast := gotByName["fast"]
		if fast == nil {
			t.Fatal("expected function 'fast' in graph")
		}
		if fast.Language != "c" {
			t.Errorf("fast.Language = %q, want %q", fast.Language, "c")
		}
		if fast.ReturnType != "int" {
			t.Errorf("fast.ReturnType = %q, want %q", fast.ReturnType, "int")
		}
		if fast.Modifier != "static inline" {
			t.Errorf("fast.Modifier = %q, want %q", fast.Modifier, "static inline")
		}
		if got, _ := fast.Metadata[metaIsDeclaration].(bool); got {
			t.Error("fast should not be marked as declaration (has body)")
		}

		// add(int a, int b) — definition in example.c
		add := gotByName["add"]
		if add == nil {
			t.Fatal("expected function 'add' in graph")
		}
		if len(add.MethodArgumentsValue) != 2 ||
			add.MethodArgumentsValue[0] != "a" ||
			add.MethodArgumentsValue[1] != "b" {
			t.Errorf("add params = %v, want [a, b]", add.MethodArgumentsValue)
		}

		// process(struct Buffer* buf, size_t_alias n)
		process := gotByName["process"]
		if process == nil {
			t.Fatal("expected function 'process' in graph")
		}
		if process.ReturnType != "void" {
			t.Errorf("process.ReturnType = %q, want %q", process.ReturnType, "void")
		}
	})

	t.Run("forward_declaration_marked", func(t *testing.T) {
		// buffer.h declares compute() and release_all() with no body.
		// example.c has a forward decl of add(int, int) too. We expect at
		// least one function_definition with Metadata[is_declaration]=true.
		decls := 0
		for _, n := range nodes[nodeTypeFunctionDefinition] {
			if v, _ := n.Metadata[metaIsDeclaration].(bool); v {
				decls++
			}
		}
		if decls < 2 {
			t.Errorf("expected ≥2 forward declarations, got %d", decls)
		}
	})

	t.Run("struct_declaration", func(t *testing.T) {
		structs := nodes[nodeTypeStructDeclaration]
		if len(structs) == 0 {
			t.Fatal("expected at least one struct_declaration")
		}

		var buffer *Node
		for _, n := range structs {
			if n.Name == "Buffer" {
				buffer = n
				break
			}
		}
		if buffer == nil {
			t.Fatal("expected struct 'Buffer'")
		}
		// Buffer has fields: char* data, size_t_alias len, int capacity
		if len(buffer.MethodArgumentsType) != 3 {
			t.Errorf("Buffer fields = %v, want 3 entries", buffer.MethodArgumentsType)
		}
	})

	t.Run("enum_declaration", func(t *testing.T) {
		enums := nodes[nodeTypeEnumDeclaration]
		var color *Node
		for _, n := range enums {
			if n.Name == "Color" {
				color = n
				break
			}
		}
		if color == nil {
			t.Fatal("expected enum 'Color'")
		}
		enumerators, _ := color.Metadata[metaEnumerators].([]string)
		want := []string{"RED=0", "GREEN", "BLUE=5"}
		if len(enumerators) != len(want) {
			t.Fatalf("Color enumerators = %v, want %v", enumerators, want)
		}
		for i, w := range want {
			if enumerators[i] != w {
				t.Errorf("enumerator[%d] = %q, want %q", i, enumerators[i], w)
			}
		}
	})

	t.Run("type_definition_unsigned_long", func(t *testing.T) {
		typedefs := nodes[nodeTypeTypeDefinition]
		var alias *Node
		for _, n := range typedefs {
			if n.Name == "size_t_alias" {
				alias = n
				break
			}
		}
		if alias == nil {
			t.Fatal("expected typedef 'size_t_alias'")
		}
		if alias.DataType != "unsigned long" {
			t.Errorf("size_t_alias.DataType = %q, want %q", alias.DataType, "unsigned long")
		}
	})

	t.Run("type_definition_anonymous_struct", func(t *testing.T) {
		typedefs := nodes[nodeTypeTypeDefinition]
		var point *Node
		for _, n := range typedefs {
			if n.Name == "Point" {
				point = n
				break
			}
		}
		if point == nil {
			t.Fatal("expected typedef 'Point'")
		}
		// DataType is the underlying struct text; we just verify it
		// references a struct.
		if point.DataType == "" {
			t.Error("Point typedef should have non-empty DataType")
		}
	})

	t.Run("variable_declarations", func(t *testing.T) {
		vars := nodes[nodeTypeVariableDecl]
		// Globals: pi (initialised), global_buf (no init), a, b, c (3 from
		// the multi-declarator), tmp (function-local in process). We
		// expect at least 6 declared variables.
		if len(vars) < 6 {
			t.Fatalf("expected ≥6 variable declarations, got %d (%v)", len(vars), names(vars))
		}

		byName := map[string]*Node{}
		for _, n := range vars {
			byName[n.Name] = n
		}

		// pi is a global float
		pi := byName["pi"]
		if pi == nil {
			t.Fatal("expected variable 'pi'")
		}
		if pi.DataType != "const float" {
			t.Errorf("pi.DataType = %q, want %q", pi.DataType, "const float")
		}
		if pi.VariableValue != "3.14f" {
			t.Errorf("pi.VariableValue = %q, want %q", pi.VariableValue, "3.14f")
		}
		if pi.Scope != "global" {
			t.Errorf("pi.Scope = %q, want %q", pi.Scope, "global")
		}

		// global_buf is char* with no initialiser
		buf := byName["global_buf"]
		if buf == nil {
			t.Fatal("expected variable 'global_buf'")
		}
		if buf.DataType != "char*" {
			t.Errorf("global_buf.DataType = %q, want %q", buf.DataType, "char*")
		}
		if buf.VariableValue != "" {
			t.Errorf("global_buf.VariableValue = %q, want empty", buf.VariableValue)
		}

		// Multi-declarator: int a = 1, b = 2, c;
		for _, n := range []string{"a", "b", "c"} {
			v := byName[n]
			if v == nil {
				t.Errorf("expected variable %q from multi-declarator", n)
				continue
			}
			if v.DataType != "int" {
				t.Errorf("%s.DataType = %q, want %q", n, v.DataType, "int")
			}
		}

		// Function-local tmp inside process()
		tmp := byName["tmp"]
		if tmp == nil {
			t.Fatal("expected function-local variable 'tmp'")
		}
		if tmp.Scope != "process" {
			t.Errorf("tmp.Scope = %q, want %q", tmp.Scope, "process")
		}
	})

	t.Run("includes_system_vs_local", func(t *testing.T) {
		incs := nodes[nodeTypeIncludeStatement]
		byName := map[string]*Node{}
		for _, n := range incs {
			byName[n.Name] = n
		}

		// <stdio.h> and <stdlib.h> should be system; "buffer.h" local.
		stdio := byName["stdio.h"]
		if stdio == nil {
			t.Fatal("expected include 'stdio.h'")
		}
		if v, _ := stdio.Metadata[metaSystemInclude].(bool); !v {
			t.Errorf("stdio.h should be system include")
		}

		buffer := byName["buffer.h"]
		if buffer == nil {
			t.Fatal("expected include 'buffer.h'")
		}
		if v, _ := buffer.Metadata[metaSystemInclude].(bool); v {
			t.Errorf("buffer.h should NOT be system include")
		}
	})

	t.Run("call_expressions_linked_to_caller", func(t *testing.T) {
		calls := nodes[nodeTypeCallExpression]
		// process() body calls malloc, free, add. We expect at least these
		// 3 names to appear among call_expression nodes.
		want := map[string]bool{"malloc": false, "free": false, "add": false}
		for _, n := range calls {
			if _, ok := want[n.Name]; ok {
				want[n.Name] = true
			}
		}
		for name, found := range want {
			if !found {
				t.Errorf("expected call to %q in graph", name)
			}
		}

		// The call to add() inside process() should produce an edge from
		// the process function node to the add call node.
		var processFn *Node
		for _, n := range nodes[nodeTypeFunctionDefinition] {
			if n.Name == "process" {
				processFn = n
				break
			}
		}
		if processFn == nil {
			t.Fatal("expected process() function node")
		}
		hasAddEdge := false
		for _, e := range processFn.OutgoingEdges {
			if e.To.Type == nodeTypeCallExpression && e.To.Name == "add" {
				hasAddEdge = true
				break
			}
		}
		if !hasAddEdge {
			t.Error("expected outgoing edge from process() to call_expression 'add'")
		}
	})
}

// TestParseCCallExpression_MethodAndQualified covers the call-shape
// metadata branches (IsArrow, IsQualified, Receiver) that don't appear in
// the example.c integration fixture but are essential for C++ rule
// matching. Each case parses a synthetic snippet, locates the
// call_expression, and verifies the metadata flags that
// parseCCallExpression sets.
func TestParseCCallExpression_MethodAndQualified(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		isCpp     bool
		wantName  string
		wantFlags map[string]bool
	}{
		{
			name:     "arrow method call",
			code:     "void f(struct Buffer* b) { b->free(); }",
			wantName: "free",
			wantFlags: map[string]bool{
				"is_method": true,
				"is_arrow":  true,
			},
		},
		{
			name:     "qualified namespace call",
			code:     "void f() { ns::do_thing(); }",
			isCpp:    true,
			wantName: "ns::do_thing",
			wantFlags: map[string]bool{
				"is_qualified": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewCodeGraph()
			tree, root := parseSnippetForTest(t, tt.code, tt.isCpp)
			defer tree.Close()

			call := findFirstNodeOfType(root, "call_expression")
			if call == nil {
				t.Fatal("call_expression not found")
			}
			parseCCallExpression(call, []byte(tt.code), g, "test.c", nil)

			calls := collectByType(g)[nodeTypeCallExpression]
			if len(calls) != 1 {
				t.Fatalf("expected 1 call_expression node, got %d", len(calls))
			}
			got := calls[0]
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			for key, want := range tt.wantFlags {
				v, _ := got.Metadata[key].(bool)
				if v != want {
					t.Errorf("Metadata[%q] = %v, want %v", key, v, want)
				}
			}
		})
	}
}

// TestParseCLikeDeclaration_IsCppFlag confirms that the isCpp flag flips
// the Language tag on produced variable_declaration nodes. The branch is
// otherwise unreachable until parser_cpp.go (PR-04) starts dispatching
// declaration nodes from .cpp files.
func TestParseCLikeDeclaration_IsCppFlag(t *testing.T) {
	code := "int answer = 42;"
	tree, root := parseCSnippetForTest(t, code)
	defer tree.Close()

	decl := findFirstNodeOfType(root, "declaration")
	if decl == nil {
		t.Fatal("declaration not found")
	}

	g := NewCodeGraph()
	parseCLikeDeclaration(decl, []byte(code), g, "test.cpp", nil, true)

	vars := collectByType(g)[nodeTypeVariableDecl]
	if len(vars) != 1 {
		t.Fatalf("expected 1 variable_declaration, got %d", len(vars))
	}
	if vars[0].Language != "cpp" {
		t.Errorf("Language = %q, want %q", vars[0].Language, "cpp")
	}
}

// collectByType groups every node in the graph by its Type field. Useful
// for end-to-end assertions that need to enumerate one category at a time.
func collectByType(g *CodeGraph) map[string][]*Node {
	out := map[string][]*Node{}
	for _, n := range g.Nodes {
		out[n.Type] = append(out[n.Type], n)
	}
	return out
}

// names returns the Name field from every node, useful in error messages
// when assertions on a category fail.
func names(nodes []*Node) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Name)
	}
	return out
}

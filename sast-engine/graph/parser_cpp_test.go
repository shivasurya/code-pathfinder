package graph

import (
	"testing"
)

// TestParseCppEndToEnd parses ../testdata/cpp/ as a complete project via
// Initialize and asserts every C++-only node category is correctly
// populated:
//
//   - Classes with inheritance (Dog : public Animal)
//   - Pure virtual methods (Animal::speak = 0)
//   - Virtual destructors (~Animal)
//   - Override-marked methods (Dog::speak override)
//   - Class data members (Dog::age)
//   - Access specifier propagation (public/private)
//   - Out-of-line method definitions (Dog::bark())
//   - Namespaces (named and anonymous)
//   - Templates (typename T)
//   - Struct (C++ flavour, public default access)
//   - Scoped enum (enum class)
//   - Typedef
//   - throw / try-catch
//   - Method calls with arrow / dot / qualified shapes
//   - Forward declarations in headers
//
// Runs end-to-end through Initialize() so the dispatcher in parser.go is
// exercised alongside the parse functions in parser_cpp.go.
func TestParseCppEndToEnd(t *testing.T) {
	graph := Initialize("../testdata/cpp", nil)
	if graph == nil {
		t.Fatal("Initialize returned nil")
	}

	nodes := collectByType(graph)

	t.Run("class_declarations_with_inheritance", func(t *testing.T) {
		classes := nodes[nodeTypeClassDeclaration]
		byName := indexByName(classes)

		animal := byName["Animal"]
		dog := byName["Dog"]
		if animal == nil {
			t.Fatal("expected class Animal")
		}
		if dog == nil {
			t.Fatal("expected class Dog")
		}

		if dog.SuperClass != "Animal" {
			t.Errorf("Dog.SuperClass = %q, want %q", dog.SuperClass, "Animal")
		}
		inheritance, _ := dog.Metadata[metaInheritance].([]string)
		if len(inheritance) == 0 || inheritance[0] != "public Animal" {
			t.Errorf("Dog inheritance = %v, want [public Animal]", inheritance)
		}
		if animal.Language != languageCpp {
			t.Errorf("Animal.Language = %q, want %q", animal.Language, languageCpp)
		}
	})

	t.Run("namespace_propagates_to_classes", func(t *testing.T) {
		dog := indexByName(nodes[nodeTypeClassDeclaration])["Dog"]
		if dog == nil {
			t.Fatal("expected class Dog")
		}
		if dog.PackageName != "mylib" {
			t.Errorf("Dog.PackageName = %q, want %q", dog.PackageName, "mylib")
		}
	})

	t.Run("anonymous_namespace_has_no_name", func(t *testing.T) {
		anon := false
		for _, n := range nodes["namespace_definition"] {
			if n.Name == "" {
				anon = true
				break
			}
		}
		if !anon {
			t.Error("expected at least one anonymous namespace_definition")
		}
	})

	t.Run("method_declarations_with_access_and_override", func(t *testing.T) {
		methods := nodes[nodeTypeMethodDeclaration]
		var speakAnimal, speakDog, bark *Node
		for _, m := range methods {
			switch {
			case m.Name == "speak" && hasMetadataBool(m, metaIsPureVirtual):
				speakAnimal = m
			case m.Name == "speak" && hasMetadataBool(m, metaIsOverride):
				speakDog = m
			case m.Name == "bark":
				bark = m
			}
		}

		if speakAnimal == nil {
			t.Fatal("expected pure virtual speak in Animal")
		}
		if !hasMetadataBool(speakAnimal, metaIsPureVirtual) {
			t.Error("Animal.speak should be marked pure virtual")
		}
		if !hasMetadataBool(speakAnimal, metaIsVirtual) {
			t.Error("Animal.speak should be marked virtual")
		}
		if speakAnimal.Modifier != "public" {
			t.Errorf("Animal.speak Modifier = %q, want %q", speakAnimal.Modifier, "public")
		}

		if speakDog == nil {
			t.Fatal("expected Dog.speak override")
		}
		if !hasMetadataBool(speakDog, metaIsOverride) {
			t.Error("Dog.speak should be marked override")
		}

		if bark == nil {
			t.Fatal("expected private method bark")
		}
		if bark.Modifier != "private" {
			t.Errorf("Dog.bark Modifier = %q, want %q", bark.Modifier, "private")
		}
	})

	t.Run("destructor_recognised_as_method", func(t *testing.T) {
		methods := nodes[nodeTypeMethodDeclaration]
		found := false
		for _, m := range methods {
			if m.Name == "~Animal" {
				found = true
				if !hasMetadataBool(m, metaIsDestructor) {
					t.Error("destructor should carry is_destructor metadata")
				}
				break
			}
		}
		if !found {
			t.Error("expected destructor ~Animal among method_declaration nodes")
		}
	})

	t.Run("class_field_declaration", func(t *testing.T) {
		fields := nodes[nodeTypeFieldDecl]
		var age *Node
		for _, f := range fields {
			if f.Name == "age" {
				age = f
				break
			}
		}
		if age == nil {
			t.Fatal("expected field 'age' in Dog")
		}
		if age.DataType != "int" {
			t.Errorf("age.DataType = %q, want %q", age.DataType, "int")
		}
		if age.Modifier != "public" {
			t.Errorf("age.Modifier = %q, want %q", age.Modifier, "public")
		}
	})

	t.Run("template_parameters_recorded", func(t *testing.T) {
		templates := nodes["template_declaration"]
		if len(templates) == 0 {
			t.Fatal("expected at least one template_declaration node")
		}
		params, _ := templates[0].Metadata[metaTemplateParams].([]string)
		if len(params) == 0 || params[0] != "T" {
			t.Errorf("template params = %v, want [T]", params)
		}
	})

	t.Run("throw_statement", func(t *testing.T) {
		throws := nodes[nodeTypeThrowStmt]
		if len(throws) == 0 {
			t.Fatal("expected ThrowStmt node")
		}
		expr, _ := throws[0].Metadata[metaThrowExpr].(string)
		if expr == "" {
			t.Errorf("throw_expression should not be empty")
		}
	})

	t.Run("try_statement_with_catch_types", func(t *testing.T) {
		tries := nodes[nodeTypeTryStmt]
		if len(tries) == 0 {
			t.Fatal("expected TryStmt node")
		}
		catches, _ := tries[0].Metadata[metaCatchClauses].([]string)
		if len(catches) == 0 || catches[0] == "" {
			t.Errorf("expected at least one catch clause type, got %v", catches)
		}
	})

	t.Run("call_expressions_with_shapes", func(t *testing.T) {
		calls := nodes[nodeTypeCallExpression]
		hasArrow, hasMethodDot, hasQualified := false, false, false
		for _, c := range calls {
			if hasMetadataBool(c, metaIsArrow) {
				hasArrow = true
			}
			if hasMetadataBool(c, metaIsMethod) && !hasMetadataBool(c, metaIsArrow) {
				hasMethodDot = true
			}
			if hasMetadataBool(c, metaIsQualified) {
				hasQualified = true
			}
		}
		// example.cpp uses d.speak() (dot) and identity<int>(42) (qualified-like)
		// but no arrow call. We assert the shapes that appear.
		if !hasMethodDot {
			t.Error("expected at least one dot method call")
		}
		if !hasQualified {
			t.Error("expected at least one qualified call (e.g., identity<int>())")
		}
		_ = hasArrow
	})

	t.Run("scoped_enum_marked", func(t *testing.T) {
		enums := nodes[nodeTypeEnumDeclaration]
		var color *Node
		for _, e := range enums {
			if e.Name == "Color" {
				color = e
				break
			}
		}
		if color == nil {
			t.Fatal("expected enum Color")
		}
		if v, _ := color.Metadata["is_scoped"].(bool); !v {
			t.Error("Color should be marked is_scoped (declared as enum class)")
		}
	})

	t.Run("typedef_recorded_with_cpp_language", func(t *testing.T) {
		typedefs := nodes[nodeTypeTypeDefinition]
		var alias *Node
		for _, td := range typedefs {
			if td.Name == "size_alias" {
				alias = td
				break
			}
		}
		if alias == nil {
			t.Fatal("expected typedef size_alias")
		}
		if alias.Language != languageCpp {
			t.Errorf("size_alias.Language = %q, want %q", alias.Language, languageCpp)
		}
		if alias.DataType != "unsigned long" {
			t.Errorf("size_alias.DataType = %q, want %q", alias.DataType, "unsigned long")
		}
	})

	t.Run("struct_with_cpp_language", func(t *testing.T) {
		structs := nodes[nodeTypeStructDeclaration]
		var point *Node
		for _, s := range structs {
			if s.Name == "Point" {
				point = s
				break
			}
		}
		if point == nil {
			t.Fatal("expected struct Point")
		}
		if point.Language != languageCpp {
			t.Errorf("Point.Language = %q, want %q", point.Language, languageCpp)
		}
		if len(point.MethodArgumentsType) != 2 {
			t.Errorf("Point fields = %v, want 2 entries", point.MethodArgumentsType)
		}
	})

	t.Run("forward_declarations_in_header", func(t *testing.T) {
		// buffer.hpp declares Buffer class with constructor, destructor,
		// and append() method as inline declarations. They should appear
		// as method_declaration nodes with is_declaration=true.
		methods := nodes[nodeTypeMethodDeclaration]
		seen := map[string]bool{}
		for _, m := range methods {
			if m.File == "../testdata/cpp/buffer.hpp" {
				seen[m.Name] = true
			}
		}
		for _, want := range []string{"Buffer", "~Buffer", "append"} {
			if !seen[want] {
				t.Errorf("expected %q method declaration in buffer.hpp; saw %v", want, seen)
			}
		}
	})

	t.Run("regression_no_java_tagged_nodes_in_cpp_files", func(t *testing.T) {
		for _, n := range graph.Nodes {
			if n.File == "../testdata/cpp/example.cpp" || n.File == "../testdata/cpp/buffer.hpp" {
				if n.Language != languageCpp {
					t.Errorf("node %q (%s) in C++ file has Language=%q, want %q",
						n.Name, n.Type, n.Language, languageCpp)
				}
			}
		}
	})
}

// TestParseCppClassSpecifier_ForwardDeclaration verifies the no-body
// short-circuit so a forward `class Foo;` does not produce a phantom
// class_declaration node.
func TestParseCppClassSpecifier_ForwardDeclaration(t *testing.T) {
	code := "class Foo;"
	tree, root := parseSnippetForTest(t, code, true)
	defer tree.Close()

	cls := findFirstNodeOfType(root, "class_specifier")
	if cls == nil {
		t.Fatal("class_specifier not found")
	}
	g := NewCodeGraph()
	if got := parseCppClassSpecifier(cls, []byte(code), g, "f.cpp", nil); got != nil {
		t.Errorf("forward declaration should return nil, got %+v", got)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("forward declaration should not add nodes, got %d", len(g.Nodes))
	}
}

// TestExtractCatchExceptionTypes_CatchAll covers the `catch (...)` shape
// where tree-sitter emits the parameter list as the literal "(...)" with
// no parameter_declaration child — extractCatchExceptionTypes should
// emit "..." for that handler.
func TestExtractCatchExceptionTypes_CatchAll(t *testing.T) {
	code := `void f() { try { dangerous(); } catch (...) { recover(); } }`
	tree, root := parseSnippetForTest(t, code, true)
	defer tree.Close()

	try := findFirstNodeOfType(root, "try_statement")
	if try == nil {
		t.Fatal("try_statement not found")
	}
	got := extractCatchExceptionTypes(try, []byte(code))
	if len(got) != 1 || got[0] != "..." {
		t.Errorf("got %v, want [...]", got)
	}
}

// TestExtractTemplateParameters_Nil verifies the nil-list guard.
func TestExtractTemplateParameters_Nil(t *testing.T) {
	if got := extractTemplateParameters(nil, nil); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// TestRecordAccessSpecifier_OutsideClassContext is a no-op — the access
// specifier flows only when currentContext is a class node. This covers
// the early-return path when recordAccessSpecifier is invoked with no
// surrounding class (e.g., bug or future struct support that calls it
// with a non-class context).
func TestRecordAccessSpecifier_OutsideClassContext(t *testing.T) {
	code := `class C { public: };`
	tree, root := parseSnippetForTest(t, code, true)
	defer tree.Close()

	access := findFirstNodeOfType(root, "access_specifier")
	if access == nil {
		t.Fatal("access_specifier not found")
	}

	// Pass a non-class context (e.g., a function node) — should be a no-op.
	notAClass := &Node{Type: nodeTypeFunctionDefinition, Metadata: map[string]any{}}
	recordAccessSpecifier(access, []byte(code), notAClass)
	if _, set := notAClass.Metadata[metaCurrentAccess]; set {
		t.Error("recordAccessSpecifier should not mutate non-class context")
	}
}

// indexByName returns a map of Name → Node for fast lookup. Names that
// repeat across files keep whichever one comes first; tests that need
// disambiguation should walk the slice directly instead.
func indexByName(nodes []*Node) map[string]*Node {
	out := make(map[string]*Node, len(nodes))
	for _, n := range nodes {
		if _, ok := out[n.Name]; !ok {
			out[n.Name] = n
		}
	}
	return out
}

// hasMetadataBool returns the bool value stored under key, defaulting to
// false when absent or wrong type. Cleans up assertion code.
func hasMetadataBool(n *Node, key string) bool {
	if n == nil || n.Metadata == nil {
		return false
	}
	v, _ := n.Metadata[key].(bool)
	return v
}

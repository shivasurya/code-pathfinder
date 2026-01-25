package resolution

import (
	"context"
	"fmt"
	"os"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	python "github.com/smacker/go-tree-sitter/python"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// TestRealWorldInference demonstrates type inference on realistic Python code.
func TestRealWorldInference(t *testing.T) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	// Set up type inference
	inferencer := NewBidirectionalInferencer(nil, nil, nil, 1000)
	store := NewTypeStore()

	// Simulate known type: app is Application
	store.Set("app", core.NewConcreteType("Application", 0.95),
		core.ConfidenceAssignment, "test.py", 0, 0)

	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("           REAL-WORLD TYPE INFERENCE DEMO")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Test Case 1: Attribute access (app.user_service)
	t.Run("AttributeAccess", func(t *testing.T) {
		code := []byte("app.user_service")
		tree, _ := parser.ParseCtx(context.Background(), nil, code)
		node := tree.RootNode().Child(0).Child(0)

		typ, conf := inferencer.InferType(node, store, code, "test.py", nil, "", "")

		fmt.Printf("Test 1: Attribute Access\n")
		fmt.Printf("  Code:       app.user_service\n")
		fmt.Printf("  Result:     %s\n", typ.FQN())
		fmt.Printf("  Confidence: %.2f\n", conf)
		fmt.Printf("  Status:     %s\n\n", getStatus(typ))
	})

	// Test Case 2: Method call (service.get_user(123))
	t.Run("MethodCall", func(t *testing.T) {
		code := []byte("service.get_user(123)")
		tree, _ := parser.ParseCtx(context.Background(), nil, code)
		node := tree.RootNode().Child(0).Child(0)

		store.Set("service", core.NewConcreteType("UserService", 0.95),
			core.ConfidenceAssignment, "test.py", 0, 0)

		typ, conf := inferencer.InferType(node, store, code, "test.py", nil, "", "")

		fmt.Printf("Test 2: Method Call\n")
		fmt.Printf("  Code:       service.get_user(123)\n")
		fmt.Printf("  Result:     %s\n", typ.FQN())
		fmt.Printf("  Confidence: %.2f\n", conf)
		fmt.Printf("  Status:     %s\n\n", getStatus(typ))
	})

	// Test Case 3: Chained call (app.user_service.get_user())
	t.Run("ChainedCall", func(t *testing.T) {
		code := []byte("app.user_service.get_user()")
		tree, _ := parser.ParseCtx(context.Background(), nil, code)
		node := tree.RootNode().Child(0).Child(0)

		typ, conf := inferencer.InferType(node, store, code, "test.py", nil, "", "")

		fmt.Printf("Test 3: Chained Call\n")
		fmt.Printf("  Code:       app.user_service.get_user()\n")
		fmt.Printf("  Result:     %s\n", typ.FQN())
		fmt.Printf("  Confidence: %.2f\n", conf)
		fmt.Printf("  Status:     %s\n\n", getStatus(typ))
	})

	// Test Case 4: Self reference (self.database)
	t.Run("SelfReference", func(t *testing.T) {
		code := []byte("self.database")
		tree, _ := parser.ParseCtx(context.Background(), nil, code)
		node := tree.RootNode().Child(0).Child(0)

		selfType := core.NewConcreteType("UserService", 0.95)

		typ, conf := inferencer.InferType(node, store, code, "test.py", selfType, "UserService", "")

		fmt.Printf("Test 4: Self Reference\n")
		fmt.Printf("  Code:       self.database\n")
		fmt.Printf("  Context:    Inside UserService class\n")
		fmt.Printf("  Result:     %s\n", typ.FQN())
		fmt.Printf("  Confidence: %.2f\n", conf)
		fmt.Printf("  Status:     %s\n\n", getStatus(typ))
	})

	// Test Case 5: Literal inference
	t.Run("Literals", func(t *testing.T) {
		literals := []struct {
			code     string
			expected string
		}{
			{`"hello"`, "builtins.str"},
			{`123`, "builtins.int"},
			{`[1, 2, 3]`, "builtins.list"},
			{`{"key": "value"}`, "builtins.dict"},
			{`True`, "builtins.bool"},
		}

		fmt.Printf("Test 5: Literal Inference\n")
		for _, lit := range literals {
			code := []byte(lit.code)
			tree, _ := parser.ParseCtx(context.Background(), nil, code)
			node := tree.RootNode().Child(0).Child(0)

			typ, conf := inferencer.InferType(node, NewTypeStore(), code, "test.py", nil, "", "")

			fmt.Printf("  Code: %-20s → Type: %-15s Conf: %.2f %s\n",
				lit.code, typ.FQN(), conf, getStatus(typ))
		}
		fmt.Println()
	})

	fmt.Println("═══════════════════════════════════════════════════════════")
}

// TestComplexRealWorldScenario tests a more complex real-world scenario.
func TestComplexRealWorldScenario(t *testing.T) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	inferencer := NewBidirectionalInferencer(nil, nil, nil, 1000)
	store := NewTypeStore()

	// Known types
	store.Set("app", core.NewConcreteType("myapp.Application", 0.95),
		core.ConfidenceAssignment, "test.py", 0, 0)

	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("        COMPLEX REAL-WORLD SCENARIO ANALYSIS")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Analyze each assignment in the function
	assignments := []struct {
		line     int
		code     string
		variable string
	}{
		{3, "app.user_service", "service"},
		{6, "service.get_user(user_id)", "user"},
		{9, "user.get_display_name()", "display_name"},
		{12, `"User: " + display_name`, "message"},
	}

	for _, assign := range assignments {
		code := []byte(assign.code)
		tree, _ := parser.ParseCtx(context.Background(), nil, code)
		node := tree.RootNode().Child(0).Child(0)

		typ, conf := inferencer.InferType(node, store, code, "test.py", nil, "", "")

		fmt.Printf("Line %d: %s = %s\n", assign.line, assign.variable, assign.code)
		fmt.Printf("  → Inferred Type: %s (confidence: %.2f)\n", typ.FQN(), conf)
		fmt.Printf("  → Status: %s\n\n", getStatus(typ))

		// Update store for next iteration
		if !core.IsAnyType(typ) {
			store.Set(assign.variable, typ, core.ConfidenceAssignment, "test.py", assign.line, 0)
		}
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
}

// TestStrategySelection demonstrates type inference for different code patterns.
func TestStrategySelection(t *testing.T) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	inferencer := NewBidirectionalInferencer(nil, nil, nil, 1000)
	store := NewTypeStore()
	store.Set("obj", core.NewConcreteType("MyClass", 0.95),
		core.ConfidenceAssignment, "test.py", 0, 0)

	testCases := []struct {
		name     string
		code     string
		expected string
	}{
		{"String Literal", `"hello"`, "builtins.str"},
		{"Integer Literal", `42`, "builtins.int"},
		{"List Literal", `[1, 2, 3]`, "builtins.list"},
		{"Dict Literal", `{"key": "value"}`, "builtins.dict"},
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("          TYPE INFERENCE FOR COMMON PATTERNS")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	for _, tc := range testCases {
		code := []byte(tc.code)
		tree, _ := parser.ParseCtx(context.Background(), nil, code)
		node := tree.RootNode().Child(0).Child(0)

		typ, conf := inferencer.InferType(node, store, code, "test.py", nil, "", "")

		fmt.Printf("%-20s: %s\n", tc.name, code)
		fmt.Printf("  → Inferred Type: %s (confidence: %.2f) %s\n\n",
			typ.FQN(), conf, getStatus(typ))
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
}

func getStatus(typ core.Type) string {
	if core.IsAnyType(typ) {
		return "❌ Unknown"
	}
	return "✅ Resolved"
}

// Run this to see the demo output.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

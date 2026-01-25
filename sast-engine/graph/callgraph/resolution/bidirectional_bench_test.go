package resolution

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	python "github.com/smacker/go-tree-sitter/python"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// BenchmarkInferType_SimpleVariable benchmarks type inference for simple variable access.
func BenchmarkInferType_SimpleVariable(b *testing.B) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	source := []byte("my_var")
	tree, _ := parser.ParseCtx(context.Background(), nil, source)
	node := tree.RootNode().Child(0).Child(0)

	store := NewTypeStore()
	store.Set("my_var", core.NewConcreteType("myapp.MyClass", 0.95),
		core.ConfidenceAssignment, "test.py", 0, 0)

	inferencer := NewBidirectionalInferencer(nil, nil, nil, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inferencer.InferType(node, store, source, "test.py", nil, "", "")
	}
}

// BenchmarkInferType_MethodCall benchmarks type inference for method calls.
func BenchmarkInferType_MethodCall(b *testing.B) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	source := []byte("service.get_user()")
	tree, _ := parser.ParseCtx(context.Background(), nil, source)
	node := tree.RootNode().Child(0).Child(0)

	store := NewTypeStore()
	store.Set("service", core.NewConcreteType("myapp.UserService", 0.95),
		core.ConfidenceAssignment, "test.py", 0, 0)

	inferencer := NewBidirectionalInferencer(nil, nil, nil, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inferencer.InferType(node, store, source, "test.py", nil, "", "")
	}
}

// BenchmarkInferType_ChainedCall benchmarks type inference for chained method calls.
func BenchmarkInferType_ChainedCall(b *testing.B) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	source := []byte("app.service.get_user()")
	tree, _ := parser.ParseCtx(context.Background(), nil, source)
	node := tree.RootNode().Child(0).Child(0)

	store := NewTypeStore()
	store.Set("app", core.NewConcreteType("myapp.Application", 0.95),
		core.ConfidenceAssignment, "test.py", 0, 0)

	inferencer := NewBidirectionalInferencer(nil, nil, nil, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inferencer.InferType(node, store, source, "test.py", nil, "", "")
	}
}

// BenchmarkInferType_SelfReference benchmarks type inference for self.method() calls.
func BenchmarkInferType_SelfReference(b *testing.B) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	source := []byte("self.process()")
	tree, _ := parser.ParseCtx(context.Background(), nil, source)
	node := tree.RootNode().Child(0).Child(0)

	store := NewTypeStore()
	selfType := core.NewConcreteType("myapp.Handler", 0.95)

	inferencer := NewBidirectionalInferencer(nil, nil, nil, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inferencer.InferType(node, store, source, "test.py", selfType, "myapp.Handler", "")
	}
}

// BenchmarkTypeStore_Lookup benchmarks TypeStore variable lookups.
func BenchmarkTypeStore_Lookup(b *testing.B) {
	store := NewTypeStore()
	store.Set("var1", core.NewConcreteType("Type1", 0.9), core.ConfidenceAssignment, "test.py", 0, 0)
	store.Set("var2", core.NewConcreteType("Type2", 0.9), core.ConfidenceAssignment, "test.py", 0, 0)
	store.Set("var3", core.NewConcreteType("Type3", 0.9), core.ConfidenceAssignment, "test.py", 0, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Lookup("var2")
	}
}

// BenchmarkTypeStore_Set benchmarks TypeStore variable assignments.
func BenchmarkTypeStore_Set(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := NewTypeStore()
		store.Set("myvar", core.NewConcreteType("MyType", 0.9),
			core.ConfidenceAssignment, "test.py", 0, 0)
	}
}

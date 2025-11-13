package callgraph

import (
	"testing"
)

// TestLanguageAnalyzer_Interface verifies the interface is well-defined.
func TestLanguageAnalyzer_Interface(t *testing.T) {
	// Compile-time check that mockAnalyzer implements LanguageAnalyzer
	var _ LanguageAnalyzer = (*mockAnalyzer)(nil)
}

// mockAnalyzer is a test implementation of LanguageAnalyzer.
type mockAnalyzer struct{}

func (m *mockAnalyzer) Name() string { return "mock" }

func (m *mockAnalyzer) FileExtensions() []string { return []string{".mock"} }

func (m *mockAnalyzer) Parse(filePath string, source []byte) (*ParsedModule, error) {
	return &ParsedModule{FilePath: filePath, Language: "mock"}, nil
}

func (m *mockAnalyzer) ExtractImports(module *ParsedModule) (*ImportMap, error) {
	return &ImportMap{}, nil
}

func (m *mockAnalyzer) ExtractFunctions(module *ParsedModule) ([]*FunctionDef, error) {
	return []*FunctionDef{}, nil
}

func (m *mockAnalyzer) ExtractClasses(module *ParsedModule) ([]*ClassDef, error) {
	return []*ClassDef{}, nil
}

func (m *mockAnalyzer) InferTypes(module *ParsedModule, registry *ModuleRegistry) (*TypeContext, error) {
	return &TypeContext{}, nil
}

func (m *mockAnalyzer) ExtractCallSites(fn *FunctionDef) ([]*CallSite, error) {
	return []*CallSite{}, nil
}

func (m *mockAnalyzer) ExtractStatements(fn *FunctionDef) ([]*Statement, error) {
	return []*Statement{}, nil
}

func (m *mockAnalyzer) ExtractVariables(fn *FunctionDef) ([]*Variable, error) {
	return []*Variable{}, nil
}

func (m *mockAnalyzer) AnalyzeTaint(fn *FunctionDef, cfg *CFG) (*TaintSummary, error) {
	return &TaintSummary{}, nil
}

func (m *mockAnalyzer) ResolveType(expr string, context *TypeContext) (*TypeInfo, error) {
	return &TypeInfo{}, nil
}

func (m *mockAnalyzer) SupportsFramework(name string) bool {
	return name == "mock-framework"
}

// TestLanguageRegistry_Registration verifies registration behavior.
func TestLanguageRegistry_Registration(t *testing.T) {
	registry := NewLanguageRegistry()
	analyzer := &mockAnalyzer{}

	registry.Register(analyzer)

	// Verify retrieval by extension
	got, ok := registry.GetByExtension(".mock")
	if !ok {
		t.Fatal("expected analyzer to be registered by extension")
	}
	if got.Name() != "mock" {
		t.Errorf("expected name 'mock', got '%s'", got.Name())
	}

	// Verify retrieval by name
	got, ok = registry.GetByName("mock")
	if !ok {
		t.Fatal("expected analyzer to be registered by name")
	}
	if got.Name() != "mock" {
		t.Errorf("expected name 'mock', got '%s'", got.Name())
	}
}

// TestLanguageRegistry_GetByExtension_NotFound verifies error handling.
func TestLanguageRegistry_GetByExtension_NotFound(t *testing.T) {
	registry := NewLanguageRegistry()

	_, ok := registry.GetByExtension(".unknown")
	if ok {
		t.Error("expected false for unknown extension")
	}
}

// TestLanguageRegistry_GetByName_NotFound verifies error handling.
func TestLanguageRegistry_GetByName_NotFound(t *testing.T) {
	registry := NewLanguageRegistry()

	_, ok := registry.GetByName("unknown")
	if ok {
		t.Error("expected false for unknown language")
	}
}

// TestLanguageRegistry_MultipleExtensions verifies multi-extension registration.
func TestLanguageRegistry_MultipleExtensions(t *testing.T) {
	registry := NewLanguageRegistry()

	// Mock analyzer with multiple extensions
	analyzer := &multiExtAnalyzer{}
	registry.Register(analyzer)

	// Verify all extensions are registered
	for _, ext := range []string{".ext1", ".ext2", ".ext3"} {
		got, ok := registry.GetByExtension(ext)
		if !ok {
			t.Errorf("expected analyzer for extension %s", ext)
		}
		if got.Name() != "multi-ext" {
			t.Errorf("expected name 'multi-ext', got '%s'", got.Name())
		}
	}
}

type multiExtAnalyzer struct{ mockAnalyzer }

func (m *multiExtAnalyzer) Name() string { return "multi-ext" }

func (m *multiExtAnalyzer) FileExtensions() []string {
	return []string{".ext1", ".ext2", ".ext3"}
}

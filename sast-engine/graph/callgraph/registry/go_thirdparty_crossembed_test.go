package registry

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// mockStdlibLoaderForEmbed implements core.GoStdlibLoader for embed resolution tests.
type mockStdlibLoaderForEmbed struct {
	types map[string]*core.GoStdlibType // key: "importPath.TypeName"
}

func (m *mockStdlibLoaderForEmbed) ValidateStdlibImport(importPath string) bool {
	for k := range m.types {
		// key format: "importPath.TypeName"
		if len(k) > len(importPath) && k[:len(importPath)] == importPath {
			return true
		}
	}
	return false
}

func (m *mockStdlibLoaderForEmbed) GetFunction(_, _ string) (*core.GoStdlibFunction, error) {
	return nil, nil //nolint:nilnil
}

func (m *mockStdlibLoaderForEmbed) GetType(importPath, typeName string) (*core.GoStdlibType, error) {
	key := importPath + "." + typeName
	t, ok := m.types[key]
	if !ok {
		return nil, nil //nolint:nilnil
	}
	return t, nil
}

func (m *mockStdlibLoaderForEmbed) PackageCount() int { return len(m.types) }

// buildLoaderWithRegistry creates a GoThirdPartyLocalLoader whose registry
// field points at a GoModuleRegistry with the given StdlibLoader attached.
func buildLoaderWithRegistry(stdlibLoader core.GoStdlibLoader) *GoThirdPartyLocalLoader {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = stdlibLoader
	return &GoThirdPartyLocalLoader{
		registry: reg,
	}
}

// makePkgWithEmbedType constructs a minimal GoStdlibPackage with one type that
// embeds the given cross-package interface name (e.g. "context.Context").
func makePkgWithEmbedType(typeName, embedName string) *core.GoStdlibPackage {
	pkg := core.NewGoStdlibPackage("github.com/example/mypkg", "")
	pkg.Types[typeName] = &core.GoStdlibType{
		Name:    typeName,
		Kind:    "interface",
		Methods: map[string]*core.GoStdlibFunction{},
		Embeds:  []string{embedName},
	}
	return pkg
}

// ---------------------------------------------------------------------------
// TestResolveEmbeddings_ViaStdlibLoader
// ---------------------------------------------------------------------------

// TestResolveEmbeddings_ViaStdlibLoader verifies that resolveEmbeddings copies
// methods from a StdlibLoader-provided type when the embed is cross-package.
func TestResolveEmbeddings_ViaStdlibLoader(t *testing.T) {
	stdlibLoader := &mockStdlibLoaderForEmbed{
		types: map[string]*core.GoStdlibType{
			"context.Context": {
				Name: "Context",
				Methods: map[string]*core.GoStdlibFunction{
					"Deadline": {Name: "Deadline"},
					"Done":     {Name: "Done"},
					"Err":      {Name: "Err"},
					"Value":    {Name: "Value"},
				},
			},
		},
	}

	loader := buildLoaderWithRegistry(stdlibLoader)
	pkg := makePkgWithEmbedType("CancelableClient", "context.Context")

	loader.resolveEmbeddings(pkg)

	typ := pkg.Types["CancelableClient"]
	assert.Contains(t, typ.Methods, "Deadline", "Deadline should be copied from context.Context via StdlibLoader")
	assert.Contains(t, typ.Methods, "Done")
	assert.Contains(t, typ.Methods, "Err")
	assert.Contains(t, typ.Methods, "Value")
}

// ---------------------------------------------------------------------------
// TestResolveEmbeddings_FallbackToWellKnown
// ---------------------------------------------------------------------------

// TestResolveEmbeddings_FallbackToWellKnown verifies that when StdlibLoader is
// nil, resolveEmbeddings still resolves io.Closer via the well-known table.
func TestResolveEmbeddings_FallbackToWellKnown(t *testing.T) {
	loader := &GoThirdPartyLocalLoader{
		registry: nil, // no registry → StdlibLoader unavailable
	}

	pkg := makePkgWithEmbedType("Resource", "io.Closer")

	loader.resolveEmbeddings(pkg)

	typ := pkg.Types["Resource"]
	assert.Contains(t, typ.Methods, "Close", "Close should resolve via well-known table even without StdlibLoader")
}

// ---------------------------------------------------------------------------
// TestResolveEmbeddings_NilRegistryStdlibLoader
// ---------------------------------------------------------------------------

// TestResolveEmbeddings_NilRegistryStdlibLoader ensures no panic when registry
// is non-nil but StdlibLoader is nil; should fall back to well-known table.
func TestResolveEmbeddings_NilRegistryStdlibLoader(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = nil // explicitly nil
	loader := &GoThirdPartyLocalLoader{registry: reg}

	pkg := makePkgWithEmbedType("Resource", "io.Closer")

	assert.NotPanics(t, func() {
		loader.resolveEmbeddings(pkg)
	})

	typ := pkg.Types["Resource"]
	assert.Contains(t, typ.Methods, "Close", "well-known fallback should fire when StdlibLoader is nil")
}

// ---------------------------------------------------------------------------
// TestResolveEmbeddings_DoesNotOverwriteExistingMethods
// ---------------------------------------------------------------------------

// TestResolveEmbeddings_DoesNotOverwriteExistingMethods ensures that methods
// already present on the type are not replaced by embedded versions.
func TestResolveEmbeddings_DoesNotOverwriteExistingMethods(t *testing.T) {
	customClose := &core.GoStdlibFunction{Name: "Close", Confidence: 0.5} //nolint:mnd

	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = &mockStdlibLoaderForEmbed{
		types: map[string]*core.GoStdlibType{
			"io.Closer": {
				Name: "Closer",
				Methods: map[string]*core.GoStdlibFunction{
					"Close": {Name: "Close", Confidence: 1.0},
				},
			},
		},
	}
	loader := &GoThirdPartyLocalLoader{registry: reg}

	pkg := core.NewGoStdlibPackage("github.com/example/mypkg", "")
	pkg.Types["Resource"] = &core.GoStdlibType{
		Name: "Resource",
		Kind: "struct",
		Methods: map[string]*core.GoStdlibFunction{
			"Close": customClose, // already present
		},
		Embeds: []string{"io.Closer"},
	}

	loader.resolveEmbeddings(pkg)

	// The custom Close (Confidence 0.5) must NOT be replaced by the stdlib one (1.0).
	assert.InDelta(t, 0.5, pkg.Types["Resource"].Methods["Close"].Confidence, 0.001)
}

// ---------------------------------------------------------------------------
// TestResolveEmbeddings_SamePackageEmbedSkipped
// ---------------------------------------------------------------------------

// TestResolveEmbeddings_SamePackageEmbedSkipped verifies that same-package embeds
// (no dot in name) are ignored by resolveEmbeddings (they're handled earlier by
// flattenEmbeddedMethods).
func TestResolveEmbeddings_SamePackageEmbedSkipped(t *testing.T) {
	loader := &GoThirdPartyLocalLoader{registry: nil}

	pkg := core.NewGoStdlibPackage("github.com/example/mypkg", "")
	pkg.Types["Client"] = &core.GoStdlibType{
		Name:    "Client",
		Kind:    "interface",
		Methods: map[string]*core.GoStdlibFunction{},
		Embeds:  []string{"EnqueueClient"}, // no dot → same package
	}

	// Must not panic even when EnqueueClient type is absent from pkg.Types.
	assert.NotPanics(t, func() {
		loader.resolveEmbeddings(pkg)
	})

	// No methods should have been added (same-package embed, no fallback available).
	assert.Empty(t, pkg.Types["Client"].Methods)
}

package registry

import (
	"errors"
	"fmt"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// mockThirdPartyLoader — lightweight GoThirdPartyLoader stub for unit tests.
//
// The mock is populated with a packages map. ValidateImport returns true when
// the import path is present. GetType and GetFunction behave as follows:
//   - Package missing   → non-nil error  (simulates "package not found")
//   - Type/func missing → depends on missingReturnsError flag:
//     false (default)   → (nil, nil)    — authoritative miss
//     true              → (nil, error)  — transient failure
// ---------------------------------------------------------------------------

type mockThirdPartyLoader struct {
	packages           map[string]*core.GoStdlibPackage
	missingReturnsError bool // when true, missing symbol returns error instead of (nil, nil)
}

func newMock(pkgs map[string]*core.GoStdlibPackage) *mockThirdPartyLoader {
	return &mockThirdPartyLoader{packages: pkgs}
}

func newMockTransient(pkgs map[string]*core.GoStdlibPackage) *mockThirdPartyLoader {
	return &mockThirdPartyLoader{packages: pkgs, missingReturnsError: true}
}

func (m *mockThirdPartyLoader) ValidateImport(importPath string) bool {
	_, ok := m.packages[importPath]
	return ok
}

func (m *mockThirdPartyLoader) GetType(importPath, typeName string) (*core.GoStdlibType, error) {
	pkg, ok := m.packages[importPath]
	if !ok {
		return nil, fmt.Errorf("package %q not found", importPath)
	}
	t, ok := pkg.Types[typeName]
	if !ok {
		if m.missingReturnsError {
			return nil, fmt.Errorf("type %s not found (transient)", typeName)
		}
		return nil, nil //nolint:nilnil // authoritative miss
	}
	return t, nil
}

func (m *mockThirdPartyLoader) GetFunction(importPath, funcName string) (*core.GoStdlibFunction, error) {
	pkg, ok := m.packages[importPath]
	if !ok {
		return nil, fmt.Errorf("package %q not found", importPath)
	}
	fn, ok := pkg.Functions[funcName]
	if !ok {
		if m.missingReturnsError {
			return nil, fmt.Errorf("function %s not found (transient)", funcName)
		}
		return nil, nil //nolint:nilnil // authoritative miss
	}
	return fn, nil
}

func (m *mockThirdPartyLoader) PackageCount() int {
	return len(m.packages)
}

// ---------------------------------------------------------------------------
// Test data helpers
// ---------------------------------------------------------------------------

func gormPackage() *core.GoStdlibPackage {
	return &core.GoStdlibPackage{
		ImportPath: "gorm.io/gorm",
		Types: map[string]*core.GoStdlibType{
			"DB": {Name: "DB", Kind: "struct", Methods: map[string]*core.GoStdlibFunction{
				"Raw": {Name: "Raw", Confidence: 1.0},
			}},
		},
		Functions:  map[string]*core.GoStdlibFunction{},
		Constants:  map[string]*core.GoStdlibConstant{},
		Variables:  map[string]*core.GoStdlibVariable{},
	}
}

func ginPackage() *core.GoStdlibPackage {
	return &core.GoStdlibPackage{
		ImportPath: "github.com/gin-gonic/gin",
		Types: map[string]*core.GoStdlibType{
			"Context": {Name: "Context", Kind: "struct", Methods: map[string]*core.GoStdlibFunction{
				"Query": {Name: "Query", Confidence: 1.0},
			}},
		},
		Functions:  map[string]*core.GoStdlibFunction{"Default": {Name: "Default", Confidence: 1.0}},
		Constants:  map[string]*core.GoStdlibConstant{},
		Variables:  map[string]*core.GoStdlibVariable{},
	}
}

func internalPackage() *core.GoStdlibPackage {
	return &core.GoStdlibPackage{
		ImportPath: "internal.company.com/mylib",
		Types: map[string]*core.GoStdlibType{
			"Client": {Name: "Client", Kind: "struct"},
		},
		Functions:  map[string]*core.GoStdlibFunction{},
		Constants:  map[string]*core.GoStdlibConstant{},
		Variables:  map[string]*core.GoStdlibVariable{},
	}
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewGoThirdPartyCombinedLoader(t *testing.T) {
	cdn := newMock(nil)
	local := newMock(nil)
	c := NewGoThirdPartyCombinedLoader(cdn, local)
	assert.NotNil(t, c)
	assert.Equal(t, cdn, c.cdnLoader)
	assert.Equal(t, local, c.localLoader)
}

func TestNewGoThirdPartyCombinedLoader_BothNil(t *testing.T) {
	c := NewGoThirdPartyCombinedLoader(nil, nil)
	assert.NotNil(t, c)
	assert.Nil(t, c.cdnLoader)
	assert.Nil(t, c.localLoader)
}

func TestGoThirdPartyCombinedLoader_ImplementsInterface(t *testing.T) {
	// Compile-time interface compliance.
	var _ core.GoThirdPartyLoader = (*GoThirdPartyCombinedLoader)(nil)
	t.Log("GoThirdPartyCombinedLoader correctly implements core.GoThirdPartyLoader")
}

// ---------------------------------------------------------------------------
// ValidateImport
// ---------------------------------------------------------------------------

func TestCombined_ValidateImport_CDN(t *testing.T) {
	cdn := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})
	local := newMock(nil)
	c := NewGoThirdPartyCombinedLoader(cdn, local)
	assert.True(t, c.ValidateImport("gorm.io/gorm"))
}

func TestCombined_ValidateImport_Local(t *testing.T) {
	cdn := newMock(nil)
	local := newMock(map[string]*core.GoStdlibPackage{"internal.company.com/mylib": internalPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, local)
	assert.True(t, c.ValidateImport("internal.company.com/mylib"))
}

func TestCombined_ValidateImport_Neither(t *testing.T) {
	cdn := newMock(nil)
	local := newMock(nil)
	c := NewGoThirdPartyCombinedLoader(cdn, local)
	assert.False(t, c.ValidateImport("unknown.io/pkg"))
}

func TestCombined_ValidateImport_CDNNil(t *testing.T) {
	local := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})
	c := NewGoThirdPartyCombinedLoader(nil, local)
	assert.True(t, c.ValidateImport("gorm.io/gorm"))
}

func TestCombined_ValidateImport_LocalNil(t *testing.T) {
	cdn := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, nil)
	assert.True(t, c.ValidateImport("gorm.io/gorm"))
}

func TestCombined_ValidateImport_BothNil(t *testing.T) {
	c := NewGoThirdPartyCombinedLoader(nil, nil)
	assert.False(t, c.ValidateImport("gorm.io/gorm"))
}

// ---------------------------------------------------------------------------
// GetType
// ---------------------------------------------------------------------------

func TestCombined_GetType_CDNHit(t *testing.T) {
	cdn := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})
	local := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, local)

	typ, err := c.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	require.NotNil(t, typ)
	assert.Equal(t, "DB", typ.Name)
}

func TestCombined_GetType_CDNMiss_LocalHit(t *testing.T) {
	cdn := newMock(nil) // CDN doesn't know the package
	local := newMock(map[string]*core.GoStdlibPackage{"internal.company.com/mylib": internalPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, local)

	typ, err := c.GetType("internal.company.com/mylib", "Client")
	require.NoError(t, err)
	require.NotNil(t, typ)
	assert.Equal(t, "Client", typ.Name)
}

func TestCombined_GetType_BothMiss(t *testing.T) {
	// Both loaders are non-nil but neither knows the package.
	// CDN skips (ValidateImport false), local returns its own "not found" error.
	cdn := newMock(nil)
	local := newMock(nil)
	c := NewGoThirdPartyCombinedLoader(cdn, local)

	_, err := c.GetType("unknown.io/pkg", "Foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown.io/pkg")
}

func TestCombined_GetType_CDNOnly(t *testing.T) {
	cdn := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, nil)

	typ, err := c.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	assert.Equal(t, "DB", typ.Name)
}

func TestCombined_GetType_LocalOnly(t *testing.T) {
	local := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})
	c := NewGoThirdPartyCombinedLoader(nil, local)

	typ, err := c.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	assert.Equal(t, "DB", typ.Name)
}

func TestCombined_GetType_BothNil(t *testing.T) {
	c := NewGoThirdPartyCombinedLoader(nil, nil)

	_, err := c.GetType("gorm.io/gorm", "DB")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in any loader")
}

// CDN has the package but not the requested type: authoritative miss (nil, nil).
// Must NOT fall back to local, even when local has the type.
func TestCombined_GetType_CDNAuthoritativeMiss(t *testing.T) {
	cdnPkg := gormPackage()
	// Remove "DB" so CDN authoritatively says it doesn't exist.
	delete(cdnPkg.Types, "DB")
	cdn := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": cdnPkg})
	local := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, local)

	_, err := c.GetType("gorm.io/gorm", "DB")
	require.Error(t, err)
	// Error must mention "authoritative" to confirm the CDN path was taken.
	assert.Contains(t, err.Error(), "authoritative")
}

// CDN has the package but GetType returns a non-nil error (transient failure).
// Must fall back to local.
func TestCombined_GetType_CDNTransientError_FallsBackToLocal(t *testing.T) {
	local := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})

	// CDN knows gorm.io/gorm but returns a transient error for "DB" (type absent in CDN
	// package, missingReturnsError=true). Must fall back to local.
	cdnPkg := &core.GoStdlibPackage{
		ImportPath: "gorm.io/gorm",
		Types:      map[string]*core.GoStdlibType{}, // no DB type → transient error returned
		Functions:  map[string]*core.GoStdlibFunction{},
		Constants:  map[string]*core.GoStdlibConstant{},
		Variables:  map[string]*core.GoStdlibVariable{},
	}
	cdn2 := newMockTransient(map[string]*core.GoStdlibPackage{"gorm.io/gorm": cdnPkg})
	c2 := NewGoThirdPartyCombinedLoader(cdn2, local)

	typ, err := c2.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	require.NotNil(t, typ)
	assert.Equal(t, "DB", typ.Name)
}

// CDN transient error with no local loader: error is propagated.
func TestCombined_GetType_CDNTransientError_NoLocal(t *testing.T) {
	cdnPkg := &core.GoStdlibPackage{
		ImportPath: "gorm.io/gorm",
		Types:      map[string]*core.GoStdlibType{},
		Functions:  map[string]*core.GoStdlibFunction{},
		Constants:  map[string]*core.GoStdlibConstant{},
		Variables:  map[string]*core.GoStdlibVariable{},
	}
	cdn := newMockTransient(map[string]*core.GoStdlibPackage{"gorm.io/gorm": cdnPkg})
	c := NewGoThirdPartyCombinedLoader(cdn, nil)

	_, err := c.GetType("gorm.io/gorm", "DB")
	require.Error(t, err)
	// Falls through to "not found in any loader" since local is nil.
	assert.Contains(t, err.Error(), "not found in any loader")
}

// ---------------------------------------------------------------------------
// GetFunction
// ---------------------------------------------------------------------------

func TestCombined_GetFunction_CDNFirst(t *testing.T) {
	cdn := newMock(map[string]*core.GoStdlibPackage{"github.com/gin-gonic/gin": ginPackage()})
	local := newMock(map[string]*core.GoStdlibPackage{"github.com/gin-gonic/gin": ginPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, local)

	fn, err := c.GetFunction("github.com/gin-gonic/gin", "Default")
	require.NoError(t, err)
	require.NotNil(t, fn)
	assert.Equal(t, "Default", fn.Name)
}

func TestCombined_GetFunction_LocalFallback(t *testing.T) {
	cdn := newMock(nil) // CDN doesn't know gin
	local := newMock(map[string]*core.GoStdlibPackage{"github.com/gin-gonic/gin": ginPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, local)

	fn, err := c.GetFunction("github.com/gin-gonic/gin", "Default")
	require.NoError(t, err)
	require.NotNil(t, fn)
	assert.Equal(t, "Default", fn.Name)
}

func TestCombined_GetFunction_BothMiss(t *testing.T) {
	// Both loaders are non-nil but neither knows the package.
	// CDN skips (ValidateImport false), local returns its own "not found" error.
	cdn := newMock(nil)
	local := newMock(nil)
	c := NewGoThirdPartyCombinedLoader(cdn, local)

	_, err := c.GetFunction("unknown.io/pkg", "Foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown.io/pkg")
}

func TestCombined_GetFunction_CDNAuthoritativeMiss(t *testing.T) {
	ginPkg := ginPackage()
	delete(ginPkg.Functions, "Default") // CDN knows gin but not Default
	cdn := newMock(map[string]*core.GoStdlibPackage{"github.com/gin-gonic/gin": ginPkg})
	local := newMock(map[string]*core.GoStdlibPackage{"github.com/gin-gonic/gin": ginPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, local)

	_, err := c.GetFunction("github.com/gin-gonic/gin", "Default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authoritative")
}

func TestCombined_GetFunction_CDNTransientError_FallsBackToLocal(t *testing.T) {
	cdnPkg := &core.GoStdlibPackage{
		ImportPath: "github.com/gin-gonic/gin",
		Types:      map[string]*core.GoStdlibType{},
		Functions:  map[string]*core.GoStdlibFunction{}, // Default absent → transient error
		Constants:  map[string]*core.GoStdlibConstant{},
		Variables:  map[string]*core.GoStdlibVariable{},
	}
	cdn := newMockTransient(map[string]*core.GoStdlibPackage{"github.com/gin-gonic/gin": cdnPkg})
	local := newMock(map[string]*core.GoStdlibPackage{"github.com/gin-gonic/gin": ginPackage()})
	c := NewGoThirdPartyCombinedLoader(cdn, local)

	fn, err := c.GetFunction("github.com/gin-gonic/gin", "Default")
	require.NoError(t, err)
	require.NotNil(t, fn)
	assert.Equal(t, "Default", fn.Name)
}

func TestCombined_GetFunction_BothNil(t *testing.T) {
	c := NewGoThirdPartyCombinedLoader(nil, nil)
	_, err := c.GetFunction("gorm.io/gorm", "Open")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in any loader")
}

func TestCombined_GetFunction_CDNTransientError_NoLocal(t *testing.T) {
	cdnPkg := &core.GoStdlibPackage{
		ImportPath: "github.com/gin-gonic/gin",
		Types:      map[string]*core.GoStdlibType{},
		Functions:  map[string]*core.GoStdlibFunction{},
		Constants:  map[string]*core.GoStdlibConstant{},
		Variables:  map[string]*core.GoStdlibVariable{},
	}
	cdn := newMockTransient(map[string]*core.GoStdlibPackage{"github.com/gin-gonic/gin": cdnPkg})
	c := NewGoThirdPartyCombinedLoader(cdn, nil)

	_, err := c.GetFunction("github.com/gin-gonic/gin", "Default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in any loader")
}

// ---------------------------------------------------------------------------
// PackageCount
// ---------------------------------------------------------------------------

func TestCombined_PackageCount(t *testing.T) {
	cdn := newMock(map[string]*core.GoStdlibPackage{
		"gorm.io/gorm":              gormPackage(),
		"github.com/gin-gonic/gin": ginPackage(),
	})
	local := newMock(map[string]*core.GoStdlibPackage{
		"internal.company.com/mylib": internalPackage(),
	})
	c := NewGoThirdPartyCombinedLoader(cdn, local)
	assert.Equal(t, 3, c.PackageCount())
}

func TestCombined_PackageCount_CDNNil(t *testing.T) {
	local := newMock(map[string]*core.GoStdlibPackage{"gorm.io/gorm": gormPackage()})
	c := NewGoThirdPartyCombinedLoader(nil, local)
	assert.Equal(t, 1, c.PackageCount())
}

func TestCombined_PackageCount_LocalNil(t *testing.T) {
	cdn := newMock(map[string]*core.GoStdlibPackage{
		"gorm.io/gorm":              gormPackage(),
		"github.com/gin-gonic/gin": ginPackage(),
	})
	c := NewGoThirdPartyCombinedLoader(cdn, nil)
	assert.Equal(t, 2, c.PackageCount())
}

func TestCombined_PackageCount_BothNil(t *testing.T) {
	c := NewGoThirdPartyCombinedLoader(nil, nil)
	assert.Equal(t, 0, c.PackageCount())
}

// Large counts as documented in the spec (100 CDN + 20 local = 120).
func TestCombined_PackageCount_LargeCounts(t *testing.T) {
	cdnPkgs := make(map[string]*core.GoStdlibPackage, 100)
	for i := range 100 {
		path := fmt.Sprintf("cdn.example.com/pkg%d", i)
		cdnPkgs[path] = &core.GoStdlibPackage{ImportPath: path,
			Types: map[string]*core.GoStdlibType{}, Functions: map[string]*core.GoStdlibFunction{},
			Constants: map[string]*core.GoStdlibConstant{}, Variables: map[string]*core.GoStdlibVariable{}}
	}
	localPkgs := make(map[string]*core.GoStdlibPackage, 20)
	for i := range 20 {
		path := fmt.Sprintf("local.example.com/internal%d", i)
		localPkgs[path] = &core.GoStdlibPackage{ImportPath: path,
			Types: map[string]*core.GoStdlibType{}, Functions: map[string]*core.GoStdlibFunction{},
			Constants: map[string]*core.GoStdlibConstant{}, Variables: map[string]*core.GoStdlibVariable{}}
	}
	cdn := newMock(cdnPkgs)
	local := newMock(localPkgs)
	c := NewGoThirdPartyCombinedLoader(cdn, local)
	assert.Equal(t, 120, c.PackageCount())
}

// ---------------------------------------------------------------------------
// Unused variable to satisfy the errors import (used by mock)
// ---------------------------------------------------------------------------

var _ = errors.New // keep errors import in scope via blank identifier


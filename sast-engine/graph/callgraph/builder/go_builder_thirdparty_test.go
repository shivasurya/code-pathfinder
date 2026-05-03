package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestThirdPartyResolution_Check25_MethodValidation tests the full pipeline:
// go.mod dependency → vendor/ source → tree-sitter extraction → Pattern 1b Check 2.5 resolution.
func TestThirdPartyResolution_Check25_MethodValidation(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create go.mod with gorm dependency
	goMod := `module testapp

go 1.21

require gorm.io/gorm v1.25.7
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// 2. Create vendor/gorm.io/gorm/ with type metadata source
	vendorDir := filepath.Join(tmpDir, "vendor", "gorm.io", "gorm")
	err = os.MkdirAll(vendorDir, 0755)
	require.NoError(t, err)

	gormSrc := `package gorm

type DB struct {
	Error error
}

func (db *DB) Where(query interface{}, args ...interface{}) *DB {
	return db
}

func (db *DB) Raw(sql string, values ...interface{}) *DB {
	return db
}

func (db *DB) Exec(sql string, values ...interface{}) *DB {
	return db
}

func Open(dialector interface{}) (*DB, error) {
	return nil, nil
}
`
	err = os.WriteFile(filepath.Join(vendorDir, "gorm.go"), []byte(gormSrc), 0644)
	require.NoError(t, err)

	// 3. Create user code that uses gorm
	mainSrc := `package main

import "gorm.io/gorm"

func handler(db *gorm.DB) {
	input := "user input"
	db.Raw(input)
	db.Where(input)
	db.Exec(input)
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainSrc), 0644)
	require.NoError(t, err)

	// 4. Build code graph and call graph
	codeGraph := graph.Initialize(tmpDir, nil)
	require.NotNil(t, codeGraph)

	goRegistry, err := resolution.BuildGoModuleRegistry(tmpDir)
	require.NoError(t, err)

	// Initialize third-party loader (this is what scan.go would do)
	InitGoThirdPartyLoader(goRegistry, tmpDir, false, nil)
	require.NotNil(t, goRegistry.ThirdPartyLoader, "ThirdPartyLoader should be initialized")

	goTypeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)

	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, goTypeEngine, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, callGraph)

	// 5. Verify that third-party methods resolved correctly via Check 2.5
	// Look for call sites from handler function
	handlerFQN := "testapp.handler"
	callSites, ok := callGraph.CallSites[handlerFQN]
	require.True(t, ok, "handler function should have call sites")

	resolvedTargets := make(map[string]bool)
	for _, cs := range callSites {
		if cs.Resolved {
			resolvedTargets[cs.TargetFQN] = true
		}
	}

	// These should be resolved via Check 2.5 (third-party vendor/)
	assert.True(t, resolvedTargets["gorm.io/gorm.DB.Raw"],
		"db.Raw() should resolve to gorm.io/gorm.DB.Raw via Check 2.5")
	assert.True(t, resolvedTargets["gorm.io/gorm.DB.Where"],
		"db.Where() should resolve to gorm.io/gorm.DB.Where via Check 2.5")
	assert.True(t, resolvedTargets["gorm.io/gorm.DB.Exec"],
		"db.Exec() should resolve to gorm.io/gorm.DB.Exec via Check 2.5")
}

// TestThirdPartyResolution_SubpackagePath tests resolution for subpackages
// within a third-party module (e.g., github.com/gin-gonic/gin/binding).
func TestThirdPartyResolution_SubpackagePath(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module testapp

go 1.21

require github.com/gin-gonic/gin v1.9.1
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create vendor with gin Context
	vendorDir := filepath.Join(tmpDir, "vendor", "github.com", "gin-gonic", "gin")
	err = os.MkdirAll(vendorDir, 0755)
	require.NoError(t, err)

	ginSrc := `package gin

type Context struct{}

func (c *Context) Query(key string) string { return "" }
func (c *Context) Param(key string) string { return "" }

type Engine struct{}

func Default() *Engine { return nil }
`
	err = os.WriteFile(filepath.Join(vendorDir, "gin.go"), []byte(ginSrc), 0644)
	require.NoError(t, err)

	mainSrc := `package main

import "github.com/gin-gonic/gin"

func handler(c *gin.Context) {
	q := c.Query("search")
	_ = q
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainSrc), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	goRegistry, err := resolution.BuildGoModuleRegistry(tmpDir)
	require.NoError(t, err)

	InitGoThirdPartyLoader(goRegistry, tmpDir, false, nil)
	goTypeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)

	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, goTypeEngine, nil, nil)
	require.NoError(t, err)

	handlerFQN := "testapp.handler"
	callSites := callGraph.CallSites[handlerFQN]

	resolvedTargets := make(map[string]bool)
	for _, cs := range callSites {
		if cs.Resolved {
			resolvedTargets[cs.TargetFQN] = true
		}
	}

	assert.True(t, resolvedTargets["github.com/gin-gonic/gin.Context.Query"],
		"c.Query() should resolve to github.com/gin-gonic/gin.Context.Query")
}

// TestThirdPartyResolution_TypeSourceLabeling verifies that call sites resolved
// via the ThirdPartyLoader carry TypeSource == "thirdparty_local", enabling
// the resolution-report stats to attribute them correctly.
func TestThirdPartyResolution_TypeSourceLabeling(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module testapp

go 1.21

require gorm.io/gorm v1.25.7
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644))

	// Vendor gorm with a type and method.
	vendorDir := filepath.Join(tmpDir, "vendor", "gorm.io", "gorm")
	require.NoError(t, os.MkdirAll(vendorDir, 0755))
	gormSrc := `package gorm
type DB struct{}
func (db *DB) Find(dest interface{}) *DB { return db }
func Open(dialector interface{}) (*DB, error) { return nil, nil }
`
	require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "gorm.go"), []byte(gormSrc), 0644))

	// User code: Pattern 1a (gorm.Open) + Pattern 1b Check 2.5 (db.Find).
	mainSrc := `package main

import "gorm.io/gorm"

func handler(db *gorm.DB) {
	db.Find(nil)
}

func setup() {
	gorm.Open(nil)
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainSrc), 0644))

	codeGraph := graph.Initialize(tmpDir, nil)
	require.NotNil(t, codeGraph)

	goRegistry, err := resolution.BuildGoModuleRegistry(tmpDir)
	require.NoError(t, err)
	InitGoThirdPartyLoader(goRegistry, tmpDir, false, nil)
	require.NotNil(t, goRegistry.ThirdPartyLoader)

	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, resolution.NewGoTypeInferenceEngine(goRegistry), nil, nil)
	require.NoError(t, err)

	// Check 2.5 path: db.Find — method call on third-party receiver.
	handlerSites := callGraph.CallSites["testapp.handler"]
	var findSite *core.CallSite
	for i := range handlerSites {
		if handlerSites[i].TargetFQN == "gorm.io/gorm.DB.Find" {
			findSite = &handlerSites[i]
			break
		}
	}
	require.NotNil(t, findSite, "db.Find() should be resolved")
	assert.Equal(t, "thirdparty_local", findSite.TypeSource,
		"Check 2.5 resolution must carry TypeSource=thirdparty_local")

	// Pattern 1a path: gorm.Open — import-qualified function call.
	setupSites := callGraph.CallSites["testapp.setup"]
	var openSite *core.CallSite
	for i := range setupSites {
		if setupSites[i].TargetFQN == "gorm.io/gorm.Open" {
			openSite = &setupSites[i]
			break
		}
	}
	require.NotNil(t, openSite, "gorm.Open() should be resolved")
	assert.Equal(t, "thirdparty_local", openSite.TypeSource,
		"Pattern 1a third-party resolution must carry TypeSource=thirdparty_local")
}

package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseGoModRequires tests parsing require directives from go.mod.
func TestParseGoModRequires(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module github.com/example/myapp

go 1.21

require (
	gorm.io/gorm v1.25.7
	github.com/gin-gonic/gin v1.9.1
	github.com/stretchr/testify v1.9.0 // indirect
)

require github.com/redis/go-redis/v9 v9.5.1
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	requires := parseGoModRequires(tmpDir)

	assert.Equal(t, "v1.25.7", requires["gorm.io/gorm"])
	assert.Equal(t, "v1.9.1", requires["github.com/gin-gonic/gin"])
	assert.Equal(t, "v1.9.0", requires["github.com/stretchr/testify"])
	assert.Equal(t, "v9.5.1", requires["github.com/redis/go-redis/v9"])
}

// TestExtractGoPackageWithTreeSitter tests extracting type metadata from Go source.
func TestExtractGoPackageWithTreeSitter(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a minimal third-party package source file
	src := `package gorm

// DB is the main database handle.
type DB struct {
	Error        error
	RowsAffected int64
	Statement    *Statement
}

// Statement holds the current query state.
type Statement struct {
	SQL string
}

// Dialector is the database driver interface.
type Dialector interface {
	Initialize(db *DB) error
	Name() string
}

// Where adds a WHERE clause.
func (db *DB) Where(query interface{}, args ...interface{}) *DB {
	return db
}

// Find retrieves records.
func (db *DB) Find(dest interface{}, conds ...interface{}) *DB {
	return db
}

// Raw executes a raw SQL query.
func (db *DB) Raw(sql string, values ...interface{}) *DB {
	return db
}

// Exec executes a raw SQL statement.
func (db *DB) Exec(sql string, values ...interface{}) *DB {
	return db
}

// Create inserts a new record.
func (db *DB) Create(value interface{}) *DB {
	return db
}

// Open creates a new DB connection.
func Open(dialector Dialector, opts ...Option) (*DB, error) {
	return nil, nil
}

// Option configures the DB.
type Option struct{}

// unexportedFunc should not be extracted.
func unexportedFunc() {}

// unexportedType should not be extracted.
type unexportedType struct {
	field string
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "gorm.go"), []byte(src), 0644)
	require.NoError(t, err)

	pkg, err := extractGoPackageWithTreeSitter("gorm.io/gorm", tmpDir)
	require.NoError(t, err)
	require.NotNil(t, pkg)

	// Verify package metadata
	assert.Equal(t, "gorm.io/gorm", pkg.ImportPath)

	// Verify types extracted
	assert.Contains(t, pkg.Types, "DB")
	assert.Contains(t, pkg.Types, "Statement")
	assert.Contains(t, pkg.Types, "Dialector")
	assert.Contains(t, pkg.Types, "Option")

	// Verify unexported types NOT extracted
	assert.NotContains(t, pkg.Types, "unexportedType")

	// Verify DB type details
	dbType := pkg.Types["DB"]
	assert.Equal(t, "struct", dbType.Kind)
	assert.NotNil(t, dbType.Methods)

	// Verify methods on DB
	assert.Contains(t, dbType.Methods, "Where")
	assert.Contains(t, dbType.Methods, "Find")
	assert.Contains(t, dbType.Methods, "Raw")
	assert.Contains(t, dbType.Methods, "Exec")
	assert.Contains(t, dbType.Methods, "Create")

	// Verify Where method details
	whereMethod := dbType.Methods["Where"]
	assert.Equal(t, "Where", whereMethod.Name)
	assert.NotEmpty(t, whereMethod.Params)
	assert.NotEmpty(t, whereMethod.Returns)
	assert.Equal(t, "*DB", whereMethod.Returns[0].Type)

	// Verify DB fields
	assert.NotEmpty(t, dbType.Fields)
	fieldNames := make([]string, 0, len(dbType.Fields))
	for _, f := range dbType.Fields {
		fieldNames = append(fieldNames, f.Name)
	}
	assert.Contains(t, fieldNames, "Error")
	assert.Contains(t, fieldNames, "RowsAffected")
	assert.Contains(t, fieldNames, "Statement")

	// Verify Dialector interface
	dialType := pkg.Types["Dialector"]
	assert.Equal(t, "interface", dialType.Kind)
	assert.Contains(t, dialType.Methods, "Initialize")
	assert.Contains(t, dialType.Methods, "Name")

	// Verify package-level function
	assert.Contains(t, pkg.Functions, "Open")
	openFn := pkg.Functions["Open"]
	assert.Equal(t, "Open", openFn.Name)
	assert.Len(t, openFn.Returns, 2)
	assert.Equal(t, "*DB", openFn.Returns[0].Type)
	assert.Equal(t, "error", openFn.Returns[1].Type)

	// Verify unexported function NOT extracted
	assert.NotContains(t, pkg.Functions, "unexportedFunc")
}

// TestGoThirdPartyLocalLoader_VendorResolution tests resolution from vendor/.
func TestGoThirdPartyLocalLoader_VendorResolution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod with dependency
	goMod := `module github.com/example/myapp

go 1.21

require gorm.io/gorm v1.25.7
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create vendor/gorm.io/gorm/ with Go source
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

func Open(dialector interface{}) (*DB, error) {
	return nil, nil
}
`
	err = os.WriteFile(filepath.Join(vendorDir, "gorm.go"), []byte(gormSrc), 0644)
	require.NoError(t, err)

	// Create loader and test
	loader := NewGoThirdPartyLocalLoader(tmpDir, false, nil)

	// Validate import
	assert.True(t, loader.ValidateImport("gorm.io/gorm"))
	assert.False(t, loader.ValidateImport("unknown.io/pkg"))

	// Get type
	dbType, err := loader.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	require.NotNil(t, dbType)

	assert.Equal(t, "struct", dbType.Kind)
	assert.Contains(t, dbType.Methods, "Where")
	assert.Contains(t, dbType.Methods, "Raw")

	// Get function
	openFn, err := loader.GetFunction("gorm.io/gorm", "Open")
	require.NoError(t, err)
	require.NotNil(t, openFn)
	assert.Equal(t, "Open", openFn.Name)
	assert.Len(t, openFn.Returns, 2)
	assert.Equal(t, "*DB", openFn.Returns[0].Type)
}

// TestGoThirdPartyLocalLoader_MethodValidation tests that method existence
// can be validated — the key use case for Pattern 1b Check 2.5.
func TestGoThirdPartyLocalLoader_MethodValidation(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module github.com/example/myapp

go 1.21

require github.com/gin-gonic/gin v1.9.1
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	vendorDir := filepath.Join(tmpDir, "vendor", "github.com", "gin-gonic", "gin")
	err = os.MkdirAll(vendorDir, 0755)
	require.NoError(t, err)

	ginSrc := `package gin

type Context struct {
	Request  interface{}
	Writer   interface{}
}

func (c *Context) Query(key string) string {
	return ""
}

func (c *Context) Param(key string) string {
	return ""
}

func (c *Context) PostForm(key string) string {
	return ""
}

func (c *Context) JSON(code int, obj interface{}) {}

type Engine struct{}

func Default() *Engine { return nil }
`
	err = os.WriteFile(filepath.Join(vendorDir, "context.go"), []byte(ginSrc), 0644)
	require.NoError(t, err)

	loader := NewGoThirdPartyLocalLoader(tmpDir, false, nil)

	// Verify type and methods
	ctxType, err := loader.GetType("github.com/gin-gonic/gin", "Context")
	require.NoError(t, err)
	require.NotNil(t, ctxType)

	// Methods that exist
	_, hasQuery := ctxType.Methods["Query"]
	assert.True(t, hasQuery, "Context should have Query method")

	_, hasParam := ctxType.Methods["Param"]
	assert.True(t, hasParam, "Context should have Param method")

	_, hasPostForm := ctxType.Methods["PostForm"]
	assert.True(t, hasPostForm, "Context should have PostForm method")

	_, hasJSON := ctxType.Methods["JSON"]
	assert.True(t, hasJSON, "Context should have JSON method")

	// Method that doesn't exist — this is what Check 2.5 needs to detect
	_, hasFake := ctxType.Methods["FakeMethod"]
	assert.False(t, hasFake, "Context should NOT have FakeMethod")

	// Return type inference
	defaultFn, err := loader.GetFunction("github.com/gin-gonic/gin", "Default")
	require.NoError(t, err)
	require.NotNil(t, defaultFn)
	assert.Equal(t, "*Engine", defaultFn.Returns[0].Type)
}

// TestInterfaceEmbedding_SamePackage tests that embedded interface methods are flattened.
// This is the posthog.Client pattern: Client embeds EnqueueClient and io.Closer.
func TestInterfaceEmbedding_SamePackage(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate posthog package with interface embedding
	src := `package posthog

type Client interface {
	io.Closer
	EnqueueClient

	IsFeatureEnabled(payload FeatureFlagPayload) (interface{}, error)
	GetFeatureFlag(payload FeatureFlagPayload) (interface{}, error)
}

type EnqueueClient interface {
	Enqueue(msg Message) error
}

type Message interface{}
type FeatureFlagPayload struct{}
type Capture struct {
	DistinctId string
	Event      string
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "posthog.go"), []byte(src), 0644)
	require.NoError(t, err)

	pkg, err := extractGoPackageWithTreeSitter("github.com/posthog/posthog-go", tmpDir)
	require.NoError(t, err)
	require.NotNil(t, pkg)

	// Client interface should exist
	clientType, ok := pkg.Types["Client"]
	require.True(t, ok, "Client type should exist")
	assert.Equal(t, "interface", clientType.Kind)

	// Direct methods
	assert.Contains(t, clientType.Methods, "IsFeatureEnabled", "Direct method should be extracted")
	assert.Contains(t, clientType.Methods, "GetFeatureFlag", "Direct method should be extracted")

	// Flattened from EnqueueClient (same-package embed)
	assert.Contains(t, clientType.Methods, "Enqueue",
		"Enqueue should be flattened from embedded EnqueueClient interface")

	// Flattened from io.Closer (cross-package embed, well-known)
	assert.Contains(t, clientType.Methods, "Close",
		"Close should be flattened from embedded io.Closer interface")

	// Verify Embeds field captured the embedding info
	assert.Contains(t, clientType.Embeds, "EnqueueClient")
	assert.Contains(t, clientType.Embeds, "io.Closer")
}

// TestInterfaceEmbedding_MultiLevel tests multi-level embedding:
// A embeds B, B embeds C → A should have C's methods.
func TestInterfaceEmbedding_MultiLevel(t *testing.T) {
	tmpDir := t.TempDir()

	src := `package pkg

type Level1 interface {
	Level2
	L1Method() string
}

type Level2 interface {
	Level3
	L2Method() int
}

type Level3 interface {
	L3Method() error
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "levels.go"), []byte(src), 0644)
	require.NoError(t, err)

	pkg, err := extractGoPackageWithTreeSitter("example.com/pkg", tmpDir)
	require.NoError(t, err)

	l1 := pkg.Types["Level1"]
	require.NotNil(t, l1)

	assert.Contains(t, l1.Methods, "L1Method", "Own method")
	assert.Contains(t, l1.Methods, "L2Method", "Flattened from Level2")
	assert.Contains(t, l1.Methods, "L3Method", "Flattened from Level3 (multi-level)")
}

// TestHasGoFiles tests the hasGoFiles helper.
func TestHasGoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty dir
	assert.False(t, hasGoFiles(tmpDir))

	// Dir with non-Go file
	err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# hi"), 0644)
	require.NoError(t, err)
	assert.False(t, hasGoFiles(tmpDir))

	// Dir with test file only
	err = os.WriteFile(filepath.Join(tmpDir, "foo_test.go"), []byte("package foo"), 0644)
	require.NoError(t, err)
	assert.False(t, hasGoFiles(tmpDir))

	// Dir with Go source file
	err = os.WriteFile(filepath.Join(tmpDir, "foo.go"), []byte("package foo"), 0644)
	require.NoError(t, err)
	assert.True(t, hasGoFiles(tmpDir))
}

// makeVendoredProject creates a minimal project with a vendored gorm stub
// and returns the project root directory. Shared by disk-cache tests.
func makeVendoredProject(t *testing.T, version string) string {
	t.Helper()
	projectDir := t.TempDir()

	goMod := "module github.com/example/myapp\n\ngo 1.21\n\nrequire gorm.io/gorm " + version + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644))

	vendorDir := filepath.Join(projectDir, "vendor", "gorm.io", "gorm")
	require.NoError(t, os.MkdirAll(vendorDir, 0755))

	gormSrc := `package gorm
type DB struct{ Error error }
func (db *DB) Where(query interface{}, args ...interface{}) *DB { return db }
`
	require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "gorm.go"), []byte(gormSrc), 0644))
	return projectDir
}

// TestDiskCacheWriteAndRead verifies that a cold extraction is persisted to disk
// and a second loader instance reads from the disk cache instead of re-parsing.
func TestDiskCacheWriteAndRead(t *testing.T) {
	projectDir := makeVendoredProject(t, "v1.25.7")

	// Cold run: loader extracts from vendor/ and writes to disk cache.
	loader1 := NewGoThirdPartyLocalLoader(projectDir, false, nil)
	dbType, err := loader1.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	require.NotNil(t, dbType)
	assert.Contains(t, dbType.Methods, "Where")

	// Cache-index.json must exist after the cold run.
	indexPath := filepath.Join(loader1.cacheDir, "cache-index.json")
	_, statErr := os.Stat(indexPath)
	assert.NoError(t, statErr, "cache-index.json should be written after cold extraction")

	// Package JSON file must exist.
	entry := loader1.diskIndex.Entries["gorm.io/gorm"]
	require.NotNil(t, entry, "cache-index.json should contain gorm.io/gorm entry")
	pkgFile := filepath.Join(loader1.cacheDir, entry.File)
	_, statErr = os.Stat(pkgFile)
	assert.NoError(t, statErr, "package JSON file should exist on disk")

	// Warm run: new loader with same projectDir reads from disk cache.
	// Remove vendor/ to prove it's not re-parsing from source.
	require.NoError(t, os.RemoveAll(filepath.Join(projectDir, "vendor")))

	loader2 := NewGoThirdPartyLocalLoader(projectDir, false, nil)
	dbType2, err := loader2.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	require.NotNil(t, dbType2, "disk cache hit should return the type even without vendor/")
	assert.Contains(t, dbType2.Methods, "Where", "disk-cached type should retain methods")
}

// TestCacheVersionMismatch verifies that a go.mod version bump causes re-extraction
// and invalidates the stale disk cache entry.
func TestCacheVersionMismatch(t *testing.T) {
	projectDir := makeVendoredProject(t, "v1.25.7")

	// Cold run at v1.25.7.
	loader1 := NewGoThirdPartyLocalLoader(projectDir, false, nil)
	_, err := loader1.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)

	// Simulate a go.mod upgrade to v1.25.8: update go.mod and add new method to vendor source.
	goMod := "module github.com/example/myapp\n\ngo 1.21\n\nrequire gorm.io/gorm v1.25.8\n"
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644))

	vendorDir := filepath.Join(projectDir, "vendor", "gorm.io", "gorm")
	require.NoError(t, os.MkdirAll(vendorDir, 0755))
	newSrc := `package gorm
type DB struct{ Error error }
func (db *DB) Where(query interface{}, args ...interface{}) *DB { return db }
func (db *DB) Save(value interface{}) *DB { return db }
`
	require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "gorm.go"), []byte(newSrc), 0644))

	// New loader: same cache dir, but go.mod says v1.25.8 — cache entry is v1.25.7 → miss.
	loader2 := NewGoThirdPartyLocalLoader(projectDir, false, nil)
	dbType, err := loader2.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	require.NotNil(t, dbType)

	// New method from v1.25.8 source must be present (re-extraction happened).
	assert.Contains(t, dbType.Methods, "Save", "re-extraction should pick up new method after version bump")

	// Cache-index entry should now reflect v1.25.8.
	entry := loader2.diskIndex.Entries["gorm.io/gorm"]
	require.NotNil(t, entry)
	assert.Equal(t, "v1.25.8", entry.Version, "cache-index.json should be updated to new version")
}

// TestRefreshCacheFlush verifies that refreshCache=true wipes existing cache files
// and forces a fresh extraction on the next load.
func TestRefreshCacheFlush(t *testing.T) {
	projectDir := makeVendoredProject(t, "v1.25.7")

	// Cold run to populate cache.
	loader1 := NewGoThirdPartyLocalLoader(projectDir, false, nil)
	_, err := loader1.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	cacheDir := loader1.cacheDir

	// Verify cache files exist.
	indexPath := filepath.Join(cacheDir, "cache-index.json")
	_, statErr := os.Stat(indexPath)
	require.NoError(t, statErr, "cache-index.json should exist after cold run")

	// Inject a sentinel into the cached package JSON to detect whether it gets reused.
	entry := loader1.diskIndex.Entries["gorm.io/gorm"]
	require.NotNil(t, entry)
	pkgPath := filepath.Join(cacheDir, entry.File)
	data, err := os.ReadFile(pkgPath)
	require.NoError(t, err)
	var pkgMap map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &pkgMap))
	pkgMap["_sentinel"] = "stale"
	patched, err := json.Marshal(pkgMap)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(pkgPath, patched, 0644))

	// Refresh run: cache dir is wiped, extraction happens from vendor/ again.
	loader2 := NewGoThirdPartyLocalLoader(projectDir, true, nil)
	dbType, err := loader2.GetType("gorm.io/gorm", "DB")
	require.NoError(t, err)
	require.NotNil(t, dbType)
	assert.Contains(t, dbType.Methods, "Where", "fresh extraction should succeed after cache flush")

	// The new cache entry should NOT contain the sentinel we injected.
	entry2 := loader2.diskIndex.Entries["gorm.io/gorm"]
	require.NotNil(t, entry2)
	freshData, err := os.ReadFile(filepath.Join(cacheDir, entry2.File))
	require.NoError(t, err)
	assert.NotContains(t, string(freshData), `"_sentinel"`, "flushed cache should not contain stale sentinel")
}

// TestPackageCount verifies PackageCount reflects the number of go.mod requires.
func TestPackageCount(t *testing.T) {
	projectDir := makeVendoredProject(t, "v1.25.7")
	loader := NewGoThirdPartyLocalLoader(projectDir, false, nil)
	assert.Equal(t, 1, loader.PackageCount())
}

// TestGetType_NotFound verifies that GetType returns an error for unknown types.
func TestGoThirdPartyLocalGetType_NotFound(t *testing.T) {
	projectDir := makeVendoredProject(t, "v1.25.7")
	loader := NewGoThirdPartyLocalLoader(projectDir, false, nil)

	_, err := loader.GetType("gorm.io/gorm", "NonExistentType")
	assert.Error(t, err)
}

// TestGetFunction_NotFound verifies that GetFunction returns an error for unknown functions.
func TestGoThirdPartyLocalGetFunction_NotFound(t *testing.T) {
	projectDir := makeVendoredProject(t, "v1.25.7")
	loader := NewGoThirdPartyLocalLoader(projectDir, false, nil)

	_, err := loader.GetFunction("gorm.io/gorm", "NonExistentFunc")
	assert.Error(t, err)
}

// TestGetType_PackageNotFound verifies that GetType returns an error when the
// package cannot be located in vendor/ or GOMODCACHE.
func TestGetType_PackageNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	goMod := "module github.com/example/myapp\n\ngo 1.21\n\nrequire github.com/missing/pkg v1.0.0\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644))

	// No vendor/ directory, no GOMODCACHE entry → source not found.
	loader := NewGoThirdPartyLocalLoader(tmpDir, false, nil)
	loader.cacheDir = "" // disable disk cache so we go straight to findPackageSource

	_, err := loader.GetType("github.com/missing/pkg", "SomeType")
	assert.Error(t, err)
}

// TestFindPackageSource_GOMODCACHE verifies that findPackageSource falls back to
// GOMODCACHE when vendor/ does not contain the package.
func TestFindPackageSource_GOMODCACHE(t *testing.T) {
	projectDir := t.TempDir()

	goMod := "module github.com/example/myapp\n\ngo 1.21\n\nrequire example.com/mylib v1.2.3\n"
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644))

	// Set up a fake GOMODCACHE with the module source.
	fakeCache := t.TempDir()
	modDir := filepath.Join(fakeCache, "example.com", "mylib@v1.2.3")
	require.NoError(t, os.MkdirAll(modDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "mylib.go"), []byte(`package mylib

type Client struct{}
func (c *Client) Call() string { return "" }
`), 0644))

	t.Setenv("GOMODCACHE", fakeCache)

	loader := NewGoThirdPartyLocalLoader(projectDir, false, nil)
	loader.cacheDir = "" // disable disk cache

	typ, err := loader.GetType("example.com/mylib", "Client")
	require.NoError(t, err)
	require.NotNil(t, typ)
	assert.Contains(t, typ.Methods, "Call")
}

// TestFindPackageSource_GOMODCACHE_Subpackage verifies resolution of a subpackage
// (import path = module/subpkg) from GOMODCACHE.
func TestFindPackageSource_GOMODCACHE_Subpackage(t *testing.T) {
	projectDir := t.TempDir()

	goMod := "module github.com/example/myapp\n\ngo 1.21\n\nrequire example.com/sdk v2.0.0\n"
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644))

	fakeCache := t.TempDir()
	subDir := filepath.Join(fakeCache, "example.com", "sdk@v2.0.0", "auth")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "auth.go"), []byte(`package auth

type Token struct{}
func (t *Token) Verify() bool { return true }
`), 0644))

	t.Setenv("GOMODCACHE", fakeCache)

	loader := NewGoThirdPartyLocalLoader(projectDir, false, nil)
	loader.cacheDir = ""

	typ, err := loader.GetType("example.com/sdk/auth", "Token")
	require.NoError(t, err)
	require.NotNil(t, typ)
	assert.Contains(t, typ.Methods, "Verify")
}

// TestLoadCacheIndex_InvalidJSON verifies that a corrupt cache-index.json
// produces an empty (not nil) index rather than a crash.
func TestLoadCacheIndex_InvalidJSON(t *testing.T) {
	projectDir := makeVendoredProject(t, "v1.25.7")
	loader := NewGoThirdPartyLocalLoader(projectDir, false, nil)

	// Overwrite cache-index.json with garbage.
	require.NoError(t, os.WriteFile(
		filepath.Join(loader.cacheDir, "cache-index.json"),
		[]byte("not-valid-json{{{"),
		0644,
	))

	// Re-load the index — should return an empty index, not panic.
	idx := loader.loadCacheIndex()
	require.NotNil(t, idx)
	assert.Empty(t, idx.Entries)
}

// TestWriteToDiskCache_NilDiskIndex verifies writeToDiskCache is a no-op when
// diskIndex is nil (disk cache unavailable).
func TestWriteToDiskCache_NilDiskIndex(t *testing.T) {
	projectDir := makeVendoredProject(t, "v1.25.7")
	loader := NewGoThirdPartyLocalLoader(projectDir, false, nil)
	loader.diskIndex = nil // simulate unavailable disk cache

	// Should not panic.
	loader.writeToDiskCache("gorm.io/gorm", nil)
}

// TestIsExported_EmptyString verifies isExported handles empty input without panic.
func TestIsExported_EmptyString(t *testing.T) {
	assert.False(t, isExported(""))
}

// TestExtractMethodDecl_NoReceiverType verifies that method declarations with
// no parseable receiver type are silently skipped.
func TestExtractMethodDecl_NoReceiverType(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid method + exported function only; no unusual receiver syntax.
	src := `package mypkg

type Svc struct{}

func (s *Svc) DoWork() string { return "" }
func StandaloneFunc() int { return 0 }
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "svc.go"), []byte(src), 0644))

	pkg, err := extractGoPackageWithTreeSitter("example.com/mypkg", tmpDir)
	require.NoError(t, err)
	require.NotNil(t, pkg)

	assert.Contains(t, pkg.Types, "Svc")
	assert.Contains(t, pkg.Types["Svc"].Methods, "DoWork")
	assert.Contains(t, pkg.Functions, "StandaloneFunc")
}

// TestInitDiskCache_NoGoMod verifies that a project with no go.mod produces a
// loader with PackageCount == 0 (no crash, graceful degradation).
func TestInitDiskCache_NoGoMod(t *testing.T) {
	emptyDir := t.TempDir()
	loader := NewGoThirdPartyLocalLoader(emptyDir, false, nil)
	assert.Equal(t, 0, loader.PackageCount())
}

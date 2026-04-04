package registry

import (
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
	fieldNames := make([]string, 0)
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
	loader := NewGoThirdPartyLocalLoader(tmpDir, nil)

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

	loader := NewGoThirdPartyLocalLoader(tmpDir, nil)

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

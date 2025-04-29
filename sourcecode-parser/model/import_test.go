package model

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestNewImportType(t *testing.T) {
	importType := NewImportType("java.util.List", "test/Test.java")

	assert.NotNil(t, importType)
	assert.Equal(t, "java.util.List", importType.ImportedType)
	assert.Equal(t, "test/Test.java", importType.SourceDeclaration)
}

func TestImportType_Insert(t *testing.T) {
	// Create a temporary SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	// Create the import_decl table with wrong schema to cause prepare to fail
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS import_decl (
			id INTEGER PRIMARY KEY AUTOINCREMENT
		)
	`)
	assert.NoError(t, err)

	// Test prepare statement failure (wrong schema)
	importType := &ImportType{
		ImportedType:      "java.util.List",
		SourceDeclaration: "test/Test.java",
	}
	err = importType.Insert(db)
	assert.Error(t, err)

	// Drop and recreate table with correct schema
	_, err = db.Exec("DROP TABLE IF EXISTS import_decl")
	assert.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS import_decl (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			import_type TEXT NOT NULL,
			import_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(import_type, import_name, file_path)
		)
	`)
	assert.NoError(t, err)

	// Test successful insertion
	err = importType.Insert(db)
	assert.NoError(t, err)

	// Verify the insertion
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM import_decl WHERE import_type = ?", importType.ImportedType).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Test duplicate insertion (should fail due to UNIQUE constraint)
	err = importType.Insert(db)
	assert.Error(t, err)
}

func TestImportType_GetAPrimaryQlClass(t *testing.T) {
	importType := &ImportType{}
	assert.Equal(t, "ImportType", importType.GetAPrimaryQlClass())
}

func TestImportType_GetImportedType(t *testing.T) {
	importType := &ImportType{ImportedType: "java.util.List"}
	assert.Equal(t, "java.util.List", importType.GetImportedType())
}

func TestImportType_ToString(t *testing.T) {
	importType := &ImportType{ImportedType: "java.util.List"}
	assert.Equal(t, "import java.util.List;", importType.ToString())
}

func TestImportType_GetProxyEnv(t *testing.T) {
	importType := &ImportType{
		ImportedType:      "java.util.List",
		SourceDeclaration: "test/Test.java",
	}

	proxyEnv := importType.GetProxyEnv()

	assert.Equal(t, "java.util.List", proxyEnv["GetImportType"])
	assert.Equal(t, "test/Test.java", proxyEnv["GetSourceDeclaration"])
	assert.Equal(t, "ImportType", proxyEnv["GetAPrimaryQlClass"])
}

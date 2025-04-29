package model

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	_ "github.com/mattn/go-sqlite3"
)

func TestPackage(t *testing.T) {
	t.Run("NewPackage constructor", func(t *testing.T) {
		pkg := NewPackage(
			"com.example",
			[]string{"TestClass1", "TestClass2"},
			true,
			"complexity:10",
			"http://example.com",
		)

		// Verify all fields are set correctly
		assert.Equal(t, "com.example", pkg.QualifiedName)
		assert.Equal(t, []string{"TestClass1", "TestClass2"}, pkg.TopLevelTypes)
		assert.True(t, pkg.FromSource)
		assert.Equal(t, "complexity:10", pkg.Metrics)
		assert.Equal(t, "http://example.com", pkg.URL)
	})

	t.Run("GetFromSource", func(t *testing.T) {
		pkg := &Package{FromSource: true}
		assert.True(t, pkg.GetFromSource())

		pkg.FromSource = false
		assert.False(t, pkg.GetFromSource())
	})

	t.Run("GetAPrimaryQlClass", func(t *testing.T) {
		pkg := &Package{}
		assert.Equal(t, "Package", pkg.GetAPrimaryQlClass())
	})

	t.Run("GetATopLevelType", func(t *testing.T) {
		pkg := &Package{TopLevelTypes: []string{"Class1", "Class2"}}
		assert.Equal(t, []string{"Class1", "Class2"}, pkg.GetATopLevelType())
	})

	t.Run("GetMetrics", func(t *testing.T) {
		pkg := &Package{Metrics: "complexity:5;depth:3"}
		assert.Equal(t, "complexity:5;depth:3", pkg.GetMetrics())
	})

	t.Run("GetURL", func(t *testing.T) {
		pkg := &Package{URL: "http://test.com"}
		assert.Equal(t, "http://test.com", pkg.GetURL())
	})

	t.Run("Insert - Success", func(t *testing.T) {
		// Create an in-memory SQLite database for testing
		db, err := sql.Open("sqlite3", ":memory:")
		assert.NoError(t, err)
		defer db.Close()

		// Create the package table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS package (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				package_name TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(package_name)
			);
		`)
		assert.NoError(t, err)

		// Test successful insertion
		pkg := &Package{QualifiedName: "com.example.test"}
		err = pkg.Insert(db)
		assert.NoError(t, err)

		// Verify the insertion
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM package WHERE package_name = ?", pkg.QualifiedName).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Insert - Duplicate", func(t *testing.T) {
		// Create an in-memory SQLite database for testing
		db, err := sql.Open("sqlite3", ":memory:")
		assert.NoError(t, err)
		defer db.Close()

		// Create the package table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS package (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				package_name TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(package_name)
			);
		`)
		assert.NoError(t, err)

		// Insert package
		pkg := &Package{QualifiedName: "com.example.test"}
		err = pkg.Insert(db)
		assert.NoError(t, err)

		// Try to insert the same package again
		err = pkg.Insert(db)
		assert.Error(t, err) // Should fail due to UNIQUE constraint
	})

	t.Run("Insert - Error", func(t *testing.T) {
		// Create an in-memory SQLite database for testing
		db, err := sql.Open("sqlite3", ":memory:")
		assert.NoError(t, err)
		defer db.Close()

		// Create table with wrong schema to force an error
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS package (
				id INTEGER PRIMARY KEY AUTOINCREMENT
				-- missing required package_name column
			);
		`)
		assert.NoError(t, err)

		// Try to insert into incorrectly structured table
		pkg := &Package{QualifiedName: "test.package"}
		err = pkg.Insert(db)
		assert.Error(t, err) // Should fail due to missing column
	})
}
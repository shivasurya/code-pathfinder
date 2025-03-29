package java

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/stretchr/testify/assert"
)

// TestImportTypeStructure tests the structure and methods of the ImportType struct
func TestImportTypeStructure(t *testing.T) {
	// Test case 1: Basic import statement
	t.Run("Basic import statement", func(t *testing.T) {
		// Create an import type manually
		importType := &model.ImportType{
			ImportedType:      "java.util.List",
			SourceDeclaration: "Test.java",
		}

		// Verify the structure
		assert.Equal(t, "java.util.List", importType.ImportedType)
		assert.Equal(t, "Test.java", importType.SourceDeclaration)

		// Test the ToString method
		expected := "import java.util.List;"
		assert.Equal(t, expected, importType.ToString())

		// Test the GetImportedType method
		assert.Equal(t, "java.util.List", importType.GetImportedType())

		// Test the GetAPrimaryQlClass method
		assert.Equal(t, "ImportType", importType.GetAPrimaryQlClass())
	})

	// Test case 2: Import with nested package
	t.Run("Import with nested package", func(t *testing.T) {
		// Create an import type manually
		importType := &model.ImportType{
			ImportedType:      "com.example.project.util.Helper",
			SourceDeclaration: "Test.java",
		}

		// Verify the structure
		assert.Equal(t, "com.example.project.util.Helper", importType.ImportedType)

		// Test the ToString method
		expected := "import com.example.project.util.Helper;"
		assert.Equal(t, expected, importType.ToString())
	})

	// Test case 3: Using the constructor function
	t.Run("Using constructor function", func(t *testing.T) {
		// Create an import type using the constructor
		importType := model.NewImportType("java.io.File", "Test.java")

		// Verify the structure
		assert.Equal(t, "java.io.File", importType.ImportedType)
		assert.Equal(t, "Test.java", importType.SourceDeclaration)

		// Test the ToString method
		expected := "import java.io.File;"
		assert.Equal(t, expected, importType.ToString())
	})
}

// TestPackageStructure tests the structure and methods of the Package struct
func TestPackageStructure(t *testing.T) {
	// Test case 1: Basic package declaration
	t.Run("Basic package declaration", func(t *testing.T) {
		// Create a package manually
		pkg := &model.Package{
			QualifiedName: "com.example",
			TopLevelTypes: []string{"Main", "Helper"},
			FromSource:    true,
			Metrics:       "metrics-data",
			URL:           "http://example.com/package",
		}

		// Verify the structure
		assert.Equal(t, "com.example", pkg.QualifiedName)
		assert.Equal(t, []string{"Main", "Helper"}, pkg.TopLevelTypes)
		assert.True(t, pkg.FromSource)
		assert.Equal(t, "metrics-data", pkg.Metrics)
		assert.Equal(t, "http://example.com/package", pkg.URL)

		// Test the GetFromSource method
		assert.True(t, pkg.GetFromSource())

		// Test the GetAPrimaryQlClass method
		assert.Equal(t, "Package", pkg.GetAPrimaryQlClass())

		// Test the GetATopLevelType method
		assert.Equal(t, []string{"Main", "Helper"}, pkg.GetATopLevelType())

		// Test the GetMetrics method
		assert.Equal(t, "metrics-data", pkg.GetMetrics())

		// Test the GetURL method
		assert.Equal(t, "http://example.com/package", pkg.GetURL())
	})

	// Test case 2: Package with no top-level types
	t.Run("Package with no top-level types", func(t *testing.T) {
		// Create a package manually
		pkg := &model.Package{
			QualifiedName: "com.example.empty",
			TopLevelTypes: []string{},
			FromSource:    false,
		}

		// Verify the structure
		assert.Equal(t, "com.example.empty", pkg.QualifiedName)
		assert.Empty(t, pkg.TopLevelTypes)
		assert.False(t, pkg.FromSource)

		// Test the GetFromSource method
		assert.False(t, pkg.GetFromSource())
	})

	// Test case 3: Using the constructor function
	t.Run("Using constructor function", func(t *testing.T) {
		// Create a package using the constructor
		pkg := model.NewPackage(
			"org.test",
			[]string{"Test", "TestHelper"},
			true,
			"test-metrics",
			"http://test.org",
		)

		// Verify the structure
		assert.Equal(t, "org.test", pkg.QualifiedName)
		assert.Equal(t, []string{"Test", "TestHelper"}, pkg.TopLevelTypes)
		assert.True(t, pkg.FromSource)
		assert.Equal(t, "test-metrics", pkg.Metrics)
		assert.Equal(t, "http://test.org", pkg.URL)
	})
}

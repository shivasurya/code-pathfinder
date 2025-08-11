package db

import (
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
)

func TestNewStorageNode(t *testing.T) {
	// Create a temporary directory for the database
	tempDir := t.TempDir()

	// Initialize the StorageNode
	storageNode := NewStorageNode(tempDir)

	// Check if the database file was created
	dbPath := tempDir + "/pathfinder.db" // Updated to match the actual filename used in NewStorageNode
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("Database file was not created: %v", err)
	}

	// Check if the StorageNode is initialized correctly
	if storageNode.DB == nil {
		t.Fatal("StorageNode DB is not initialized")
	}

	// Close the database connection
	if err := storageNode.DB.Close(); err != nil {
		t.Fatalf("Failed to close database connection: %v", err)
	}
}

func TestAddAndGetPackages(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock package
	mockPackage := &model.Package{QualifiedName: "test.package"}

	// Add the package
	storageNode.AddPackage(mockPackage)

	// Retrieve the packages
	packages := storageNode.GetPackages()

	// Verify the package was added
	if len(packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(packages))
	}

	if packages[0].QualifiedName != "test.package" {
		t.Fatalf("Expected package name 'test.package', got '%s'", packages[0].QualifiedName)
	}
}

func TestAddAndGetImportDecls(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock import declaration
	mockImport := &model.ImportType{ImportedType: "fmt"}

	// Add the import declaration
	storageNode.AddImportDecl(mockImport)

	// Retrieve the import declarations
	importDecls := storageNode.GetImportDecls()

	// Verify the import declaration was added
	if len(importDecls) != 1 {
		t.Fatalf("Expected 1 import declaration, got %d", len(importDecls))
	}

	if importDecls[0].ImportedType != "fmt" {
		t.Fatalf("Expected import name 'fmt', got '%s'", importDecls[0].ImportedType)
	}
}

func TestAddAndGetClassDecls(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock class declaration
	mockClass := &model.Class{ClassOrInterface: model.ClassOrInterface{RefType: model.RefType{QualifiedName: "TestClass"}}}

	// Add the class declaration
	storageNode.AddClassDecl(mockClass)

	// Retrieve the class declarations
	classDecls := storageNode.GetClassDecls()

	// Verify the class declaration was added
	if len(classDecls) != 1 {
		t.Fatalf("Expected 1 class declaration, got %d", len(classDecls))
	}

	if classDecls[0].QualifiedName != "TestClass" {
		t.Fatalf("Expected class name 'TestClass', got '%s'", classDecls[0].QualifiedName)
	}
}

func TestAddAndGetMethodDecls(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock method declaration
	mockMethod := &model.Method{Name: "TestMethod"}

	// Add the method declaration
	storageNode.AddMethodDecl(mockMethod)

	// Retrieve the method declarations
	methodDecls := storageNode.GetMethodDecls()

	// Verify the method declaration was added
	if len(methodDecls) != 1 {
		t.Fatalf("Expected 1 method declaration, got %d", len(methodDecls))
	}

	if methodDecls[0].Name != "TestMethod" {
		t.Fatalf("Expected method name 'TestMethod', got '%s'", methodDecls[0].Name)
	}
}

func TestAddAndGetAnnotations(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock annotation
	mockAnnotation := &model.Annotation{QualifiedName: "TestAnnotation"}

	// Add the annotation
	storageNode.AddAnnotation(mockAnnotation)

	// Retrieve the annotations
	annotations := storageNode.GetAnnotations()

	// Verify the annotation was added
	if len(annotations) != 1 {
		t.Fatalf("Expected 1 annotation, got %d", len(annotations))
	}

	if annotations[0].QualifiedName != "TestAnnotation" {
		t.Fatalf("Expected annotation name 'TestAnnotation', got '%s'", annotations[0].QualifiedName)
	}
}

func TestAddAndGetBinaryExprs(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock binary expression
	mockBinaryExpr := &model.BinaryExpr{Op: "+"}

	// Add the binary expression
	storageNode.AddBinaryExpr(mockBinaryExpr)

	// Retrieve the binary expressions
	binaryExprs := storageNode.GetBinaryExprs()

	// Verify the binary expression was added
	if len(binaryExprs) != 1 {
		t.Fatalf("Expected 1 binary expression, got %d", len(binaryExprs))
	}

	if binaryExprs[0].Op != "+" {
		t.Fatalf("Expected operator '+', got '%s'", binaryExprs[0].Op)
	}
}

func TestAddAndGetMethodCalls(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock method call
	mockMethodCall := &model.MethodCall{
		MethodName:      "testMethod",
		QualifiedMethod: "com.example.TestClass.testMethod",
		Arguments:       []string{"arg1", "arg2"},
		TypeArguments:   []string{"String", "Integer"},
	}

	// Add the method call
	storageNode.AddMethodCall(mockMethodCall)

	// Retrieve the method calls
	methodCalls := storageNode.GetMethodCalls()

	// Verify the method call was added
	if len(methodCalls) != 1 {
		t.Fatalf("Expected 1 method call, got %d", len(methodCalls))
	}

	if methodCalls[0].MethodName != "testMethod" {
		t.Fatalf("Expected method name 'testMethod', got '%s'", methodCalls[0].MethodName)
	}

	if methodCalls[0].QualifiedMethod != "com.example.TestClass.testMethod" {
		t.Fatalf("Expected qualified name 'com.example.TestClass.testMethod', got '%s'", methodCalls[0].QualifiedMethod)
	}
}

func TestAddAndGetFields(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock field declaration
	mockField := &model.FieldDeclaration{
		Type:       "String",
		FieldNames: []string{"testField"},
		Visibility: "private",
		IsStatic:   true,
		IsFinal:    true,
	}

	// Add the field declaration
	storageNode.AddFieldDecl(mockField)

	// Retrieve the fields
	fields := storageNode.GetFields()

	// Verify the field was added
	if len(fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(fields))
	}

	if fields[0].Type != "String" {
		t.Fatalf("Expected field type 'String', got '%s'", fields[0].Type)
	}

	if !fields[0].IsStatic {
		t.Fatal("Expected field to be static")
	}
}

func TestAddAndGetJavaDocs(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock JavaDoc
	mockJavaDoc := &model.Javadoc{
		CommentedCodeElements: "/** Test documentation */",
	}

	// Add the JavaDoc
	storageNode.AddJavaDoc(mockJavaDoc)

	// Retrieve the JavaDocs
	javaDocs := storageNode.GetJavaDocs()

	// Verify the JavaDoc was added
	if len(javaDocs) != 1 {
		t.Fatalf("Expected 1 JavaDoc, got %d", len(javaDocs))
	}

	if javaDocs[0].CommentedCodeElements != "/** Test documentation */" {
		t.Fatalf("Expected JavaDoc content '/** Test documentation */', got '%s'", javaDocs[0].CommentedCodeElements)
	}
}

func TestDuplicatePackageHandling(t *testing.T) {
	storageNode := NewStorageNode("")

	// Create a mock package
	mockPackage := &model.Package{QualifiedName: "test.package"}

	// Add the same package twice
	storageNode.AddPackage(mockPackage)
	storageNode.AddPackage(mockPackage)

	// Retrieve the packages
	packages := storageNode.GetPackages()

	// Verify only one package was added
	if len(packages) != 1 {
		t.Fatalf("Expected 1 package after duplicate addition, got %d", len(packages))
	}

	if packages[0].QualifiedName != "test.package" {
		t.Fatalf("Expected package name 'test.package', got '%s'", packages[0].QualifiedName)
	}
}

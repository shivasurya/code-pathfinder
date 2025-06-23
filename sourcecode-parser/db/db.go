package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3" // required for sqlite3

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
)

type StorageNode struct {
	DB                     *sql.DB
	Package                []*model.Package
	ImportDecl             []*model.ImportType
	Annotation             []*model.Annotation
	AddExpr                []*model.AddExpr
	AndLogicalExpr         []*model.AndLogicalExpr
	AssertStmt             []*model.AssertStmt
	BinaryExpr             []*model.BinaryExpr
	AndBitwiseExpr         []*model.AndBitwiseExpr
	BlockStmt              []*model.BlockStmt
	BreakStmt              []*model.BreakStmt
	ClassDecl              []*model.Class
	ClassInstanceExpr      []*model.ClassInstanceExpr
	ComparisonExpr         []*model.ComparisonExpr
	ContinueStmt           []*model.ContinueStmt
	DivExpr                []*model.DivExpr
	DoStmt                 []*model.DoStmt
	EQExpr                 []*model.EqExpr
	Field                  []*model.FieldDeclaration
	FileNode               []*model.File
	ForStmt                []*model.ForStmt
	IfStmt                 []*model.IfStmt
	JavaDoc                []*model.Javadoc
	LeftShiftExpr          []*model.LeftShiftExpr
	MethodDecl             []*model.Method
	MethodCall             []*model.MethodCall
	MulExpr                []*model.MulExpr
	NEExpr                 []*model.NEExpr
	OrLogicalExpr          []*model.OrLogicalExpr
	RightShiftExpr         []*model.RightShiftExpr
	RemExpr                []*model.RemExpr
	ReturnStmt             []*model.ReturnStmt
	SubExpr                []*model.SubExpr
	UnsignedRightShiftExpr []*model.UnsignedRightShiftExpr
	WhileStmt              []*model.WhileStmt
	XorBitwiseExpr         []*model.XorBitwiseExpr
	YieldStmt              []*model.YieldStmt
}

const (
	createTablePackage = `
	CREATE TABLE IF NOT EXISTS package (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		package_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(package_name)
	);`

	createTableImportDecl = `
	CREATE TABLE IF NOT EXISTS import_decl (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		import_type TEXT NOT NULL,
		import_name TEXT NOT NULL,
		file_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(import_type, import_name, file_path)
	);`

	createTableAnnotation = `
	CREATE TABLE IF NOT EXISTS annotation (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		annotation_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(annotation_name)
	);`

	createTableClassDecl = `
	CREATE TABLE IF NOT EXISTS class_decl (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_name TEXT NOT NULL,
		package_name TEXT NOT NULL,
		source_declaration TEXT,
		super_types TEXT,
		annotations TEXT,
		modifiers TEXT,
		is_top_level BOOLEAN NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (package_name) REFERENCES package(package_name)
	);`

	createTableMethodDecl = `
	CREATE TABLE IF NOT EXISTS method_decl (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		qualified_name TEXT NOT NULL,
		return_type TEXT NOT NULL,
		parameters TEXT,
		parameter_names TEXT,
		visibility TEXT NOT NULL,
		is_abstract BOOLEAN NOT NULL,
		is_strictfp BOOLEAN NOT NULL,
		is_static BOOLEAN NOT NULL,
		is_final BOOLEAN NOT NULL,
		is_constructor BOOLEAN NOT NULL,
		source_declaration TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createTableMethodCall = `
		CREATE TABLE IF NOT EXISTS method_call (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			method_name TEXT NOT NULL,
			qualified_name TEXT NOT NULL,
			parameters TEXT,
			parameters_names TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createTableFieldDecl = `
		CREATE TABLE IF NOT EXISTS field_decl (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		field_name TEXT NOT NULL,
		type TEXT NOT NULL,
		visibility TEXT NOT NULL,
		is_static BOOLEAN NOT NULL,
		is_final BOOLEAN NOT NULL,
		is_transient BOOLEAN NOT NULL,
		is_volatile BOOLEAN NOT NULL,
		source_declaration TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createTableLocalVariableDecl = `
		CREATE TABLE IF NOT EXISTS local_variable_decl (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		local_variable_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(local_variable_name)
	);`

	createTableBinaryExpr = `
		CREATE TABLE IF NOT EXISTS binary_expr (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		left_operand TEXT NOT NULL,
		right_operand TEXT NOT NULL,
		operator TEXT NOT NULL,
		source_declaration TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createTableJavadoc = `
	CREATE TABLE IF NOT EXISTS javadoc (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		javadoc_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(javadoc_name)
	);`

	createTableEntity = `
	CREATE TABLE IF NOT EXISTS entity (
		id INTEGER PRIMARY KEY,
		name TEXT UNIQUE
	);`
)

func NewStorageNode(databasePath string) *StorageNode {
	dbName := "pathfinder.db"
	if databasePath != "" {
		databasePath = databasePath + "/" + dbName
	}
	database, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		log.Fatal(err)
	}

	// create table if not exist
	if _, err := database.Exec(createTablePackage); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableImportDecl); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableAnnotation); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableClassDecl); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableMethodDecl); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableMethodCall); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableFieldDecl); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableLocalVariableDecl); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableBinaryExpr); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableJavadoc); err != nil {
		log.Fatal(err)
	}
	if _, err := database.Exec(createTableEntity); err != nil {
		log.Fatal(err)
	}

	return &StorageNode{DB: database}
}

func (s *StorageNode) AddPackage(node *model.Package) {
	// Check if the package already exists
	for _, existingPackage := range s.Package {
		if existingPackage.QualifiedName == node.QualifiedName {
			return
		}
	}
	s.Package = append(s.Package, node)
}

func (s *StorageNode) GetPackages() []*model.Package {
	return s.Package
}

func (s *StorageNode) GetImportDecls() []*model.ImportType {
	return s.ImportDecl
}

func (s *StorageNode) AddImportDecl(node *model.ImportType) {
	s.ImportDecl = append(s.ImportDecl, node)
}

func (s *StorageNode) AddClassDecl(node *model.Class) {
	s.ClassDecl = append(s.ClassDecl, node)
}

func (s *StorageNode) GetClassDecls() []*model.Class {
	return s.ClassDecl
}

func (s *StorageNode) AddMethodDecl(node *model.Method) {
	s.MethodDecl = append(s.MethodDecl, node)
}

func (s *StorageNode) GetMethodDecls() []*model.Method {
	return s.MethodDecl
}

func (s *StorageNode) AddMethodCall(node *model.MethodCall) {
	s.MethodCall = append(s.MethodCall, node)
}

func (s *StorageNode) GetMethodCalls() []*model.MethodCall {
	return s.MethodCall
}

func (s *StorageNode) AddFieldDecl(node *model.FieldDeclaration) {
	s.Field = append(s.Field, node)
}

func (s *StorageNode) GetFields() []*model.FieldDeclaration {
	return s.Field
}

func (s *StorageNode) AddBinaryExpr(node *model.BinaryExpr) {
	s.BinaryExpr = append(s.BinaryExpr, node)
}

func (s *StorageNode) GetBinaryExprs() []*model.BinaryExpr {
	return s.BinaryExpr
}

func (s *StorageNode) AddAnnotation(node *model.Annotation) {
	s.Annotation = append(s.Annotation, node)
}

func (s *StorageNode) GetAnnotations() []*model.Annotation {
	return s.Annotation
}

func (s *StorageNode) AddJavaDoc(node *model.Javadoc) {
	s.JavaDoc = append(s.JavaDoc, node)
}

func (s *StorageNode) GetJavaDocs() []*model.Javadoc {
	return s.JavaDoc
}

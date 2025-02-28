package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"

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
	CREATE_TABLE_PACKAGE = `
	CREATE TABLE IF NOT EXISTS package (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		package_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(package_name)
	);`

	CREATE_TABLE_IMPORT_DECL = `
	CREATE TABLE IF NOT EXISTS import_decl (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		import_type TEXT NOT NULL,
		import_name TEXT NOT NULL,
		file_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(import_type, import_name, file_path)
	);`

	CREATE_TABLE_ANNOTATION = `
	CREATE TABLE IF NOT EXISTS annotation (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		annotation_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(annotation_name)
	);`

	CREATE_TABLE_CLASS_DECL = `
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

	CREATE_TABLE_METHOD_DECL = `
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

	CREATE_TABLE_METHOD_CALL = `
		CREATE TABLE IF NOT EXISTS method_call (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			method_name TEXT NOT NULL,
			qualified_name TEXT NOT NULL,
			parameters TEXT,
			parameters_names TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	CREATE_TABLE_FIELD_DECL = `
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

	CREATE_TABLE_LOCAL_VARIABLE_DECL = `
		CREATE TABLE IF NOT EXISTS local_variable_decl (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		local_variable_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(local_variable_name)
	);`

	CREATE_TABLE_BINARY_EXPR = `
		CREATE TABLE IF NOT EXISTS binary_expr (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		binary_expr_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(binary_expr_name)
	);`

	CREATE_TABLE_JAVADOC = `
	CREATE TABLE IF NOT EXISTS javadoc (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		javadoc_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(javadoc_name)
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
	database.Exec(CREATE_TABLE_PACKAGE)
	database.Exec(CREATE_TABLE_IMPORT_DECL)
	database.Exec(CREATE_TABLE_ANNOTATION)
	database.Exec(CREATE_TABLE_CLASS_DECL)
	database.Exec(CREATE_TABLE_METHOD_DECL)
	database.Exec(CREATE_TABLE_METHOD_CALL)
	database.Exec(CREATE_TABLE_FIELD_DECL)
	database.Exec(CREATE_TABLE_LOCAL_VARIABLE_DECL)
	database.Exec(CREATE_TABLE_BINARY_EXPR)
	database.Exec(CREATE_TABLE_JAVADOC)

	return &StorageNode{DB: database}
}

func (s *StorageNode) AddPackage(node *model.Package) {
	s.Package = append(s.Package, node)
}

func (s *StorageNode) AddImportDecl(node *model.ImportType) {
	s.ImportDecl = append(s.ImportDecl, node)
}

func (s *StorageNode) AddClassDecl(node *model.Class) {
	s.ClassDecl = append(s.ClassDecl, node)
}

func (s *StorageNode) AddMethodDecl(node *model.Method) {
	s.MethodDecl = append(s.MethodDecl, node)
}

func (s *StorageNode) AddMethodCall(node *model.MethodCall) {
	s.MethodCall = append(s.MethodCall, node)
}

func (s *StorageNode) AddFieldDecl(node *model.FieldDeclaration) {
	s.Field = append(s.Field, node)
}

func (s *StorageNode) AddBinaryExpr(node *model.BinaryExpr) {
	s.BinaryExpr = append(s.BinaryExpr, node)
}

func (s *StorageNode) AddAnnotation(node *model.Annotation) {
	s.Annotation = append(s.Annotation, node)
}

func (s *StorageNode) AddJavaDoc(node *model.Javadoc) {
	s.JavaDoc = append(s.JavaDoc, node)
}

func (s *StorageNode) GetEntity(entityType string) interface{} {
	switch entityType {
	case "package":
		return s.Package
	case "import":
		return s.ImportDecl
	case "annotation":
		return s.Annotation
	case "add_expr":
		return s.AddExpr
	case "and_logical_expr":
		return s.AndLogicalExpr
	case "assert_stmt":
		return s.AssertStmt
	case "binary_expr":
		return s.BinaryExpr
	case "and_bitwise_expr":
		return s.AndBitwiseExpr
	case "block_stmt":
		return s.BlockStmt
	case "break_stmt":
		return s.BreakStmt
	case "class":
		return s.ClassDecl
	case "class_instance_expr":
		return s.ClassInstanceExpr
	case "comparison_expr":
		return s.ComparisonExpr
	case "continue_stmt":
		return s.ContinueStmt
	case "div_expr":
		return s.DivExpr
	case "do_stmt":
		return s.DoStmt
	case "eq_expr":
		return s.EQExpr
	case "field":
		return s.Field
	case "file":
		return s.FileNode
	case "for_stmt":
		return s.ForStmt
	case "if_stmt":
		return s.IfStmt
	case "javadoc":
		return s.JavaDoc
	case "left_shift_expr":
		return s.LeftShiftExpr
	case "method":
		return s.MethodDecl
	case "method_call":
		return s.MethodCall
	case "mul_expr":
		return s.MulExpr
	case "ne_expr":
		return s.NEExpr
	case "or_logical_expr":
		return s.OrLogicalExpr
	case "right_shift_expr":
		return s.RightShiftExpr
	case "rem_expr":
		return s.RemExpr
	case "return_stmt":
		return s.ReturnStmt
	case "sub_expr":
		return s.SubExpr
	case "unsigned_right_shift_expr":
		return s.UnsignedRightShiftExpr
	case "while_stmt":
		return s.WhileStmt
	case "xor_bitwise_expr":
		return s.XorBitwiseExpr
	case "yield_stmt":
		return s.YieldStmt
	default:
		return nil
	}
}

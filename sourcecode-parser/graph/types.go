package graph

import "github.com/shivasurya/code-pathfinder/sourcecode-parser/model"

// Node represents a node in the code graph with various properties
// describing code elements like classes, methods, variables, etc.
type Node struct {
	ID                   string
	Type                 string
	Name                 string
	CodeSnippet          string
	LineNumber           uint32
	OutgoingEdges        []*Edge
	IsExternal           bool
	Modifier             string
	ReturnType           string
	MethodArgumentsType  []string
	MethodArgumentsValue []string
	PackageName          string
	ImportPackage        []string
	SuperClass           string
	Interface            []string
	DataType             string
	Scope                string
	VariableValue        string
	hasAccess            bool
	File                 string
	isJavaSourceFile     bool
	isPythonSourceFile   bool
	ThrowsExceptions     []string
	Annotation           []string
	JavaDoc              *model.Javadoc
	BinaryExpr           *model.BinaryExpr
	ClassInstanceExpr    *model.ClassInstanceExpr
	IfStmt               *model.IfStmt
	WhileStmt            *model.WhileStmt
	DoStmt               *model.DoStmt
	ForStmt              *model.ForStmt
	BreakStmt            *model.BreakStmt
	ContinueStmt         *model.ContinueStmt
	YieldStmt            *model.YieldStmt
	AssertStmt           *model.AssertStmt
	ReturnStmt           *model.ReturnStmt
	BlockStmt            *model.BlockStmt
}

// Edge represents a directed edge between two nodes in the code graph.
type Edge struct {
	From *Node
	To   *Node
}

// CodeGraph represents the entire code graph with nodes and edges.
type CodeGraph struct {
	Nodes map[string]*Node
	Edges []*Edge
}

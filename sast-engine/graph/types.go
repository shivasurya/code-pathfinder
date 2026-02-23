package graph

import "github.com/shivasurya/code-pathfinder/sast-engine/model"

// SourceLocation stores the file location of a code snippet for lazy loading.
type SourceLocation struct {
	File      string
	StartByte uint32
	EndByte   uint32
}

// Node represents a node in the code graph with various properties
// describing code elements like classes, methods, variables, etc.
type Node struct {
	ID                   string
	Type                 string
	Name                 string
	CodeSnippet          string // DEPRECATED: Will be removed, use GetCodeSnippet() instead
	SourceLocation       *SourceLocation
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
	isGoSourceFile       bool
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
	Language             string         // "go", "python", "java" - set during parsing
	Metadata             map[string]any // Generic key-value store for language/tool-specific metadata
}

// GetCodeSnippet returns the code snippet for this node.
// If SourceLocation is set, it reads from the file (lazy loading).
// Otherwise, it returns the deprecated CodeSnippet field for backward compatibility.
func (n *Node) GetCodeSnippet() string {
	// If we have a source location, read from file (lazy load)
	if n.SourceLocation != nil {
		content, err := readFile(n.SourceLocation.File)
		if err != nil {
			// Fallback to CodeSnippet if file read fails
			return n.CodeSnippet
		}
		// Extract the specific range
		if n.SourceLocation.EndByte <= uint32(len(content)) {
			return string(content[n.SourceLocation.StartByte:n.SourceLocation.EndByte])
		}
	}
	// Fallback to deprecated CodeSnippet field
	return n.CodeSnippet
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

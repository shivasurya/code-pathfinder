package model

type Node struct {
	ID                   string
	Type                 string
	Name                 string
	CodeSnippet          string
	LineNumber           uint32
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
	File                 string
	IsJavaSourceFile     bool
	ThrowsExceptions     []string
	Annotation           []string
	JavaDoc              *Javadoc
	BinaryExpr           *BinaryExpr
	ClassInstanceExpr    *ClassInstanceExpr
	IfStmt               *IfStmt
	WhileStmt            *WhileStmt
	DoStmt               *DoStmt
	ForStmt              *ForStmt
	BreakStmt            *BreakStmt
	ContinueStmt         *ContinueStmt
	YieldStmt            *YieldStmt
	AssertStmt           *AssertStmt
	ReturnStmt           *ReturnStmt
	BlockStmt            *BlockStmt
	FileNode             *File
}

type TreeNode struct {
	Node     *Node
	Children []*TreeNode
	Parent   *TreeNode
}

func (t *TreeNode) AddChild(child *TreeNode) {
	t.Children = append(t.Children, child)
}

func (t *TreeNode) AddChildren(children []*TreeNode) {
	t.Children = append(t.Children, children...)
}

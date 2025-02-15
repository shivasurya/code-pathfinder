package model

type Node struct {
	AssertStmt        *AssertStmt
	BinaryExpr        *BinaryExpr
	BlockStmt         *BlockStmt
	BreakStmt         *BreakStmt
	ClassDecl         *Class
	ClassInstanceExpr *ClassInstanceExpr
	ContinueStmt      *ContinueStmt
	DoStmt            *DoStmt
	Field             *FieldDeclaration
	FileNode          *File
	ForStmt           *ForStmt
	IfStmt            *IfStmt
	JavaDoc           *Javadoc
	MethodDecl        *Method
	MethodCall        *MethodCall
	ReturnStmt        *ReturnStmt
	WhileStmt         *WhileStmt
	YieldStmt         *YieldStmt
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

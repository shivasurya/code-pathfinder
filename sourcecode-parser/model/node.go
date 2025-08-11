package model

type Node struct {
	NodeType               string
	NodeID                 int64
	AddExpr                *AddExpr
	AndLogicalExpr         *AndLogicalExpr
	AssertStmt             *AssertStmt
	BinaryExpr             *BinaryExpr
	AndBitwiseExpr         *AndBitwiseExpr
	BlockStmt              *BlockStmt
	BreakStmt              *BreakStmt
	ClassDecl              *Class
	ClassInstanceExpr      *ClassInstanceExpr
	ComparisonExpr         *ComparisonExpr
	ContinueStmt           *ContinueStmt
	DivExpr                *DivExpr
	DoStmt                 *DoStmt
	EQExpr                 *EqExpr
	Field                  *FieldDeclaration
	FileNode               *File
	ForStmt                *ForStmt
	IfStmt                 *IfStmt
	ImportType             *ImportType
	JavaDoc                *Javadoc
	LeftShiftExpr          *LeftShiftExpr
	MethodDecl             *Method
	MethodCall             *MethodCall
	MulExpr                *MulExpr
	NEExpr                 *NEExpr
	OrLogicalExpr          *OrLogicalExpr
	Package                *Package
	RightShiftExpr         *RightShiftExpr
	RemExpr                *RemExpr
	ReturnStmt             *ReturnStmt
	SubExpr                *SubExpr
	UnsignedRightShiftExpr *UnsignedRightShiftExpr
	WhileStmt              *WhileStmt
	XorBitwiseExpr         *XorBitwiseExpr
	YieldStmt              *YieldStmt
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

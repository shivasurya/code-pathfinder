package model

type StmtParent struct {
	Top
}

type IStmt interface {
	GetAChildStmt() *Stmt
	GetBasicBlock() *BasicBlock
	GetCompilationUnit() *CompilationUnit
	GetControlFlowNode() *ControlFlowNode
	GetEnclosingCallable() *Callable
	GetEnclosingStmt() *Stmt
	GetHalsteadID() int
	GetIndex() int
	GetMetrics() interface{}
	GetParent() *Stmt
	IsNthChildOf(parent *Stmt, n int) bool
	Pp() string
	ToString() string
}

type Stmt struct {
	StmtParent
}

type IConditionalStmt interface {
	GetCondition() *Expr
}

type ConditionalStmt struct {
	Stmt
	Condition *Expr
}

type IIfStmt interface {
	GetCondition() *Expr
	GetElse() *Stmt
	GetThen() *Stmt
	GetHalsteadID() int
	GetAPrimaryQlClass() string
	GetPP() string
	ToString() string
}

type IfStmt struct {
	ConditionalStmt
}

type DoStmt struct {
	ConditionalStmt
}

type IDoStmt interface {
	GetAPrimaryQlClass() string
	GetCondition() *Expr
	GetHalsteadID() int
	GetStmt() *Stmt
	GetPP() string
	ToString() string
}

type IForStmt interface {
	GetPrimaryQlClass() string
	GetAnInit() *Expr
	GetAnIterationVariable() *Expr
	GetAnUpdate() *Expr
	GetCondition() *Expr
	GetHalsteadID() int
	GetInit(int) *Expr
	GetStmt() *Stmt
	GetUpdate(int) *Expr
	GetPP() string
	ToString() string
}

type ForStmt struct {
	ConditionalStmt
}

type IWhileStmt interface {
	GetAPrimaryQlClass() string
	GetCondition() *Expr
	GetHalsteadID() int
	GetStmt() *Stmt
	GetPP() string
	ToString() string
}

type WhileStmt struct {
	ConditionalStmt
}

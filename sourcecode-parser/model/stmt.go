package model

import "fmt"

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
	NodeString string
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
	Else Stmt
	Then Stmt
}

func (ifStmt *IfStmt) GetCondition() *Expr {
	return ifStmt.Condition
}

func (ifStmt *IfStmt) GetElse() *Stmt {
	return &ifStmt.Else
}

func (ifStmt *IfStmt) GetThen() *Stmt {
	return &ifStmt.Then
}

func (ifStmt *IfStmt) GetAPrimaryQlClass() string {
	return "ifStmt"
}

func (ifStmt *IfStmt) GetPP() string {
	return fmt.Sprintf("if (%s) %s else %s", ifStmt.Condition.NodeString, ifStmt.Then.NodeString, ifStmt.Else.NodeString)
}

func (ifStmt *IfStmt) ToString() string {
	return fmt.Sprintf("if (%s) %s else %s", ifStmt.Condition.NodeString, ifStmt.Then.NodeString, ifStmt.Else.NodeString)
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
	GetStmt() Stmt
	GetPP() string
	ToString() string
}

type WhileStmt struct {
	ConditionalStmt
}

func (whileStmt *WhileStmt) GetAPrimaryQlClass() string {
	return "whileStmt"
}

func (whileStmt *WhileStmt) GetCondition() *Expr {
	return whileStmt.Condition
}

func (whileStmt *WhileStmt) GetHalsteadID() int {
	return 0
}

func (whileStmt *WhileStmt) GetStmt() Stmt {
	return whileStmt.Stmt
}

func (whileStmt *WhileStmt) GetPP() string {
	return ""
}

func (whileStmt *WhileStmt) ToString() string {
	return ""
}

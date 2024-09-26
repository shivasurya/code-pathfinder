package model

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

type ExprParent struct{}

func (e *ExprParent) GetAChildExpr() *Expr {
	return nil
}

func (e *ExprParent) GetChildExpr(_ int) *Expr {
	return nil
}

func (e *ExprParent) GetNumChildExpr() int64 {
	return 0
}

type Expr struct {
	ExprParent
	Kind       int
	Node       sitter.Node
	NodeString string
	Type       string
}

func (e *Expr) String() string {
	return fmt.Sprintf("Expr(%s)", e.NodeString)
}

func (e *Expr) GetAChildExpr() *Expr {
	return e
}

func (e *Expr) GetChildExpr(_ int) *Expr {
	return e
}

func (e *Expr) GetNumChildExpr() int64 {
	return 1
}

func (e *Expr) GetBoolValue() {
}

func (e *Expr) GetKind() int {
	return e.Kind
}

type BinaryExpr struct {
	Expr
	Op           string
	LeftOperand  *Expr
	RightOperand *Expr
}

func (e *BinaryExpr) GetLeftOperand() *Expr {
	return e.LeftOperand
}

func (e *BinaryExpr) GetLeftOperandString() string {
	return e.LeftOperand.NodeString
}

func (e *BinaryExpr) GetRightOperand() *Expr {
	return e.RightOperand
}

func (e *BinaryExpr) GetRightOperandString() string {
	return e.RightOperand.NodeString
}

func (e *BinaryExpr) GetOp() string {
	return e.Op
}

func (e *BinaryExpr) GetKind() int {
	return 1
}

func (e *BinaryExpr) ToString() string {
	return fmt.Sprintf("BinaryExpr(%s, %v, %v)", e.Op, e.LeftOperand, e.RightOperand)
}

func (e *BinaryExpr) GetAnOperand() *Expr {
	if e.LeftOperand != nil {
		return e.LeftOperand
	}
	return e.RightOperand
}

func (e *BinaryExpr) HasOperands(expr1, expr2 *Expr) bool {
	return e.LeftOperand == expr1 && e.RightOperand == expr2
}

type AddExpr struct {
	BinaryExpr
	op string
}

func (e *AddExpr) GetOp() string {
	return e.op
}

type AndBitwiseExpr struct {
	BinaryExpr
	op string
}

func (e *AndBitwiseExpr) GetOp() string {
	return e.op
}

type ComparisonExpr struct {
	BinaryExpr
}

func (e *ComparisonExpr) GetGreaterThanOperand() *Expr {
	return nil
}

func (e *ComparisonExpr) GetLessThanOperand() *Expr {
	return nil
}

func (e *ComparisonExpr) IsStrict() bool {
	return true
}

type AndLogicalExpr struct {
	BinaryExpr
	op string
}

func (e *AndLogicalExpr) GetOp() string {
	return e.op
}

type DivExpr struct {
	BinaryExpr
	op string
}

func (e *DivExpr) GetOp() string {
	return e.op
}

type EqExpr struct {
	BinaryExpr
	op string
}

func (e *EqExpr) GetOp() string {
	return e.op
}

type GEExpr struct {
	BinaryExpr
	op string
}

func (e *GEExpr) GetOp() string {
	return e.op
}

type GTExpr struct {
	BinaryExpr
	op string
}

func (e *GTExpr) GetOp() string {
	return e.op
}

type LEExpr struct {
	BinaryExpr
	op string
}

func (e *LEExpr) GetOp() string {
	return e.op
}

type LTExpr struct {
	BinaryExpr
	op string
}

func (e *LTExpr) GetOp() string {
	return e.op
}

type NEExpr struct {
	BinaryExpr
	op string
}

func (e *NEExpr) GetOp() string {
	return e.op
}

type LeftShiftExpr struct {
	BinaryExpr
	op string
}

func (e *LeftShiftExpr) GetOp() string {
	return e.op
}

type MulExpr struct {
	BinaryExpr
	op string
}

func (e *MulExpr) GetOp() string {
	return e.op
}

type OrBitwiseExpr struct {
	BinaryExpr
	op string
}

func (e *OrBitwiseExpr) GetOp() string {
	return e.op
}

type OrLogicalExpr struct {
	BinaryExpr
	op string
}

func (e *OrLogicalExpr) GetOp() string {
	return e.op
}

type RemExpr struct {
	BinaryExpr
	op string
}

func (e *RemExpr) GetOp() string {
	return e.op
}

type RightShiftExpr struct {
	BinaryExpr
	op string
}

func (e *RightShiftExpr) GetOp() string {
	return e.op
}

type SubExpr struct {
	BinaryExpr
	op string
}

func (e *SubExpr) GetOp() string {
	return e.op
}

type UnsignedRightShiftExpr struct {
	BinaryExpr
	op string
}

func (e *UnsignedRightShiftExpr) GetOp() string {
	return e.op
}

type XorBitwiseExpr struct {
	BinaryExpr
	op string
}

func (e *XorBitwiseExpr) GetOp() string {
	return e.op
}

type ClassInstanceExpr struct {
	Expr
	ClassName string
	Args      []*Expr
}

func (e *ClassInstanceExpr) GetClassName() string {
	return e.ClassName
}

func (e *ClassInstanceExpr) GetArgs() []*Expr {
	return e.Args
}

func (e *ClassInstanceExpr) GetArg(i int) *Expr {
	return e.Args[i]
}

func (e *ClassInstanceExpr) GetNumArgs() int {
	return len(e.Args)
}

func (e *ClassInstanceExpr) String() string {
	return fmt.Sprintf("ClassInstanceExpr(%s, %v)", e.ClassName, e.Args)
}

// Code generated from Query.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // Query

import "github.com/antlr4-go/antlr/v4"

// BaseQueryListener is a complete listener for a parse tree produced by QueryParser.
type BaseQueryListener struct{}

var _ QueryListener = &BaseQueryListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseQueryListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseQueryListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseQueryListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseQueryListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterQuery is called when production query is entered.
func (s *BaseQueryListener) EnterQuery(ctx *QueryContext) {}

// ExitQuery is called when production query is exited.
func (s *BaseQueryListener) ExitQuery(ctx *QueryContext) {}

// EnterSelect_list is called when production select_list is entered.
func (s *BaseQueryListener) EnterSelect_list(ctx *Select_listContext) {}

// ExitSelect_list is called when production select_list is exited.
func (s *BaseQueryListener) ExitSelect_list(ctx *Select_listContext) {}

// EnterSelect_item is called when production select_item is entered.
func (s *BaseQueryListener) EnterSelect_item(ctx *Select_itemContext) {}

// ExitSelect_item is called when production select_item is exited.
func (s *BaseQueryListener) ExitSelect_item(ctx *Select_itemContext) {}

// EnterEntity is called when production entity is entered.
func (s *BaseQueryListener) EnterEntity(ctx *EntityContext) {}

// ExitEntity is called when production entity is exited.
func (s *BaseQueryListener) ExitEntity(ctx *EntityContext) {}

// EnterAlias is called when production alias is entered.
func (s *BaseQueryListener) EnterAlias(ctx *AliasContext) {}

// ExitAlias is called when production alias is exited.
func (s *BaseQueryListener) ExitAlias(ctx *AliasContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseQueryListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseQueryListener) ExitExpression(ctx *ExpressionContext) {}

// EnterOrExpression is called when production orExpression is entered.
func (s *BaseQueryListener) EnterOrExpression(ctx *OrExpressionContext) {}

// ExitOrExpression is called when production orExpression is exited.
func (s *BaseQueryListener) ExitOrExpression(ctx *OrExpressionContext) {}

// EnterAndExpression is called when production andExpression is entered.
func (s *BaseQueryListener) EnterAndExpression(ctx *AndExpressionContext) {}

// ExitAndExpression is called when production andExpression is exited.
func (s *BaseQueryListener) ExitAndExpression(ctx *AndExpressionContext) {}

// EnterNotExpression is called when production notExpression is entered.
func (s *BaseQueryListener) EnterNotExpression(ctx *NotExpressionContext) {}

// ExitNotExpression is called when production notExpression is exited.
func (s *BaseQueryListener) ExitNotExpression(ctx *NotExpressionContext) {}

// EnterPrimary is called when production primary is entered.
func (s *BaseQueryListener) EnterPrimary(ctx *PrimaryContext) {}

// ExitPrimary is called when production primary is exited.
func (s *BaseQueryListener) ExitPrimary(ctx *PrimaryContext) {}

// EnterCondition is called when production condition is entered.
func (s *BaseQueryListener) EnterCondition(ctx *ConditionContext) {}

// ExitCondition is called when production condition is exited.
func (s *BaseQueryListener) ExitCondition(ctx *ConditionContext) {}

// EnterMethod_chain is called when production method_chain is entered.
func (s *BaseQueryListener) EnterMethod_chain(ctx *Method_chainContext) {}

// ExitMethod_chain is called when production method_chain is exited.
func (s *BaseQueryListener) ExitMethod_chain(ctx *Method_chainContext) {}

// EnterMethod_or_variable is called when production method_or_variable is entered.
func (s *BaseQueryListener) EnterMethod_or_variable(ctx *Method_or_variableContext) {}

// ExitMethod_or_variable is called when production method_or_variable is exited.
func (s *BaseQueryListener) ExitMethod_or_variable(ctx *Method_or_variableContext) {}

// EnterMethod is called when production method is entered.
func (s *BaseQueryListener) EnterMethod(ctx *MethodContext) {}

// ExitMethod is called when production method is exited.
func (s *BaseQueryListener) ExitMethod(ctx *MethodContext) {}

// EnterVariable is called when production variable is entered.
func (s *BaseQueryListener) EnterVariable(ctx *VariableContext) {}

// ExitVariable is called when production variable is exited.
func (s *BaseQueryListener) ExitVariable(ctx *VariableContext) {}

// EnterComparator is called when production comparator is entered.
func (s *BaseQueryListener) EnterComparator(ctx *ComparatorContext) {}

// ExitComparator is called when production comparator is exited.
func (s *BaseQueryListener) ExitComparator(ctx *ComparatorContext) {}

// EnterValue is called when production value is entered.
func (s *BaseQueryListener) EnterValue(ctx *ValueContext) {}

// ExitValue is called when production value is exited.
func (s *BaseQueryListener) ExitValue(ctx *ValueContext) {}

// EnterValue_list is called when production value_list is entered.
func (s *BaseQueryListener) EnterValue_list(ctx *Value_listContext) {}

// ExitValue_list is called when production value_list is exited.
func (s *BaseQueryListener) ExitValue_list(ctx *Value_listContext) {}

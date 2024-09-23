// Code generated from Query.g4 by ANTLR 4.13.2. DO NOT EDIT.

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

// EnterPredicate_declarations is called when production predicate_declarations is entered.
func (s *BaseQueryListener) EnterPredicate_declarations(ctx *Predicate_declarationsContext) {}

// ExitPredicate_declarations is called when production predicate_declarations is exited.
func (s *BaseQueryListener) ExitPredicate_declarations(ctx *Predicate_declarationsContext) {}

// EnterPredicate_declaration is called when production predicate_declaration is entered.
func (s *BaseQueryListener) EnterPredicate_declaration(ctx *Predicate_declarationContext) {}

// ExitPredicate_declaration is called when production predicate_declaration is exited.
func (s *BaseQueryListener) ExitPredicate_declaration(ctx *Predicate_declarationContext) {}

// EnterPredicate_name is called when production predicate_name is entered.
func (s *BaseQueryListener) EnterPredicate_name(ctx *Predicate_nameContext) {}

// ExitPredicate_name is called when production predicate_name is exited.
func (s *BaseQueryListener) ExitPredicate_name(ctx *Predicate_nameContext) {}

// EnterParameter_list is called when production parameter_list is entered.
func (s *BaseQueryListener) EnterParameter_list(ctx *Parameter_listContext) {}

// ExitParameter_list is called when production parameter_list is exited.
func (s *BaseQueryListener) ExitParameter_list(ctx *Parameter_listContext) {}

// EnterParameter is called when production parameter is entered.
func (s *BaseQueryListener) EnterParameter(ctx *ParameterContext) {}

// ExitParameter is called when production parameter is exited.
func (s *BaseQueryListener) ExitParameter(ctx *ParameterContext) {}

// EnterType is called when production type is entered.
func (s *BaseQueryListener) EnterType(ctx *TypeContext) {}

// ExitType is called when production type is exited.
func (s *BaseQueryListener) ExitType(ctx *TypeContext) {}

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

// EnterEqualityExpression is called when production equalityExpression is entered.
func (s *BaseQueryListener) EnterEqualityExpression(ctx *EqualityExpressionContext) {}

// ExitEqualityExpression is called when production equalityExpression is exited.
func (s *BaseQueryListener) ExitEqualityExpression(ctx *EqualityExpressionContext) {}

// EnterRelationalExpression is called when production relationalExpression is entered.
func (s *BaseQueryListener) EnterRelationalExpression(ctx *RelationalExpressionContext) {}

// ExitRelationalExpression is called when production relationalExpression is exited.
func (s *BaseQueryListener) ExitRelationalExpression(ctx *RelationalExpressionContext) {}

// EnterAdditiveExpression is called when production additiveExpression is entered.
func (s *BaseQueryListener) EnterAdditiveExpression(ctx *AdditiveExpressionContext) {}

// ExitAdditiveExpression is called when production additiveExpression is exited.
func (s *BaseQueryListener) ExitAdditiveExpression(ctx *AdditiveExpressionContext) {}

// EnterMultiplicativeExpression is called when production multiplicativeExpression is entered.
func (s *BaseQueryListener) EnterMultiplicativeExpression(ctx *MultiplicativeExpressionContext) {}

// ExitMultiplicativeExpression is called when production multiplicativeExpression is exited.
func (s *BaseQueryListener) ExitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) {}

// EnterUnaryExpression is called when production unaryExpression is entered.
func (s *BaseQueryListener) EnterUnaryExpression(ctx *UnaryExpressionContext) {}

// ExitUnaryExpression is called when production unaryExpression is exited.
func (s *BaseQueryListener) ExitUnaryExpression(ctx *UnaryExpressionContext) {}

// EnterPrimary is called when production primary is entered.
func (s *BaseQueryListener) EnterPrimary(ctx *PrimaryContext) {}

// ExitPrimary is called when production primary is exited.
func (s *BaseQueryListener) ExitPrimary(ctx *PrimaryContext) {}

// EnterOperand is called when production operand is entered.
func (s *BaseQueryListener) EnterOperand(ctx *OperandContext) {}

// ExitOperand is called when production operand is exited.
func (s *BaseQueryListener) ExitOperand(ctx *OperandContext) {}

// EnterMethod_chain is called when production method_chain is entered.
func (s *BaseQueryListener) EnterMethod_chain(ctx *Method_chainContext) {}

// ExitMethod_chain is called when production method_chain is exited.
func (s *BaseQueryListener) ExitMethod_chain(ctx *Method_chainContext) {}

// EnterMethod_or_variable is called when production method_or_variable is entered.
func (s *BaseQueryListener) EnterMethod_or_variable(ctx *Method_or_variableContext) {}

// ExitMethod_or_variable is called when production method_or_variable is exited.
func (s *BaseQueryListener) ExitMethod_or_variable(ctx *Method_or_variableContext) {}

// EnterMethod_invocation is called when production method_invocation is entered.
func (s *BaseQueryListener) EnterMethod_invocation(ctx *Method_invocationContext) {}

// ExitMethod_invocation is called when production method_invocation is exited.
func (s *BaseQueryListener) ExitMethod_invocation(ctx *Method_invocationContext) {}

// EnterVariable is called when production variable is entered.
func (s *BaseQueryListener) EnterVariable(ctx *VariableContext) {}

// ExitVariable is called when production variable is exited.
func (s *BaseQueryListener) ExitVariable(ctx *VariableContext) {}

// EnterPredicate_invocation is called when production predicate_invocation is entered.
func (s *BaseQueryListener) EnterPredicate_invocation(ctx *Predicate_invocationContext) {}

// ExitPredicate_invocation is called when production predicate_invocation is exited.
func (s *BaseQueryListener) ExitPredicate_invocation(ctx *Predicate_invocationContext) {}

// EnterArgument_list is called when production argument_list is entered.
func (s *BaseQueryListener) EnterArgument_list(ctx *Argument_listContext) {}

// ExitArgument_list is called when production argument_list is exited.
func (s *BaseQueryListener) ExitArgument_list(ctx *Argument_listContext) {}

// EnterArgument is called when production argument is entered.
func (s *BaseQueryListener) EnterArgument(ctx *ArgumentContext) {}

// ExitArgument is called when production argument is exited.
func (s *BaseQueryListener) ExitArgument(ctx *ArgumentContext) {}

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

// EnterSelect_clause is called when production select_clause is entered.
func (s *BaseQueryListener) EnterSelect_clause(ctx *Select_clauseContext) {}

// ExitSelect_clause is called when production select_clause is exited.
func (s *BaseQueryListener) ExitSelect_clause(ctx *Select_clauseContext) {}

// EnterSelect_expression is called when production select_expression is entered.
func (s *BaseQueryListener) EnterSelect_expression(ctx *Select_expressionContext) {}

// ExitSelect_expression is called when production select_expression is exited.
func (s *BaseQueryListener) ExitSelect_expression(ctx *Select_expressionContext) {}

// Code generated from Query.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // Query

import "github.com/antlr4-go/antlr/v4"

// QueryListener is a complete listener for a parse tree produced by QueryParser.
type QueryListener interface {
	antlr.ParseTreeListener

	// EnterQuery is called when entering the query production.
	EnterQuery(c *QueryContext)

	// EnterSelect_list is called when entering the select_list production.
	EnterSelect_list(c *Select_listContext)

	// EnterSelect_item is called when entering the select_item production.
	EnterSelect_item(c *Select_itemContext)

	// EnterEntity is called when entering the entity production.
	EnterEntity(c *EntityContext)

	// EnterAlias is called when entering the alias production.
	EnterAlias(c *AliasContext)

	// EnterExpression is called when entering the expression production.
	EnterExpression(c *ExpressionContext)

	// EnterOrExpression is called when entering the orExpression production.
	EnterOrExpression(c *OrExpressionContext)

	// EnterAndExpression is called when entering the andExpression production.
	EnterAndExpression(c *AndExpressionContext)

	// EnterPrimary is called when entering the primary production.
	EnterPrimary(c *PrimaryContext)

	// EnterCondition is called when entering the condition production.
	EnterCondition(c *ConditionContext)

	// EnterMethod_chain is called when entering the method_chain production.
	EnterMethod_chain(c *Method_chainContext)

	// EnterMethod_or_variable is called when entering the method_or_variable production.
	EnterMethod_or_variable(c *Method_or_variableContext)

	// EnterMethod is called when entering the method production.
	EnterMethod(c *MethodContext)

	// EnterVariable is called when entering the variable production.
	EnterVariable(c *VariableContext)

	// EnterComparator is called when entering the comparator production.
	EnterComparator(c *ComparatorContext)

	// EnterValue is called when entering the value production.
	EnterValue(c *ValueContext)

	// EnterValue_list is called when entering the value_list production.
	EnterValue_list(c *Value_listContext)

	// ExitQuery is called when exiting the query production.
	ExitQuery(c *QueryContext)

	// ExitSelect_list is called when exiting the select_list production.
	ExitSelect_list(c *Select_listContext)

	// ExitSelect_item is called when exiting the select_item production.
	ExitSelect_item(c *Select_itemContext)

	// ExitEntity is called when exiting the entity production.
	ExitEntity(c *EntityContext)

	// ExitAlias is called when exiting the alias production.
	ExitAlias(c *AliasContext)

	// ExitExpression is called when exiting the expression production.
	ExitExpression(c *ExpressionContext)

	// ExitOrExpression is called when exiting the orExpression production.
	ExitOrExpression(c *OrExpressionContext)

	// ExitAndExpression is called when exiting the andExpression production.
	ExitAndExpression(c *AndExpressionContext)

	// ExitPrimary is called when exiting the primary production.
	ExitPrimary(c *PrimaryContext)

	// ExitCondition is called when exiting the condition production.
	ExitCondition(c *ConditionContext)

	// ExitMethod_chain is called when exiting the method_chain production.
	ExitMethod_chain(c *Method_chainContext)

	// ExitMethod_or_variable is called when exiting the method_or_variable production.
	ExitMethod_or_variable(c *Method_or_variableContext)

	// ExitMethod is called when exiting the method production.
	ExitMethod(c *MethodContext)

	// ExitVariable is called when exiting the variable production.
	ExitVariable(c *VariableContext)

	// ExitComparator is called when exiting the comparator production.
	ExitComparator(c *ComparatorContext)

	// ExitValue is called when exiting the value production.
	ExitValue(c *ValueContext)

	// ExitValue_list is called when exiting the value_list production.
	ExitValue_list(c *Value_listContext)
}

// Code generated from Query.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // Query

import "github.com/antlr4-go/antlr/v4"

// QueryListener is a complete listener for a parse tree produced by QueryParser.
type QueryListener interface {
	antlr.ParseTreeListener

	// EnterQuery is called when entering the query production.
	EnterQuery(c *QueryContext)

	// EnterPredicate_declarations is called when entering the predicate_declarations production.
	EnterPredicate_declarations(c *Predicate_declarationsContext)

	// EnterPredicate_declaration is called when entering the predicate_declaration production.
	EnterPredicate_declaration(c *Predicate_declarationContext)

	// EnterPredicate_name is called when entering the predicate_name production.
	EnterPredicate_name(c *Predicate_nameContext)

	// EnterParameter_list is called when entering the parameter_list production.
	EnterParameter_list(c *Parameter_listContext)

	// EnterParameter is called when entering the parameter production.
	EnterParameter(c *ParameterContext)

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

	// EnterEqualityExpression is called when entering the equalityExpression production.
	EnterEqualityExpression(c *EqualityExpressionContext)

	// EnterRelationalExpression is called when entering the relationalExpression production.
	EnterRelationalExpression(c *RelationalExpressionContext)

	// EnterAdditiveExpression is called when entering the additiveExpression production.
	EnterAdditiveExpression(c *AdditiveExpressionContext)

	// EnterMultiplicativeExpression is called when entering the multiplicativeExpression production.
	EnterMultiplicativeExpression(c *MultiplicativeExpressionContext)

	// EnterUnaryExpression is called when entering the unaryExpression production.
	EnterUnaryExpression(c *UnaryExpressionContext)

	// EnterPrimary is called when entering the primary production.
	EnterPrimary(c *PrimaryContext)

	// EnterOperand is called when entering the operand production.
	EnterOperand(c *OperandContext)

	// EnterMethod_chain is called when entering the method_chain production.
	EnterMethod_chain(c *Method_chainContext)

	// EnterMethod_or_variable is called when entering the method_or_variable production.
	EnterMethod_or_variable(c *Method_or_variableContext)

	// EnterMethod is called when entering the method production.
	EnterMethod(c *MethodContext)

	// EnterVariable is called when entering the variable production.
	EnterVariable(c *VariableContext)

	// EnterPredicate_invocation is called when entering the predicate_invocation production.
	EnterPredicate_invocation(c *Predicate_invocationContext)

	// EnterArgument_list is called when entering the argument_list production.
	EnterArgument_list(c *Argument_listContext)

	// EnterComparator is called when entering the comparator production.
	EnterComparator(c *ComparatorContext)

	// EnterValue is called when entering the value production.
	EnterValue(c *ValueContext)

	// EnterValue_list is called when entering the value_list production.
	EnterValue_list(c *Value_listContext)

	// ExitQuery is called when exiting the query production.
	ExitQuery(c *QueryContext)

	// ExitPredicate_declarations is called when exiting the predicate_declarations production.
	ExitPredicate_declarations(c *Predicate_declarationsContext)

	// ExitPredicate_declaration is called when exiting the predicate_declaration production.
	ExitPredicate_declaration(c *Predicate_declarationContext)

	// ExitPredicate_name is called when exiting the predicate_name production.
	ExitPredicate_name(c *Predicate_nameContext)

	// ExitParameter_list is called when exiting the parameter_list production.
	ExitParameter_list(c *Parameter_listContext)

	// ExitParameter is called when exiting the parameter production.
	ExitParameter(c *ParameterContext)

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

	// ExitEqualityExpression is called when exiting the equalityExpression production.
	ExitEqualityExpression(c *EqualityExpressionContext)

	// ExitRelationalExpression is called when exiting the relationalExpression production.
	ExitRelationalExpression(c *RelationalExpressionContext)

	// ExitAdditiveExpression is called when exiting the additiveExpression production.
	ExitAdditiveExpression(c *AdditiveExpressionContext)

	// ExitMultiplicativeExpression is called when exiting the multiplicativeExpression production.
	ExitMultiplicativeExpression(c *MultiplicativeExpressionContext)

	// ExitUnaryExpression is called when exiting the unaryExpression production.
	ExitUnaryExpression(c *UnaryExpressionContext)

	// ExitPrimary is called when exiting the primary production.
	ExitPrimary(c *PrimaryContext)

	// ExitOperand is called when exiting the operand production.
	ExitOperand(c *OperandContext)

	// ExitMethod_chain is called when exiting the method_chain production.
	ExitMethod_chain(c *Method_chainContext)

	// ExitMethod_or_variable is called when exiting the method_or_variable production.
	ExitMethod_or_variable(c *Method_or_variableContext)

	// ExitMethod is called when exiting the method production.
	ExitMethod(c *MethodContext)

	// ExitVariable is called when exiting the variable production.
	ExitVariable(c *VariableContext)

	// ExitPredicate_invocation is called when exiting the predicate_invocation production.
	ExitPredicate_invocation(c *Predicate_invocationContext)

	// ExitArgument_list is called when exiting the argument_list production.
	ExitArgument_list(c *Argument_listContext)

	// ExitComparator is called when exiting the comparator production.
	ExitComparator(c *ComparatorContext)

	// ExitValue is called when exiting the value production.
	ExitValue(c *ValueContext)

	// ExitValue_list is called when exiting the value_list production.
	ExitValue_list(c *Value_listContext)
}

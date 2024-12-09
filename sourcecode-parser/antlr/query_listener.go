// Code generated from Query.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // Query

import "github.com/antlr4-go/antlr/v4"

// QueryListener is a complete listener for a parse tree produced by QueryParser.
type QueryListener interface {
	antlr.ParseTreeListener

	// EnterQuery is called when entering the query production.
	EnterQuery(c *QueryContext)

	// EnterClass_declarations is called when entering the class_declarations production.
	EnterClass_declarations(c *Class_declarationsContext)

	// EnterClass_declaration is called when entering the class_declaration production.
	EnterClass_declaration(c *Class_declarationContext)

	// EnterClass_name is called when entering the class_name production.
	EnterClass_name(c *Class_nameContext)

	// EnterMethod_declarations is called when entering the method_declarations production.
	EnterMethod_declarations(c *Method_declarationsContext)

	// EnterMethod_declaration is called when entering the method_declaration production.
	EnterMethod_declaration(c *Method_declarationContext)

	// EnterMethod_name is called when entering the method_name production.
	EnterMethod_name(c *Method_nameContext)

	// EnterMethod_body is called when entering the method_body production.
	EnterMethod_body(c *Method_bodyContext)

	// EnterReturn_statement is called when entering the return_statement production.
	EnterReturn_statement(c *Return_statementContext)

	// EnterReturn_type is called when entering the return_type production.
	EnterReturn_type(c *Return_typeContext)

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

	// EnterType is called when entering the type production.
	EnterType(c *TypeContext)

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

	// EnterMethod_invocation is called when entering the method_invocation production.
	EnterMethod_invocation(c *Method_invocationContext)

	// EnterVariable is called when entering the variable production.
	EnterVariable(c *VariableContext)

	// EnterPredicate_invocation is called when entering the predicate_invocation production.
	EnterPredicate_invocation(c *Predicate_invocationContext)

	// EnterArgument_list is called when entering the argument_list production.
	EnterArgument_list(c *Argument_listContext)

	// EnterArgument is called when entering the argument production.
	EnterArgument(c *ArgumentContext)

	// EnterComparator is called when entering the comparator production.
	EnterComparator(c *ComparatorContext)

	// EnterValue is called when entering the value production.
	EnterValue(c *ValueContext)

	// EnterValue_list is called when entering the value_list production.
	EnterValue_list(c *Value_listContext)

	// EnterSelect_clause is called when entering the select_clause production.
	EnterSelect_clause(c *Select_clauseContext)

	// EnterSelect_expression is called when entering the select_expression production.
	EnterSelect_expression(c *Select_expressionContext)

	// ExitQuery is called when exiting the query production.
	ExitQuery(c *QueryContext)

	// ExitClass_declarations is called when exiting the class_declarations production.
	ExitClass_declarations(c *Class_declarationsContext)

	// ExitClass_declaration is called when exiting the class_declaration production.
	ExitClass_declaration(c *Class_declarationContext)

	// ExitClass_name is called when exiting the class_name production.
	ExitClass_name(c *Class_nameContext)

	// ExitMethod_declarations is called when exiting the method_declarations production.
	ExitMethod_declarations(c *Method_declarationsContext)

	// ExitMethod_declaration is called when exiting the method_declaration production.
	ExitMethod_declaration(c *Method_declarationContext)

	// ExitMethod_name is called when exiting the method_name production.
	ExitMethod_name(c *Method_nameContext)

	// ExitMethod_body is called when exiting the method_body production.
	ExitMethod_body(c *Method_bodyContext)

	// ExitReturn_statement is called when exiting the return_statement production.
	ExitReturn_statement(c *Return_statementContext)

	// ExitReturn_type is called when exiting the return_type production.
	ExitReturn_type(c *Return_typeContext)

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

	// ExitType is called when exiting the type production.
	ExitType(c *TypeContext)

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

	// ExitMethod_invocation is called when exiting the method_invocation production.
	ExitMethod_invocation(c *Method_invocationContext)

	// ExitVariable is called when exiting the variable production.
	ExitVariable(c *VariableContext)

	// ExitPredicate_invocation is called when exiting the predicate_invocation production.
	ExitPredicate_invocation(c *Predicate_invocationContext)

	// ExitArgument_list is called when exiting the argument_list production.
	ExitArgument_list(c *Argument_listContext)

	// ExitArgument is called when exiting the argument production.
	ExitArgument(c *ArgumentContext)

	// ExitComparator is called when exiting the comparator production.
	ExitComparator(c *ComparatorContext)

	// ExitValue is called when exiting the value production.
	ExitValue(c *ValueContext)

	// ExitValue_list is called when exiting the value_list production.
	ExitValue_list(c *Value_listContext)

	// ExitSelect_clause is called when exiting the select_clause production.
	ExitSelect_clause(c *Select_clauseContext)

	// ExitSelect_expression is called when exiting the select_expression production.
	ExitSelect_expression(c *Select_expressionContext)
}

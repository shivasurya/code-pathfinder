package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"sourcecode-parser/antlr" // Adjust this path to your generated parser
)

// Define the data model
type Query struct {
	SelectList  []SelectItem
	WhereClause *Expression
}

type SelectItem struct {
	Entity string
	Alias  string
}

type Expression struct {
	OrExpr []*AndExpression
}

type AndExpression struct {
	NotExpr []*NotExpression
}

type NotExpression struct {
	Negation bool
	Primary  *Primary
}

type Primary struct {
	Condition *Condition
	SubExpr   *Expression
}

type Condition struct {
	Alias       string
	MethodChain []string
	Comparator  string
	Value       Value
}

type Value struct {
	StringValue *string
	NumberValue *float64
}

// TreeShapeListener is an example listener for parse tree.
type TreeShapeListener struct {
	*parser.BaseQueryListener
	Model *Query
}

func NewTreeShapeListener() *TreeShapeListener {
	return &TreeShapeListener{
		Model: &Query{},
	}
}

func (t *TreeShapeListener) ExitQuery(ctx *parser.QueryContext) {
	t.Model.SelectList = parseSelectList(ctx.Select_list().(*parser.Select_listContext))
	if ctx.Expression() != nil {
		t.Model.WhereClause = parseExpression(ctx.Expression().(*parser.ExpressionContext))
	}
}

func parseSelectList(ctx *parser.Select_listContext) []SelectItem {
	var selectList []SelectItem
	for _, item := range ctx.AllSelect_item() {
		selectCtx := item.(*parser.Select_itemContext)
		selectList = append(selectList, SelectItem{
			Entity: selectCtx.Entity().GetText(),
			Alias:  selectCtx.Alias().GetText(),
		})
	}
	return selectList
}

func parseExpression(ctx *parser.ExpressionContext) *Expression {
	var orExpr []*AndExpression
	for _, andExprCtx := range ctx.OrExpression().AllAndExpression() {
		orExpr = append(orExpr, parseAndExpression(andExprCtx.(*parser.AndExpressionContext)))
	}
	return &Expression{OrExpr: orExpr}
}

func parseAndExpression(ctx *parser.AndExpressionContext) *AndExpression {
	var notExpr []*NotExpression
	for _, notExprCtx := range ctx.AllNotExpression() {
		notExpr = append(notExpr, parseNotExpression(notExprCtx.(*parser.NotExpressionContext)))
	}
	return &AndExpression{NotExpr: notExpr}
}

func parseNotExpression(ctx *parser.NotExpressionContext) *NotExpression {
	negation := ctx.NotExpression() != nil

	var primary *Primary
	if ctx.Primary() != nil {
		primaryCtx := ctx.Primary().(*parser.PrimaryContext)
		primary = parsePrimary(primaryCtx)
	}

	return &NotExpression{
		Negation: negation,
		Primary:  primary,
	}
}

func parsePrimary(ctx *parser.PrimaryContext) *Primary {
	var primary *Primary
	if ctx.Condition() != nil {
		primary = &Primary{
			Condition: parseCondition(ctx.Condition().(*parser.ConditionContext)),
		}
	} else if ctx.Expression() != nil {
		primary = &Primary{
			SubExpr: parseExpression(ctx.Expression().(*parser.ExpressionContext)),
		}
	}
	return primary
}

func parseCondition(ctx *parser.ConditionContext) *Condition {
	alias := ctx.Alias().GetText()
	var methodChain []string
	for _, methodOrVarCtx := range ctx.Method_chain().AllMethod_or_variable() {
		methodChain = append(methodChain, methodOrVarCtx.GetText())
	}
	comparator := ctx.Comparator().GetText()
	value := parseValue(ctx.Value())

	return &Condition{
		Alias:       alias,
		MethodChain: methodChain,
		Comparator:  comparator,
		Value:       *value,
	}
}

func parseValue(ctx parser.IValueContext) *Value {
	if ctx.STRING() != nil {
		str := strings.Trim(ctx.STRING().GetText(), "\"")
		return &Value{StringValue: &str}
	} else if ctx.NUMBER() != nil {
		num, _ := strconv.ParseFloat(ctx.NUMBER().GetText(), 64)
		return &Value{NumberValue: &num}
	}
	return nil
}

// Evaluation Functions

func evaluateExpression(expr *Expression) bool {
	if expr == nil {
		return false
	}

	result := evaluateAndExpression(expr.OrExpr[0])
	for _, andExpr := range expr.OrExpr[1:] {
		result = result || evaluateAndExpression(andExpr)
	}

	return result
}

func evaluateAndExpression(andExpr *AndExpression) bool {
	if andExpr == nil {
		return false
	}

	result := evaluateNotExpression(andExpr.NotExpr[0])
	for _, notExpr := range andExpr.NotExpr[1:] {
		result = result && evaluateNotExpression(notExpr)
	}

	return result
}

func evaluateNotExpression(notExpr *NotExpression) bool {
	if notExpr == nil {
		return false
	}

	result := evaluatePrimary(notExpr.Primary)
	if notExpr.Negation {
		return !result
	}

	return result
}

func evaluatePrimary(primary *Primary) bool {
	if primary == nil {
		return false
	}

	if primary.Condition != nil {
		return evaluateCondition(primary.Condition)
	} else if primary.SubExpr != nil {
		return evaluateExpression(primary.SubExpr)
	}

	return false
}

func evaluateCondition(cond *Condition) bool {
	// Implement your own condition evaluation logic here
	// For demonstration purposes, we'll assume all conditions evaluate to true
	return true
}

func main() {
	queries := []string{
		"FIND User AS u, Data AS d WHERE u.age > 30 AND d.ty = ( 30, 100, 1000 )",
		"FIND User AS u WHERE u.getData() > 30",
		"FIND User AS u WHERE u.age > 30 AND u.name = \"John\"",
		"FIND User AS u WHERE u.getData() > 30 AND u.name = \"John\"",
		"FIND User AS u WHERE u.age > 30 OR (u.salary > 50000 AND u.id < 100)",
		"FIND User AS u WHERE (u.age > 30 AND u.name = \"John\") OR (u.getData() > 30 AND (u.id < 100 AND u.salary > 50000))",
		"FIND User AS u WHERE NOT u.age > 30 AND (u.name = \"John\" OR u.getData() > 30)",
	}

	for _, input := range queries {
		// Create the ANTLR input stream
		is := antlr.NewInputStream(input)

		// Create the lexer
		lexer := parser.NewQueryLexer(is)
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

		// Create the parser
		p := parser.NewQueryParser(stream)

		// Parse the query
		tree := p.Query()

		// Create a listener
		listener := NewTreeShapeListener()

		// Walk the tree
		antlr.ParseTreeWalkerDefault.Walk(listener, tree)

		// Print the model
		fmt.Printf("Query: %s\n", input)
		fmt.Printf("Model: %+v\n", listener.Model)

		// Evaluate the where clause
		result := evaluateExpression(listener.Model.WhereClause)
		fmt.Printf("Evaluation result: %v\n\n", result)
	}
}

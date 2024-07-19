package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
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

func evaluateExpression(expr *Expression, data map[string]interface{}) bool {
	if expr == nil {
		return false
	}

	stack := []bool{}
	for _, andExpr := range expr.OrExpr {
		stack = append(stack, evaluateAndExpression(andExpr, data))
	}

	result := stack[0]
	for i := 1; i < len(stack); i++ {
		result = result || stack[i]
	}

	return result
}

func evaluateAndExpression(andExpr *AndExpression, data map[string]interface{}) bool {
	if andExpr == nil {
		return false
	}

	stack := []bool{}
	for _, notExpr := range andExpr.NotExpr {
		stack = append(stack, evaluateNotExpression(notExpr, data))
	}

	result := stack[0]
	for i := 1; i < len(stack); i++ {
		result = result && stack[i]
	}

	return result
}

func evaluateNotExpression(notExpr *NotExpression, data map[string]interface{}) bool {
	if notExpr == nil {
		return false
	}

	result := evaluatePrimary(notExpr.Primary, data)
	if notExpr.Negation {
		return !result
	}

	return result
}

func evaluatePrimary(primary *Primary, data map[string]interface{}) bool {
	if primary == nil {
		return false
	}

	if primary.Condition != nil {
		return evaluateCondition(primary.Condition, data)
	} else if primary.SubExpr != nil {
		return evaluateExpression(primary.SubExpr, data)
	}

	return false
}

func evaluateCondition(cond *Condition, data map[string]interface{}) bool {
	aliasData, exists := data[cond.Alias]
	if !exists {
		return false
	}

	fieldValue := aliasData.(map[string]interface{})
	for i, methodOrVar := range cond.MethodChain {
		if i == len(cond.MethodChain)-1 {
			break
		}
		fieldValue = fieldValue[methodOrVar].(map[string]interface{})
	}

	lastMethodOrVar := cond.MethodChain[len(cond.MethodChain)-1]
	var fieldVal interface{}
	if strings.HasSuffix(lastMethodOrVar, "()") {
		// Handle method calls, e.g., u.age()
		method := strings.TrimSuffix(lastMethodOrVar, "()")
		fieldVal = fieldValue[method]
	} else {
		// Handle field accesses, e.g., u.age
		fieldVal = fieldValue[lastMethodOrVar]
	}

	var value interface{}
	if cond.Value.StringValue != nil {
		value = *cond.Value.StringValue
	} else if cond.Value.NumberValue != nil {
		value = *cond.Value.NumberValue
	}

	switch cond.Comparator {
	case "=":
		return fieldVal == value
	case "!=":
		return fieldVal != value
	case "<":
		fieldValFloat, _ := fieldVal.(float64)
		valueFloat, _ := value.(float64)
		return fieldValFloat < valueFloat
	case ">":
		fieldValFloat, _ := fieldVal.(float64)
		valueFloat, _ := value.(float64)
		return fieldValFloat > valueFloat
	case "<=":
		fieldValFloat, _ := fieldVal.(float64)
		valueFloat, _ := value.(float64)
		return fieldValFloat <= valueFloat
	case ">=":
		fieldValFloat, _ := fieldVal.(float64)
		valueFloat, _ := value.(float64)
		return fieldValFloat >= valueFloat
	default:
		return false
	}
}

func main() {
	queries := []string{
		"FIND User AS u, Data AS d WHERE u.age > 30",
		"FIND User AS u WHERE u.getData() < 30",
		"FIND User AS u WHERE u.age > 30 AND u.name = \"John\"",
		"FIND User AS u WHERE u.getData() > 30 AND u.name = \"John\"",
		"FIND User AS u WHERE u.age > 30 OR (u.salary > 50000 AND u.id < 100)",
		"FIND User AS u WHERE (u.age > 30 AND u.name = \"John\") OR (u.getData() > 30 AND (u.id < 100 AND u.salary > 50000))",
		"FIND User AS u WHERE NOT u.age > 30 AND (u.name = \"John\" OR u.getData() > 30)",
	}

	for _, input := range queries {

		var resultHolder []map[string]interface{}
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

		// Sample data for evaluation
		data := []map[string]interface{}{
			{
				"u": map[string]interface{}{
					"age":        34.0,
					"name":       "John",
					"getData":    4.0,
					"id":         50.0,
					"salary":     60000.0,
					"ty":         []float64{30.0, 100.0, 1000.0},
					"department": "HR",
				},
				"d": map[string]interface{}{
					"ty": []float64{30.0, 100.0, 1000.0},
				},
			},
			{
				"u": map[string]interface{}{
					"age":        28.0,
					"name":       "Alice",
					"getData":    60.0,
					"id":         51.0,
					"salary":     75000.0,
					"ty":         []float64{50.0, 200.0, 2000.0},
					"department": "IT",
				},
				"d": map[string]interface{}{
					"ty": []float64{50.0, 200.0, 2000.0},
				},
			},
			{
				"u": map[string]interface{}{
					"age":        42.0,
					"name":       "Bob",
					"getData":    30.0,
					"id":         52.0,
					"salary":     90000.0,
					"ty":         []float64{40.0, 150.0, 1500.0},
					"department": "Finance",
				},
				"d": map[string]interface{}{
					"ty": []float64{40.0, 150.0, 1500.0},
				},
			},
			{
				"u": map[string]interface{}{
					"age":        31.0,
					"name":       "Emma",
					"getData":    55.0,
					"id":         53.0,
					"salary":     70000.0,
					"ty":         []float64{35.0, 120.0, 1200.0},
					"department": "Marketing",
				},
				"d": map[string]interface{}{
					"ty": []float64{35.0, 120.0, 1200.0},
				},
			},
		}

		// Print the model
		fmt.Printf("Query: %s\n", input)
		fmt.Printf("Model: %+v\n", listener.Model)

		// Evaluate the where clause
		for _, datum := range data {
			result := evaluateExpression(listener.Model.WhereClause, datum)
			if result {
				resultHolder = append(resultHolder, datum)
			}
		}
		fmt.Printf("Evaluation result: %v\n\n", resultHolder)
	}
}

package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
)

// Data Structures
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
	Expr      *Expression
}

type Condition struct {
	Alias            string
	MethodChain      string
	Comparator       string
	Value            Value
	RightMethodChain string
}

type Value struct {
	StringValue  string
	NumberValue  float64
	IntegerValue int
	IsList       bool
	IsInteger    bool
	IsFloat      bool
	IsNull       bool
	List         []Value
}

// TreeNode represents a node in the parse tree
type TreeNode struct {
	Type     string
	Text     string
	Children []*TreeNode
}

// NewTreeNode creates a new tree node
func NewTreeNode(nodeType, text string) *TreeNode {
	return &TreeNode{
		Type:     nodeType,
		Text:     text,
		Children: []*TreeNode{},
	}
}

// PrintTreeDFS recursively prints the tree using DFS
func PrintTreeDFS(node *TreeNode, depth int) {
	if node == nil {
		return
	}

	for i := 0; i < depth; i++ {
		fmt.Print("  ")
	}
	if node.Text != "" {
		fmt.Printf("%s: %s\n", node.Type, node.Text)
	} else {
		fmt.Printf("%s\n", node.Type)
	}

	for _, child := range node.Children {
		PrintTreeDFS(child, depth+1)
	}
}

// Listener for Parse Tree
type TreeShapeListener struct {
	*parser.BaseQueryListener
	Model *Query
	Root  *TreeNode
	Stack []*TreeNode
}

func NewTreeShapeListener() *TreeShapeListener {
	return &TreeShapeListener{
		Model: &Query{},
		Stack: []*TreeNode{},
	}
}

func getRuleName(ctx antlr.ParserRuleContext) string {
	switch ctx.(type) {
	case *parser.QueryContext:
		return "Query"
	case *parser.Select_listContext:
		return "Select_list"
	case *parser.Select_itemContext:
		return "Select_item"
	case *parser.EntityContext:
		return "Entity"
	case *parser.AliasContext:
		return "Alias"
	case *parser.ExpressionContext:
		return "Expression"
	case *parser.OrExpressionContext:
		return "OrExpression"
	case *parser.AndExpressionContext:
		return "AndExpression"
	case *parser.NotExpressionContext:
		return "NotExpression"
	case *parser.PrimaryContext:
		return "Primary"
	case *parser.ConditionContext:
		return "Condition"
	case *parser.Method_chainContext:
		return "Method_chain"
	case *parser.Method_or_variableContext:
		return "Method_or_variable"
	case *parser.MethodContext:
		return "Method"
	case *parser.VariableContext:
		return "Variable"
	case *parser.ComparatorContext:
		return "Comparator"
	case *parser.ValueContext:
		return "Value"
	case *parser.Value_listContext:
		return "Value_list"
	default:
		return "Unknown"
	}
}

// VisitTerminal is called when a terminal node is visited.
func (l *TreeShapeListener) VisitTerminal(node antlr.TerminalNode) {
	text := node.GetText()
	var nodeType string

	switch text {
	case "AND":
		nodeType = "AND"
	case "OR":
		nodeType = "OR"
	case "WHERE":
		nodeType = "WHERE"
	case "AS":
		nodeType = "AS"
	default:
		return
	}

	terminalNode := NewTreeNode(nodeType, text)
	if l.Root == nil {
		l.Root = terminalNode
	} else if len(l.Stack) > 0 {
		l.Stack[len(l.Stack)-1].Children = append(l.Stack[len(l.Stack)-1].Children, terminalNode)
	}
}

func (l *TreeShapeListener) EnterEveryRule(ctx antlr.ParserRuleContext) {
	ruleName := getRuleName(ctx)
	node := NewTreeNode(ruleName, ctx.GetText())
	if l.Root == nil {
		l.Root = node
	}
	if len(l.Stack) > 0 {
		l.Stack[len(l.Stack)-1].Children = append(l.Stack[len(l.Stack)-1].Children, node)
	}
	l.Stack = append(l.Stack, node)
}

// ExitEveryRule is called when any rule is exited.
func (l *TreeShapeListener) ExitEveryRule(ctx antlr.ParserRuleContext) {
	l.Stack = l.Stack[:len(l.Stack)-1]
}

func (l *TreeShapeListener) EnterQuery(ctx *parser.QueryContext) {
	l.Model = &Query{}
}

func (l *TreeShapeListener) EnterSelect_item(ctx *parser.Select_itemContext) {
	entity := ctx.Entity().GetText()
	alias := ctx.Alias().GetText()
	l.Model.SelectList = append(l.Model.SelectList, SelectItem{Entity: entity, Alias: alias})
}

func (l *TreeShapeListener) EnterExpression(ctx *parser.ExpressionContext) {
	l.Model.WhereClause = parseExpression(ctx)
}

func parseExpression(ctx *parser.ExpressionContext) *Expression {

	expr := &Expression{}
	for _, andExprCtx := range ctx.OrExpression().AllAndExpression() {
		expr.OrExpr = append(expr.OrExpr, parseAndExpression(andExprCtx.(*parser.AndExpressionContext)))
	}
	return expr
}

func parseAndExpression(ctx *parser.AndExpressionContext) *AndExpression {
	andExpr := &AndExpression{}
	for _, notExprCtx := range ctx.AllNotExpression() {
		andExpr.NotExpr = append(andExpr.NotExpr, parseNotExpression(notExprCtx.(*parser.NotExpressionContext)))
	}
	return andExpr
}

func parseNotExpression(ctx *parser.NotExpressionContext) *NotExpression {
	notExpr := &NotExpression{
		Negation: ctx.NotExpression() != nil,
	}

	if ctx.Primary() != nil {
		notExpr.Primary = parsePrimary(ctx.Primary().(*parser.PrimaryContext))
	}

	return notExpr
}

func parsePrimary(ctx *parser.PrimaryContext) *Primary {
	primary := &Primary{}

	if ctx.Condition() != nil {
		primary.Condition = parseCondition(ctx.Condition().(*parser.ConditionContext))
	} else if ctx.Expression() != nil {
		primary.Expr = parseExpression(ctx.Expression().(*parser.ExpressionContext))
	}

	return primary
}

func parseCondition(ctx *parser.ConditionContext) *Condition {
	condition := &Condition{
		Alias:       ctx.Alias().GetText(),
		MethodChain: ctx.Method_chain(0).GetText(),
		Comparator:  ctx.Comparator().GetText(),
	}

	if ctx.Value() != nil {
		condition.Value = parseValue(ctx.Value().GetText())
	} else if ctx.Method_chain(1) != nil {
		condition.RightMethodChain = ctx.Method_chain(1).GetText()
	} else if ctx.Value_list() != nil {
		condition.Value = parseValueList(ctx.Value_list())
	}

	return condition
}

// String method for Expression
func (e *Expression) String() string {
	var parts []string
	for _, andExpr := range e.OrExpr {
		parts = append(parts, andExpr.String())
	}
	return strings.Join(parts, " OR ")
}

// String method for AndExpression
func (ae *AndExpression) String() string {
	var parts []string
	for _, notExpr := range ae.NotExpr {
		parts = append(parts, notExpr.String())
	}
	return strings.Join(parts, " AND ")
}

// String method for NotExpression
func (ne *NotExpression) String() string {
	if ne.Negation {
		return "NOT " + ne.Primary.String()
	}
	return ne.Primary.String()
}

// String method for Primary
func (p *Primary) String() string {
	if p.Condition != nil {
		return p.Condition.String()
	}
	if p.Expr != nil {
		return "(" + p.Expr.String() + ")"
	}
	return ""
}

// String method for Condition
func (c *Condition) String() string {
	right := ""
	if c.RightMethodChain != "" {
		right = c.RightMethodChain
	} else if c.Value.StringValue != "" {
		right = "\"" + c.Value.StringValue + "\""
	} else if c.Value.NumberValue != 0 {
		right = fmt.Sprintf("%f", c.Value.NumberValue)
	} else if c.Value.IntegerValue != 0 {
		right = fmt.Sprintf("%d", c.Value.IntegerValue)
	} else if c.Value.IsList {
		var listParts []string
		for _, v := range c.Value.List {
			if v.StringValue != "" {
				listParts = append(listParts, "\""+v.StringValue+"\"")
			} else {
				listParts = append(listParts, fmt.Sprintf("%f", v.NumberValue))
			}
		}
		right = "(" + strings.Join(listParts, ", ") + ")"
	}
	return fmt.Sprintf("%s.%s %s %s", c.Alias, c.MethodChain, c.Comparator, right)
}

func parseValue(text string) Value {
	if strings.HasPrefix(text, "\"") {
		return Value{StringValue: strings.Trim(text, "\"")}
	}
	if intValue, err := strconv.Atoi(text); err == nil {
		return Value{IntegerValue: intValue, IsInteger: true}
	}
	if floatValue, err := strconv.ParseFloat(text, 64); err == nil {
		return Value{NumberValue: floatValue, IsFloat: true}
	}
	return Value{IsNull: true}
}

func parseValueList(ctx parser.IValue_listContext) Value {
	var list []Value
	for _, v := range ctx.AllValue() {
		list = append(list, parseValue(v.GetText()))
	}
	return Value{IsList: true, List: list}
}

// Stack-based Evaluation Functions
func evaluateExpression(expr *Expression, data map[string]interface{}) bool {
	if expr == nil {
		return false
	}

	for _, andExpr := range expr.OrExpr {
		if evaluateAndExpression(andExpr, data) {
			return true
		}
	}

	return false
}

func evaluateAndExpression(andExpr *AndExpression, data map[string]interface{}) bool {
	if andExpr == nil {
		return false
	}

	for _, notExpr := range andExpr.NotExpr {
		if !evaluateNotExpression(notExpr, data) {
			return false
		}
	}

	return true
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
	}
	return evaluateExpression(primary.Expr, data)
}

func evaluateCondition(condition *Condition, data map[string]interface{}) bool {
	leftValue := getValueFromMethodChain(data[condition.Alias], condition.MethodChain)
	var rightValue interface{}

	if condition.RightMethodChain != "" {
		rightValue = getValueFromMethodChain(data[condition.Alias], condition.RightMethodChain)
	} else {
		if condition.Value.IsNull {
			rightValue = nil
		} else if condition.Value.IsFloat {
			rightValue = condition.Value.NumberValue
		} else if condition.Value.IsInteger {
			rightValue = condition.Value.IntegerValue
		} else if condition.Value.StringValue != "" {
			rightValue = condition.Value.StringValue
		} else if condition.Value.IsList {
			rightValue = condition.Value.List
		}
	}

	fmt.Print("Left Value: ")
	fmt.Println(leftValue)
	fmt.Print("Right Value: ")
	fmt.Println(rightValue)
	fmt.Printf("Comparator: %s\n", condition.Comparator)
	fmt.Print("Comparison Result: ")
	fmt.Println(compareValues(leftValue, rightValue))
	switch condition.Comparator {
	case "=":
		return compareValues(leftValue, rightValue) == 0
	case "!=":
		return compareValues(leftValue, rightValue) != 0
	case "<":
		return compareValues(leftValue, rightValue) < 0
	case ">":
		return compareValues(leftValue, rightValue) > 0
	case "<=":
		return compareValues(leftValue, rightValue) <= 0
	case ">=":
		return compareValues(leftValue, rightValue) >= 0
	case "LIKE":
		return matchLikePattern(leftValue, rightValue)
	case "IN":
		return valueInList(leftValue, rightValue)
	default:
		return false
	}
}

func getValueFromMethodChain(obj interface{}, methodChain string) interface{} {
	if obj == nil || methodChain == "" {
		return nil
	}

	value := reflect.ValueOf(obj)
	methods := strings.Split(methodChain, ".")

	for _, method := range methods {
		if strings.Contains(method, "()") {
			method = strings.TrimSuffix(method, "()")
			valueMap := obj.(map[string]interface{})
			methodCall := reflect.ValueOf(valueMap[method])
			value = methodCall.Call(nil)[0]
		} else {
			value = value.MapIndex(reflect.ValueOf(method))
		}

		if !value.IsValid() {
			return nil
		}
	}

	return value.Interface()
}

func compareValues(a, b interface{}) int {
	switch aTyped := a.(type) {
	case int:
		bTyped, ok := b.(int)
		if !ok {
			return 1
		}
		if aTyped < bTyped {
			return -1
		} else if aTyped > bTyped {
			return 1
		}
	case float64:
		bTyped, ok := b.(float64)
		if !ok {
			return 1
		}
		if aTyped < bTyped {
			return -1
		} else if aTyped > bTyped {
			return 1
		}
	case string:
		bTyped, ok := b.(string)
		if !ok {
			return 1
		}
		return strings.Compare(aTyped, bTyped)
	case Value:
		if aTyped.StringValue != "" {
			return compareValues(aTyped.StringValue, b)
		}
		return compareValues(aTyped.NumberValue, b)
	}

	return 0
}

func matchLikePattern(value, pattern interface{}) bool {
	valueStr, ok := value.(string)
	if !ok {
		return false
	}

	patternStr, ok := pattern.(string)
	if !ok {
		return false
	}

	regex := strings.ReplaceAll(patternStr, "%", ".*")
	regex = "^" + regex + "$"

	match, err := regexp.MatchString(regex, valueStr)
	if err != nil {
		return false
	}

	return match
}

func valueInList(value interface{}, list interface{}) bool {
	listValue, ok := list.(Value)
	if !ok {
		return false
	}

	for _, v := range listValue.List {
		if compareValues(value, v.StringValue) == 0 || compareValues(value, v.NumberValue) == 0 {
			return true
		}
	}

	return false
}

func main() {
	queries := []string{
		"FIND User AS u WHERE (u.age > 1 OR u.age < 10)",
		"FIND User AS u WHERE u.getData() < 30",
		"FIND User AS u WHERE u.age > 30 AND u.name = \"john\"",
		"FIND User AS u WHERE u.getData() > 30 AND u.name = \"john\"",
		"FIND User AS u WHERE u.age > 30 OR (u.salary > 50000 AND u.id < 100)",
		"FIND User AS u WHERE (u.age > 30 AND u.name = \"John\") OR (u.getData() > 30 AND (u.id < 100 AND u.salary > 50000))",
		"FIND User AS u WHERE NOT u.age > 30 AND (u.name = \"John\" OR u.getData() > 30)",
		"FIND User AS u WHERE u.name LIKE \"shiva%\"",
		"FIND User AS u WHERE u.getAddress() IN (\"addr1\",\"addr2\", \"addr3\")",
		"FIND User AS u WHERE u.name = u.getLastName()",
		"FIND User AS u WHERE u.name = \"shiva\"",
		"FIND User AS u WHERE u.getAge() = 24.9899",
	}

	data := []map[string]interface{}{
		{
			"u": map[string]interface{}{
				"name":   "shiva",
				"age":    24.000000,
				"salary": 50000,
				"getData": func() int {
					return 22
				},
				"getAddress": func() string {
					return "addr1"
				},
				"getAge": func() float64 {
					return 24.9899
				},
				"getLastName": func() string {
					return "shiva"
				},
			},
		},
		{
			"u": map[string]interface{}{
				"name":   "john",
				"age":    32,
				"salary": 65000,
				"getData": func() int {
					return 40
				},
				"getAddress": func() string {
					return "addr2"
				},
				"getAge": func() float64 {
					return 32.5678
				},
				"getLastName": func() string {
					return "doe"
				},
			},
		},
		{
			"u": map[string]interface{}{
				"name":   "alice",
				"age":    28,
				"salary": 55000,
				"getData": func() int {
					return 35
				},
				"getAddress": func() string {
					return "addr3"
				},
				"getAge": func() float64 {
					return 28.1234
				},
				"getLastName": func() string {
					return "smith"
				},
			},
		},
	}

	for _, input := range queries {
		resultHolder := []map[string]interface{}{}
		is := antlr.NewInputStream(input)
		lexer := parser.NewQueryLexer(is)
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		p := parser.NewQueryParser(stream)
		tree := p.Query()
		listener := NewTreeShapeListener()
		antlr.ParseTreeWalkerDefault.Walk(listener, tree)

		fmt.Println(tree.ToStringTree(nil, p))

		fmt.Printf("Query: %s\n", input)
		fmt.Printf("Model: %+v\n\n", listener.Model)
		fmt.Printf("WhereClause: %s\n\n", listener.Model.WhereClause.String())

		PrintTreeDFS(listener.Root, 0)

		for _, datum := range data {
			result := evaluateExpression(listener.Model.WhereClause, datum)
			if result {
				resultHolder = append(resultHolder, datum)
			}
		}
		fmt.Printf("Result: %+v\n\n", resultHolder)
	}
}

// Unit Tests
func TestEvaluateExpression(t *testing.T) {
	queries := []string{
		"FIND User AS u WHERE ( u.age > 1 AND u.age < 10 )",
		"FIND User AS u WHERE u.getData() < 30",
		"FIND User AS u WHERE u.age > 30 AND u.name = \"John\"",
		"FIND User AS u WHERE u.getData() > 30 AND u.name = \"John\"",
		"FIND User AS u WHERE u.age > 30 OR (u.salary > 50000 AND u.id < 100)",
		"FIND User AS u WHERE (u.age > 30 AND u.name = \"John\") OR (u.getData() > 30 AND (u.id < 100 AND u.salary > 50000))",
		"FIND User AS u WHERE NOT u.age > 30 AND (u.name = \"John\" OR u.getData() > 30)",
		"FIND User AS u WHERE u.name LIKE \"shiva%\"",
		"FIND User AS u WHERE u.getAddress() IN (\"addr1\",\"addr2\", \"addr3\")",
		"FIND User AS u WHERE u.name = u.getLastName()",
		"FIND User AS u WHERE u.name = \"shiva\"",
		"FIND User AS u WHERE u.getAge() = 24.9899",
	}

	data := []map[string]interface{}{
		{
			"u": map[string]interface{}{
				"name":   "shiva",
				"age":    24,
				"salary": 50000,
				"getData": func() int {
					return 22
				},
				"getAddress": func() string {
					return "addr1"
				},
				"getAge": func() float64 {
					return 24.9899
				},
				"getLastName": func() string {
					return "shiva"
				},
			},
		},
		{
			"u": map[string]interface{}{
				"name":   "john",
				"age":    32,
				"salary": 65000,
				"getData": func() int {
					return 40
				},
				"getAddress": func() string {
					return "addr2"
				},
				"getAge": func() float64 {
					return 32.5678
				},
				"getLastName": func() string {
					return "doe"
				},
			},
		},
		{
			"u": map[string]interface{}{
				"name":   "alice",
				"age":    28,
				"salary": 55000,
				"getData": func() int {
					return 35
				},
				"getAddress": func() string {
					return "addr3"
				},
				"getAge": func() float64 {
					return 28.1234
				},
				"getLastName": func() string {
					return "smith"
				},
			},
		},
	}

	for _, input := range queries {
		resultHolder := []map[string]interface{}{}
		is := antlr.NewInputStream(input)
		lexer := parser.NewQueryLexer(is)
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		p := parser.NewQueryParser(stream)
		tree := p.Query()
		listener := NewTreeShapeListener()
		antlr.ParseTreeWalkerDefault.Walk(listener, tree)

		fmt.Println(tree.ToStringTree(nil, p))

		for _, datum := range data {
			result := evaluateExpression(listener.Model.WhereClause, datum)
			if result {
				resultHolder = append(resultHolder, datum)
			}
		}
		expectedResult := determineExpectedResult(input)
		if !reflect.DeepEqual(resultHolder, expectedResult) {
			t.Errorf("Query: %s\nExpected: %+v\nGot: %+v\n", input, expectedResult, resultHolder)
		}
	}
}

func determineExpectedResult(query string) []map[string]interface{} {
	// Add logic to determine expected results based on the query
	// For demonstration purposes, return an empty list
	return []map[string]interface{}{}
}

package parser

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

type Parameter struct {
	Name         string
	Type         string
	DefaultValue string
}

type Predicate struct {
	PredicateName string
	Parameter     []Parameter
	Body          string
}

type PredicateInvocation struct {
	PredicateName string
	Parameter     []Parameter
	Predicate     Predicate
}

// ExpressionNode represents a node in the expression tree
type ExpressionNode struct {
	Type     string           `json:"type"`            // Type of node: "binary", "unary", "literal", "variable", "method_call", "predicate_call"
	Operator string           `json:"operator"`        // Operator for binary/unary operations
	Value    string           `json:"value"`           // Value for literals, variable names, method names
	Entity   string           `json:"entity"`          // Entity name
	Left     *ExpressionNode  `json:"left,omitempty"`  // Left operand for binary operations
	Right    *ExpressionNode  `json:"right,omitempty"` // Right operand for binary operations
	Args     []ExpressionNode `json:"args,omitempty"`  // Arguments for method/predicate calls
}

type Query struct {
	Classes             []ClassDeclaration
	SelectList          []SelectList
	Expression          string
	Condition           []string
	ExpressionTree      *ExpressionNode // New field to store the expression tree
	Predicate           []Predicate
	PredicateInvocation []PredicateInvocation
	SelectOutput        []SelectOutput
}

type CustomQueryListener struct {
	BaseQueryListener
	expression          strings.Builder
	selectList          []SelectList
	condition           []string
	Predicate           []Predicate
	PredicateInvocation []PredicateInvocation
	Classes             []ClassDeclaration
	State               State
	SelectOutput        []SelectOutput
	ExpressionTree      *ExpressionNode   // New field to store the expression tree
	currentExpression   []*ExpressionNode // Stack to track current expression being built
}

func (l *CustomQueryListener) EnterMethod_chain(ctx *Method_chainContext) { //nolint:all
	// Handle class method calls
	if ctx.Class_name() != nil {
		className := ctx.Class_name().GetText()
		methodName := ctx.Method_name().GetText()

		// Find the class and method
		for _, class := range l.Classes {
			if class.Name == className {
				for _, method := range class.Methods {
					if method.Name == methodName {
						// Store method call information
						l.SelectOutput = append(l.SelectOutput, SelectOutput{
							SelectEntity: ctx.GetText(),
							Type:         "class_method",
						})
						return
					}
				}
			}
		}
	}

	// Handle existing method chain logic
	if ctx.Method_name() != nil {
		l.SelectOutput = append(l.SelectOutput, SelectOutput{
			SelectEntity: ctx.GetText(),
			Type:         "method_chain",
		})
	}
}

type State struct {
	isInPredicateDeclaration bool
}

type ClassDeclaration struct {
	Name    string
	Methods []MethodDeclaration
}

type MethodDeclaration struct {
	Name       string
	ReturnType string
	Body       string
}

type SelectList struct {
	Entity string
	Alias  string
}

type SelectOutput struct {
	SelectEntity string
	Type         string
}

func (l *CustomQueryListener) EnterClass_declaration(ctx *Class_declarationContext) { //nolint:all
	className := ctx.Class_name().GetText()
	class := ClassDeclaration{
		Name: className,
	}
	l.Classes = append(l.Classes, class)
}

func (l *CustomQueryListener) EnterMethod_declaration(ctx *Method_declarationContext) { //nolint:all
	if len(l.Classes) > 0 {
		currentClass := &l.Classes[len(l.Classes)-1]
		method := MethodDeclaration{
			Name:       ctx.Method_name().GetText(),
			ReturnType: ctx.Return_type().GetText(),
			Body:       ctx.Method_body().GetText(),
		}
		currentClass.Methods = append(currentClass.Methods, method)
	}
}

type customErrorListener struct {
	*antlr.DefaultErrorListener
	errors []string
}

func (l *customErrorListener) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	l.errors = append(l.errors, fmt.Sprintf("line %d:%d %s", line, column, msg))
}

func NewCustomQueryListener() *CustomQueryListener {
	return &CustomQueryListener{
		BaseQueryListener: BaseQueryListener{},
	}
}

//nolint:all
func (l *CustomQueryListener) EnterSelect_expression(ctx *Select_expressionContext) {
	// get the select expression type and set it to the select output
	var selectType string
	switch {
	case ctx.Variable() != nil:
		selectType = "variable"
	case ctx.Method_chain() != nil:
		selectType = "method_chain"
	case ctx.STRING() != nil:
		selectType = "string"
	}

	l.SelectOutput = append(l.SelectOutput, SelectOutput{
		SelectEntity: ctx.GetText(),
		Type:         selectType,
	})
}

//nolint:all
func (l *CustomQueryListener) EnterSelect_list(ctx *Select_listContext) {
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i).(antlr.ParseTree)
		if child, ok := child.(ISelect_itemContext); ok {
			l.selectList = append(l.selectList, SelectList{
				Entity: child.Entity().GetText(),
				Alias:  child.Alias().GetText(),
			})
		}
	}
}

//nolint:all
func (l *CustomQueryListener) EnterPredicate_invocation(ctx *Predicate_invocationContext) {
	predicateName := ctx.Predicate_name().GetText()
	parameters := ctx.Argument_list().GetText()
	// split the arguments by comma
	invokedPredicateArgs := strings.Split(parameters, ",")
	arguments := l.extractArguments(invokedPredicateArgs)
	invokedPredicate := PredicateInvocation{
		PredicateName: predicateName,
		Parameter:     arguments,
	}
	matchedPredicate, err := l.matchPredicate(invokedPredicate)
	if err == nil {
		invokedPredicate.Predicate = matchedPredicate
	}
	l.PredicateInvocation = append(l.PredicateInvocation, invokedPredicate)
}

//nolint:all
func (l *CustomQueryListener) EnterPredicate_declaration(ctx *Predicate_declarationContext) {
	if l.State == (State{}) {
		l.State = State{
			isInPredicateDeclaration: true,
		}
	}
	name := ctx.Predicate_name().GetText()
	var params []Parameter
	if ctx.Parameter_list() != nil {
		for _, paramCtx := range ctx.Parameter_list().AllParameter() {
			paramType := paramCtx.Type_().GetText()
			paramName := paramCtx.IDENTIFIER().GetText()
			param := Parameter{
				Name:         paramName,
				Type:         paramType,
				DefaultValue: "",
			}
			params = append(params, param)
		}
	}
	body := ctx.Expression()

	if l.Predicate == nil {
		l.Predicate = []Predicate{}
	}

	l.Predicate = append(l.Predicate, Predicate{
		PredicateName: name,
		Parameter:     params,
		Body:          body.GetText(),
	})
}

//nolint:all
func (l *CustomQueryListener) ExitPredicate_declaration(_ *Predicate_declarationContext) {
	if l.State.isInPredicateDeclaration {
		l.State = State{
			isInPredicateDeclaration: false,
		}
	}
}

func (l *CustomQueryListener) EnterEqualityExpression(ctx *EqualityExpressionContext) {
	if ctx.GetChildCount() > 1 {
		conditionText := ctx.GetText()
		if !l.State.isInPredicateDeclaration {
			l.condition = append(l.condition, conditionText)

			// Create a binary expression node for equality operations
			// We'll simplify this to avoid type issues
			if strings.Contains(conditionText, "==") {
				node := &ExpressionNode{
					Type:     "binary",
					Operator: "==",
				}
				l.currentExpression = append(l.currentExpression, node)
			} else if strings.Contains(conditionText, "!=") {
				node := &ExpressionNode{
					Type:     "binary",
					Operator: "!=",
				}
				l.currentExpression = append(l.currentExpression, node)
			}
		}
	}
}

func (l *CustomQueryListener) ExitEqualityExpression(ctx *EqualityExpressionContext) {
	if ctx.GetChildCount() > 1 && !l.State.isInPredicateDeclaration {
		// Build the expression tree for equality operations
		if len(l.currentExpression) >= 3 {
			// Get the equality node
			eqNode := l.currentExpression[len(l.currentExpression)-3]
			// Set left and right children
			eqNode.Left = l.currentExpression[len(l.currentExpression)-2]
			eqNode.Right = l.currentExpression[len(l.currentExpression)-1]
			// Remove the children from the stack
			l.currentExpression = l.currentExpression[:len(l.currentExpression)-2]
		}
	}
}

func (l *CustomQueryListener) EnterRelationalExpression(ctx *RelationalExpressionContext) {
	if ctx.GetChildCount() > 1 {
		conditionText := ctx.GetText()
		if !l.State.isInPredicateDeclaration {
			l.condition = append(l.condition, conditionText)

			// Create a binary expression node for relational operations
			// We'll simplify this to avoid type issues
			var operator string
			if strings.Contains(conditionText, "<") && !strings.Contains(conditionText, "<=") {
				operator = "<"
			} else if strings.Contains(conditionText, ">") && !strings.Contains(conditionText, ">=") {
				operator = ">"
			} else if strings.Contains(conditionText, "<=") {
				operator = "<="
			} else if strings.Contains(conditionText, ">=") {
				operator = ">="
			} else if strings.Contains(conditionText, " in ") {
				operator = "in"
			}

			if operator != "" {
				node := &ExpressionNode{
					Type:     "binary",
					Operator: operator,
				}
				l.currentExpression = append(l.currentExpression, node)
			}
		}
	}
}

func (l *CustomQueryListener) ExitRelationalExpression(ctx *RelationalExpressionContext) {
	if ctx.GetChildCount() > 1 && !l.State.isInPredicateDeclaration {
		// Build the expression tree for relational operations
		if len(l.currentExpression) >= 3 {
			// Get the relational node
			relNode := l.currentExpression[len(l.currentExpression)-3]
			// Set left and right children
			relNode.Left = l.currentExpression[len(l.currentExpression)-2]
			relNode.Right = l.currentExpression[len(l.currentExpression)-1]
			// Remove the children from the stack
			l.currentExpression = l.currentExpression[:len(l.currentExpression)-2]
		}
	}
}

func (l *CustomQueryListener) EnterPrimary(ctx *PrimaryContext) {
	if !l.State.isInPredicateDeclaration {
		// Handle different types of primary expressions
		if ctx.Operand() != nil {
			// Handle operands (values, variables, method chains)
			operand := ctx.Operand()
			if operand.Value() != nil {
				// Handle literal values
				node := &ExpressionNode{
					Type:  "literal",
					Value: operand.Value().GetText(),
				}
				l.currentExpression = append(l.currentExpression, node)
			} else if operand.Variable() != nil {
				// Handle variables
				node := &ExpressionNode{
					Type:  "variable",
					Value: operand.Variable().GetText(),
				}
				l.currentExpression = append(l.currentExpression, node)
			} else if operand.Method_chain() != nil {
				// Handle method chains
				methodValue := operand.Method_chain().GetText()
				alias := operand.Alias().GetText()
				entity := ""
				for _, selectNode := range l.selectList {
					if selectNode.Alias == alias {
						entity = selectNode.Entity
					}
				}
				node := &ExpressionNode{
					Type:   "method_call",
					Value:  methodValue,
					Entity: entity,
				}
				l.currentExpression = append(l.currentExpression, node)
			}
		} else if ctx.Predicate_invocation() != nil {
			// Handle predicate invocations
			predInvocation := ctx.Predicate_invocation()
			node := &ExpressionNode{
				Type:  "predicate_call",
				Value: predInvocation.GetText(),
			}
			l.currentExpression = append(l.currentExpression, node)
		}
		// We'll skip the parenthesized expression check for now
	}
}

func (l *CustomQueryListener) EnterExpression(ctx *ExpressionContext) {
	if l.expression.Len() > 0 {
		l.expression.WriteString(" ")
	}
	l.expression.WriteString(ctx.GetText())

	// Only build the expression tree for the WHERE clause, not for predicates
	if !l.State.isInPredicateDeclaration && ctx.GetParent() != nil {
		// Check if this is the root expression of the WHERE clause
		parent := ctx.GetParent()
		if _, ok := parent.(*QueryContext); ok {
			// Initialize the expression tree
			l.currentExpression = make([]*ExpressionNode, 0)
		}
	}
}

func (l *CustomQueryListener) ExitExpression(ctx *ExpressionContext) {
	// Only build the expression tree for the WHERE clause, not for predicates
	if !l.State.isInPredicateDeclaration && ctx.GetParent() != nil {
		// Check if this is the root expression of the WHERE clause
		parent := ctx.GetParent()
		if _, ok := parent.(*QueryContext); ok {
			// Set the root of the expression tree
			if len(l.currentExpression) > 0 {
				l.ExpressionTree = l.currentExpression[len(l.currentExpression)-1]

				// Log the expression tree for debugging
				// treeJSON, err := json.MarshalIndent(l.ExpressionTree, "", "  ")
				// if err == nil {
				// 	log.Printf("Expression Tree: %s", string(treeJSON))
				// }
			}
		}
	}
}

func (l *CustomQueryListener) EnterOrExpression(ctx *OrExpressionContext) {
	if ctx.GetChildCount() > 1 && !l.State.isInPredicateDeclaration {
		// Create a binary expression node for OR operation
		node := &ExpressionNode{
			Type:     "binary",
			Operator: "||",
		}
		l.currentExpression = append(l.currentExpression, node)
	}
}

func (l *CustomQueryListener) ExitOrExpression(ctx *OrExpressionContext) {
	if ctx.GetChildCount() > 1 {
		var result strings.Builder
		for i := 0; i < ctx.GetChildCount(); i++ {
			child := ctx.GetChild(i).(antlr.ParseTree) //nolint:all
			if child.GetText() == "||" {
				result.WriteString(" || ")
			} else {
				result.WriteString(child.GetText())
			}
		}
		l.expression.Reset()
		l.expression.WriteString(result.String())

		// Build the expression tree for OR operations
		if !l.State.isInPredicateDeclaration && len(l.currentExpression) >= 3 {
			// Get the OR node
			orNode := l.currentExpression[len(l.currentExpression)-3]
			// Set left and right children
			orNode.Left = l.currentExpression[len(l.currentExpression)-2]
			orNode.Right = l.currentExpression[len(l.currentExpression)-1]
			// Remove the children from the stack
			l.currentExpression = l.currentExpression[:len(l.currentExpression)-2]
		}
	}
}

func (l *CustomQueryListener) EnterAndExpression(ctx *AndExpressionContext) {
	if ctx.GetChildCount() > 1 && !l.State.isInPredicateDeclaration {
		// Create a binary expression node for AND operation
		node := &ExpressionNode{
			Type:     "binary",
			Operator: "&&",
		}
		l.currentExpression = append(l.currentExpression, node)
	}
}

func (l *CustomQueryListener) ExitAndExpression(ctx *AndExpressionContext) {
	if ctx.GetChildCount() > 1 {
		var result strings.Builder
		for i := 0; i < ctx.GetChildCount(); i++ {
			child := ctx.GetChild(i).(antlr.ParseTree) //nolint:all
			if child.GetText() == "&&" {
				result.WriteString(" && ")
			} else {
				result.WriteString(child.GetText())
			}
		}
		l.expression.Reset()
		l.expression.WriteString(result.String())

		// Build the expression tree for AND operations
		if !l.State.isInPredicateDeclaration && len(l.currentExpression) >= 3 {
			// Get the AND node
			andNode := l.currentExpression[len(l.currentExpression)-3]
			// Set left and right children
			andNode.Left = l.currentExpression[len(l.currentExpression)-2]
			andNode.Right = l.currentExpression[len(l.currentExpression)-1]
			// Remove the children from the stack
			l.currentExpression = l.currentExpression[:len(l.currentExpression)-2]
		}
	}
}

func (l *CustomQueryListener) extractArguments(arguments []string) []Parameter {
	args := make([]Parameter, 0, len(arguments))
	for _, argument := range arguments {
		exprType, err := l.inferExpressionType(argument)
		if err != nil {
			continue
		}
		args = append(args, exprType)
	}
	return args
}

func (l *CustomQueryListener) inferExpressionType(argument string) (Parameter, error) {
	argument = strings.TrimSpace(argument)
	for _, entity := range l.selectList {
		if strings.Contains(argument, entity.Alias) {
			return Parameter{
				Name:         entity.Alias,
				Type:         entity.Entity,
				DefaultValue: "",
			}, nil
		}
	}
	return Parameter{}, fmt.Errorf("undefined entity: %s", argument)
}

func (l *CustomQueryListener) matchPredicate(invokedPredicate PredicateInvocation) (Predicate, error) {
	var candidates []Predicate
	for _, pred := range l.Predicate {
		if pred.PredicateName == invokedPredicate.PredicateName {
			candidates = append(candidates, pred)
		}
	}
	if len(candidates) == 0 {
		return Predicate{}, fmt.Errorf("undefined predicate: %s", invokedPredicate.PredicateName)
	}

	var matches []Predicate
	for _, pred := range candidates {
		if len(pred.Parameter) != len(invokedPredicate.Parameter) {
			continue
		}

		// Assume types are compatible unless type checking is required
		compatible := true
		for i := 0; i < len(pred.Parameter); i++ {
			param := pred.Parameter[i]
			argType := invokedPredicate.Parameter[i]
			if param.Type != argType.Type {
				compatible = false
				break
			}
		}

		if compatible {
			matches = append(matches, pred)
		}
	}

	if len(matches) == 0 {
		return Predicate{}, fmt.Errorf("no matching predicate found for %s with given arguments", invokedPredicate.PredicateName)
	} else if len(matches) > 1 {
		// Handle ambiguity if overloading is supported
		return Predicate{}, fmt.Errorf("ambiguous predicate invocation for %s", invokedPredicate.PredicateName)
	}

	return matches[0], nil
}

func ParseQuery(inputQuery string) (Query, error) {
	inputStream := antlr.NewInputStream(inputQuery)
	lexer := NewQueryLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := NewQueryParser(stream)

	errorListener := &customErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(errorListener)

	listener := NewCustomQueryListener()
	tree := p.Query()

	if len(errorListener.errors) > 0 {
		return Query{}, fmt.Errorf("\n%s", strings.Join(errorListener.errors, "\n"))
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	// Log the expression tree for debugging
	// if listener.ExpressionTree != nil {
	// 	treeJSON, err := json.MarshalIndent(listener.ExpressionTree, "", "  ")
	// 	if err == nil {
	// 		log.Printf("Expression Tree: %s", string(treeJSON))
	// 	}
	// }

	return Query{
		Classes:             listener.Classes,
		SelectList:          listener.selectList,
		Expression:          listener.expression.String(),
		Condition:           listener.condition,
		ExpressionTree:      listener.ExpressionTree,
		Predicate:           listener.Predicate,
		PredicateInvocation: listener.PredicateInvocation,
		SelectOutput:        listener.SelectOutput,
	}, nil
}

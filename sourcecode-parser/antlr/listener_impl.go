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

type Query struct {
	SelectList          []SelectList
	Expression          string
	Condition           []string
	Predicate           []Predicate
	PredicateInvocation []PredicateInvocation
	SelectOutput        []SelectOutput
}

type State struct {
	isInPredicateDeclaration bool
}

type CustomQueryListener struct {
	BaseQueryListener
	expression          strings.Builder
	selectList          []SelectList
	condition           []string
	Predicate           []Predicate
	PredicateInvocation []PredicateInvocation
	State               State
	SelectOutput        []SelectOutput
}

type SelectList struct {
	Entity string
	Alias  string
}

type SelectOutput struct {
	SelectEntity string
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
	l.SelectOutput = append(l.SelectOutput, SelectOutput{
		SelectEntity: ctx.GetText(),
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
		}
	}
}

func (l *CustomQueryListener) EnterRelationalExpression(ctx *RelationalExpressionContext) {
	if ctx.GetChildCount() > 1 {
		conditionText := ctx.GetText()
		if !l.State.isInPredicateDeclaration {
			l.condition = append(l.condition, conditionText)
		}
	}
}

func (l *CustomQueryListener) EnterExpression(ctx *ExpressionContext) {
	if l.expression.Len() > 0 {
		l.expression.WriteString(" ")
	}
	l.expression.WriteString(ctx.GetText())
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

	return Query{
		SelectList:          listener.selectList,
		Expression:          listener.expression.String(),
		Condition:           listener.condition,
		Predicate:           listener.Predicate,
		PredicateInvocation: listener.PredicateInvocation,
		SelectOutput:        listener.SelectOutput,
	}, nil
}

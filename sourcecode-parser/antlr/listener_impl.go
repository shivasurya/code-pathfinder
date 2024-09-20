package parser

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

type Predicate struct {
	PredicateName string
	Parameter     []string
	Body          string
}

type Query struct {
	SelectList []SelectList
	Expression string
	Condition  []string
	Predicate  map[string]Predicate
}

type CustomQueryListener struct {
	BaseQueryListener
	expression strings.Builder
	selectList []SelectList
	condition  []string
	Predicate  map[string]Predicate
}

type SelectList struct {
	Entity string
	Alias  string
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
func (l *CustomQueryListener) EnterPredicate_declaration(ctx *Predicate_declarationContext) {
	name := ctx.Predicate_name().GetText()
	params := []string{}
	if ctx.Parameter_list() != nil {
		for _, paramCtx := range ctx.Parameter_list().AllParameter() {
			params = append(params, paramCtx.GetText())
		}
	}
	body := ctx.Expression()

	if l.Predicate == nil {
		l.Predicate = make(map[string]Predicate)
	}

	l.Predicate[name] = Predicate{
		PredicateName: name,
		Parameter:     params,
		Body:          body.GetText(),
	}
}

func (l *CustomQueryListener) EnterEqualityExpression(ctx *EqualityExpressionContext) {
	if ctx.GetChildCount() > 1 {
		conditionText := ctx.GetText()
		l.condition = append(l.condition, conditionText)
	}
}

func (l *CustomQueryListener) EnterRelationalExpression(ctx *RelationalExpressionContext) {
	if ctx.GetChildCount() > 1 {
		conditionText := ctx.GetText()
		l.condition = append(l.condition, conditionText)
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
		SelectList: listener.selectList,
		Expression: listener.expression.String(),
		Condition:  listener.condition,
		Predicate:  listener.Predicate,
	}, nil
}

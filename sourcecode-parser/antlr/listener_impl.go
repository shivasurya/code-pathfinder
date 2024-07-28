package parser

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"strings"
)

type Query struct {
	SelectList []string
	Expression string
}

type CustomQueryListener struct {
	BaseQueryListener
	expression strings.Builder
	selectList []string
}

func NewCustomQueryListener() *CustomQueryListener {
	return &CustomQueryListener{
		BaseQueryListener: BaseQueryListener{},
	}
}

func (l *CustomQueryListener) EnterSelect_list(ctx *Select_listContext) {
	// get select list
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i).(antlr.ParseTree)
		if _, ok := child.(ISelect_itemContext); ok {
			l.selectList = append(l.selectList, child.GetText())
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
			child := ctx.GetChild(i).(antlr.ParseTree)
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
			child := ctx.GetChild(i).(antlr.ParseTree)
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

func ParseQuery(inputQuery string) Query {
	inputStream := antlr.NewInputStream(inputQuery)
	lexer := NewQueryLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := NewQueryParser(stream)

	listener := NewCustomQueryListener()
	antlr.ParseTreeWalkerDefault.Walk(listener, p.Query())

	fmt.Println(listener.selectList)

	return Query{SelectList: listener.selectList, Expression: listener.expression.String()}
}

package main

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"github.com/expr-lang/expr"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"strings"
)

// QueryListener is a complete listener for a parse tree produced by QueryParser.
type QueryListener struct {
	*parser.BaseQueryListener
	expression strings.Builder
}

func NewQueryListener() *QueryListener {
	return &QueryListener{
		BaseQueryListener: &parser.BaseQueryListener{},
	}
}

func (l *QueryListener) EnterExpression(ctx *parser.ExpressionContext) {
	if l.expression.Len() > 0 {
		l.expression.WriteString(" ")
	}
	l.expression.WriteString(ctx.GetText())
}

func (l *QueryListener) ExitOrExpression(ctx *parser.OrExpressionContext) {
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

func (l *QueryListener) ExitAndExpression(ctx *parser.AndExpressionContext) {
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

func parseQuery(inputQuery string) string {
	inputStream := antlr.NewInputStream(inputQuery)
	lexer := parser.NewQueryLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewQueryParser(stream)

	listener := NewQueryListener()
	antlr.ParseTreeWalkerDefault.Walk(listener, p.Query())

	return listener.expression.String()
}

func main() {
	inputQuery := `FIND method_declaration AS md WHERE (md.getName() == "onCreate" || md.getVisibility() == "public") && md.getReturnType() != "void"`
	expression := parseQuery(inputQuery)
	// string replace "md." with ""
	expression = strings.Replace(expression, "md.", "", -1)
	fmt.Println(expression)
	env := map[string]interface{}{
		"getName": func() string {
			return "onCreate"
		},
		"getVisibility": func() string {
			return "public"
		},
		"getReturnType": func() string {
			return "voids"
		},
	}
	program, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		fmt.Println(err)
	}
	output, err := expr.Run(program, env)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(output)
}

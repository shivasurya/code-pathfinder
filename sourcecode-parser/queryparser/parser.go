package queryparser

import "fmt"

type Parser struct {
	l         *Lexer
	errors    []string
	curToken  Token
	peekToken Token
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}
	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) nextToken() {
	p.peekToken = p.l.NextToken()
	p.curToken = p.peekToken
}

func (p *Parser) parseExpression() Expr {
	expr := p.parseLogicalOr() // Start with the lowest precedence
	return expr
}

func (p *Parser) parseLogicalOr() Expr {
	expr := p.parseLogicalAnd()
	for p.curToken.Type == KEYWORD && p.peekToken.Literal == "OR" {
		p.nextToken()
		right := p.parseLogicalAnd()
		expr = &BinaryExpr{
			Left:  expr,
			Op:    "OR",
			Right: right,
		}
	}
	return expr
}

func (p *Parser) parseLogicalAnd() Expr {
	expr := p.parseGroup()
	for p.curToken.Type == KEYWORD && p.peekToken.Literal == "AND" {
		p.nextToken()
		right := p.parseGroup()
		expr = &BinaryExpr{
			Left:  expr,
			Op:    "AND",
			Right: right,
		}
	}
	return expr
}

func (p *Parser) parseGroup() Expr {
	if p.curToken.Type == LPAREN {
		p.nextToken()               // Skip '('
		expr := p.parseExpression() // Parse expression within parentheses
		if p.curToken.Type != RPAREN {
			p.peekError(p.curToken.Type)
			return nil
		}
		p.nextToken() // Skip ')'
		return expr
	}
	return p.parseCondition() // Parse a basic condition
}

func (p *Parser) parseCondition() *Condition {
	if p.curToken.Type != IDENT {
		p.peekError(IDENT)
		return nil
	}

	field := p.curToken.Literal
	p.nextToken()

	if p.curToken.Type != OPERATOR {
		p.peekError(OPERATOR)
		return nil
	}

	operator := p.curToken.Literal
	p.nextToken()

	if p.curToken.Type != STRING && p.curToken.Type != IDENT {
		p.peekError(STRING)
		return nil
	}
	value := p.curToken.Literal
	p.nextToken() // move past the value

	fmt.Println(field, operator, value)

	return &Condition{Field: field, Operator: operator, Value: value}
}

type EvalContext interface {
	GetValue(key string) string // Retrieves a value based on a key, which helps in condition evaluation.
}

type Expr interface {
	Evaluate(ctx EvalContext) bool
}

type BinaryExpr struct {
	Left  Expr
	Right Expr
	Op    string
}

func (p *Parser) ParseQuery() *Query {
	// fmt.Printf("Current token: %s\n", p.curToken.Literal) // Debug output

	query := &Query{}

	if p.curToken.Type != KEYWORD || p.curToken.Literal != "FIND" {
		p.peekError(KEYWORD)
		return nil
	}

	query.Operation = p.curToken.Literal
	p.nextToken()

	if p.curToken.Type != IDENT {
		p.peekError(IDENT)
		return nil
	}

	query.Entity = p.curToken.Literal
	p.nextToken()

	if p.curToken.Type != KEYWORD || p.curToken.Literal != "WHERE" {
		p.peekError(KEYWORD)
		return nil
	}

	p.nextToken()
	expr := p.parseExpression()
	if expr == nil {
		// handle error or invalid expression
		return nil
	}
	query.Conditions = expr

	return query
}

func (b *BinaryExpr) Evaluate(ctx EvalContext) bool {
	switch b.Op {
	case "AND":
		return b.Left.Evaluate(ctx) && b.Right.Evaluate(ctx)
	case "OR":
		return b.Left.Evaluate(ctx) || b.Right.Evaluate(ctx)
	}
	return false
}

func (c *Condition) Evaluate(ctx EvalContext) bool {
	fmt.Printf("Evaluating condition: %s %s %s\n", c.Field, c.Operator, c.Value) // Debug output
	fieldValue := ctx.GetValue(c.Field)
	switch c.Operator {
	case "=":
		return fieldValue == c.Value
	case "!=":
		return fieldValue != c.Value
	}
	return false
}

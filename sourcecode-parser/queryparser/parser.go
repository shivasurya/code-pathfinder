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

	// Process conditions
	for p.curToken.Type != EOF {
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

		if p.curToken.Type != STRING {
			p.peekError(STRING)
			return nil
		}

		value := p.curToken.Literal
		cond := Condition{Field: field, Operator: operator, Value: value}
		query.Conditions = append(query.Conditions, cond)

		p.nextToken() // move to the next part of the condition or EOF

		// Handle AND/OR for multiple conditions
		if p.curToken.Type == KEYWORD && (p.curToken.Literal == "AND" || p.curToken.Literal == "OR") {
			p.nextToken() // Continue to next condition
		}
	}

	return query
}

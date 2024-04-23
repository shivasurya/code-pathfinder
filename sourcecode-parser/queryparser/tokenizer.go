package queryparser

import "fmt"

// Token represents a lexical token.
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	WS
	// Special tokens
	IDENT    // Identifiers
	STRING   // String literals
	KEYWORD  // Keywords such as FIND, WHERE
	OPERATOR // Operators such as =, INCLUDES, MATCHES
	LPAREN   // Left parenthesis '('
	RPAREN
)

type Token struct {
	Type    TokenType
	Literal string
}

type Node interface {
	node()
}

type Query struct {
	Operation  string
	Entity     string
	Conditions Expr
}

type Condition struct {
	Field    string
	Operator string
	Value    string
}

func (q Query) node()     {}
func (c Condition) node() {}

// Define keywords
var keywords = map[string]TokenType{
	"FIND":  KEYWORD,
	"WHERE": KEYWORD,
	"AND":   KEYWORD,
	"OR":    KEYWORD,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case KEYWORD:
		return "KEYWORD"
	case STRING:
		return "STRING"
	case OPERATOR:
		return "OPERATOR"
	case WS:
		return "WS"
	case ILLEGAL:
		return "ILLEGAL"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(t))
	}
}

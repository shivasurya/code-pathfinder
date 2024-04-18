package main

import (
	"queryparser"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseQuery(t *testing.T) {
	input := "FIND class WHERE name = 'MyClass' AND visibility = 'public' AND method = 'main'"

	lex := queryparser.NewLexer(input)
	pars := queryparser.NewParser(lex)

	query := pars.ParseQuery()

	assert.NotNil(t, query)
	assert.Equal(t, "MyClass", query.Conditions[0].Value)
}

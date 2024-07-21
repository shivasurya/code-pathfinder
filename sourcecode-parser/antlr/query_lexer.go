// Code generated from Query.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"sync"
	"unicode"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type QueryLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var QueryLexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	ChannelNames           []string
	ModeNames              []string
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func querylexerLexerInit() {
	staticData := &QueryLexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'FIND'", "'WHERE'", "','", "'AS'", "'OR'", "'AND'", "'NOT'", "'('",
		"')'", "'.'", "'='", "'!='", "'<'", "'>'", "'<='", "'>='", "'LIKE'",
		"'IN'",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "",
		"", "", "STRING", "STRING_WITH_WILDCARD", "NUMBER", "IDENTIFIER", "WS",
	}
	staticData.RuleNames = []string{
		"T__0", "T__1", "T__2", "T__3", "T__4", "T__5", "T__6", "T__7", "T__8",
		"T__9", "T__10", "T__11", "T__12", "T__13", "T__14", "T__15", "T__16",
		"T__17", "STRING", "STRING_WITH_WILDCARD", "NUMBER", "IDENTIFIER", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 23, 153, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1,
		4, 1, 5, 1, 5, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 8, 1,
		8, 1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 1, 12, 1, 12, 1, 13,
		1, 13, 1, 14, 1, 14, 1, 14, 1, 15, 1, 15, 1, 15, 1, 16, 1, 16, 1, 16, 1,
		16, 1, 16, 1, 17, 1, 17, 1, 17, 1, 18, 1, 18, 1, 18, 1, 18, 5, 18, 108,
		8, 18, 10, 18, 12, 18, 111, 9, 18, 1, 18, 1, 18, 1, 19, 1, 19, 1, 19, 1,
		19, 1, 19, 5, 19, 120, 8, 19, 10, 19, 12, 19, 123, 9, 19, 1, 19, 1, 19,
		1, 20, 4, 20, 128, 8, 20, 11, 20, 12, 20, 129, 1, 20, 1, 20, 4, 20, 134,
		8, 20, 11, 20, 12, 20, 135, 3, 20, 138, 8, 20, 1, 21, 1, 21, 5, 21, 142,
		8, 21, 10, 21, 12, 21, 145, 9, 21, 1, 22, 4, 22, 148, 8, 22, 11, 22, 12,
		22, 149, 1, 22, 1, 22, 0, 0, 23, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13,
		7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25, 13, 27, 14, 29, 15, 31, 16,
		33, 17, 35, 18, 37, 19, 39, 20, 41, 21, 43, 22, 45, 23, 1, 0, 5, 2, 0,
		34, 34, 92, 92, 1, 0, 48, 57, 3, 0, 65, 90, 95, 95, 97, 122, 4, 0, 48,
		57, 65, 90, 95, 95, 97, 122, 3, 0, 9, 10, 13, 13, 32, 32, 162, 0, 1, 1,
		0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1,
		0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17,
		1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0,
		25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0,
		0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0, 0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0, 0,
		0, 0, 41, 1, 0, 0, 0, 0, 43, 1, 0, 0, 0, 0, 45, 1, 0, 0, 0, 1, 47, 1, 0,
		0, 0, 3, 52, 1, 0, 0, 0, 5, 58, 1, 0, 0, 0, 7, 60, 1, 0, 0, 0, 9, 63, 1,
		0, 0, 0, 11, 66, 1, 0, 0, 0, 13, 70, 1, 0, 0, 0, 15, 74, 1, 0, 0, 0, 17,
		76, 1, 0, 0, 0, 19, 78, 1, 0, 0, 0, 21, 80, 1, 0, 0, 0, 23, 82, 1, 0, 0,
		0, 25, 85, 1, 0, 0, 0, 27, 87, 1, 0, 0, 0, 29, 89, 1, 0, 0, 0, 31, 92,
		1, 0, 0, 0, 33, 95, 1, 0, 0, 0, 35, 100, 1, 0, 0, 0, 37, 103, 1, 0, 0,
		0, 39, 114, 1, 0, 0, 0, 41, 127, 1, 0, 0, 0, 43, 139, 1, 0, 0, 0, 45, 147,
		1, 0, 0, 0, 47, 48, 5, 70, 0, 0, 48, 49, 5, 73, 0, 0, 49, 50, 5, 78, 0,
		0, 50, 51, 5, 68, 0, 0, 51, 2, 1, 0, 0, 0, 52, 53, 5, 87, 0, 0, 53, 54,
		5, 72, 0, 0, 54, 55, 5, 69, 0, 0, 55, 56, 5, 82, 0, 0, 56, 57, 5, 69, 0,
		0, 57, 4, 1, 0, 0, 0, 58, 59, 5, 44, 0, 0, 59, 6, 1, 0, 0, 0, 60, 61, 5,
		65, 0, 0, 61, 62, 5, 83, 0, 0, 62, 8, 1, 0, 0, 0, 63, 64, 5, 79, 0, 0,
		64, 65, 5, 82, 0, 0, 65, 10, 1, 0, 0, 0, 66, 67, 5, 65, 0, 0, 67, 68, 5,
		78, 0, 0, 68, 69, 5, 68, 0, 0, 69, 12, 1, 0, 0, 0, 70, 71, 5, 78, 0, 0,
		71, 72, 5, 79, 0, 0, 72, 73, 5, 84, 0, 0, 73, 14, 1, 0, 0, 0, 74, 75, 5,
		40, 0, 0, 75, 16, 1, 0, 0, 0, 76, 77, 5, 41, 0, 0, 77, 18, 1, 0, 0, 0,
		78, 79, 5, 46, 0, 0, 79, 20, 1, 0, 0, 0, 80, 81, 5, 61, 0, 0, 81, 22, 1,
		0, 0, 0, 82, 83, 5, 33, 0, 0, 83, 84, 5, 61, 0, 0, 84, 24, 1, 0, 0, 0,
		85, 86, 5, 60, 0, 0, 86, 26, 1, 0, 0, 0, 87, 88, 5, 62, 0, 0, 88, 28, 1,
		0, 0, 0, 89, 90, 5, 60, 0, 0, 90, 91, 5, 61, 0, 0, 91, 30, 1, 0, 0, 0,
		92, 93, 5, 62, 0, 0, 93, 94, 5, 61, 0, 0, 94, 32, 1, 0, 0, 0, 95, 96, 5,
		76, 0, 0, 96, 97, 5, 73, 0, 0, 97, 98, 5, 75, 0, 0, 98, 99, 5, 69, 0, 0,
		99, 34, 1, 0, 0, 0, 100, 101, 5, 73, 0, 0, 101, 102, 5, 78, 0, 0, 102,
		36, 1, 0, 0, 0, 103, 109, 5, 34, 0, 0, 104, 108, 8, 0, 0, 0, 105, 106,
		5, 92, 0, 0, 106, 108, 9, 0, 0, 0, 107, 104, 1, 0, 0, 0, 107, 105, 1, 0,
		0, 0, 108, 111, 1, 0, 0, 0, 109, 107, 1, 0, 0, 0, 109, 110, 1, 0, 0, 0,
		110, 112, 1, 0, 0, 0, 111, 109, 1, 0, 0, 0, 112, 113, 5, 34, 0, 0, 113,
		38, 1, 0, 0, 0, 114, 121, 5, 34, 0, 0, 115, 120, 8, 0, 0, 0, 116, 117,
		5, 92, 0, 0, 117, 120, 9, 0, 0, 0, 118, 120, 5, 37, 0, 0, 119, 115, 1,
		0, 0, 0, 119, 116, 1, 0, 0, 0, 119, 118, 1, 0, 0, 0, 120, 123, 1, 0, 0,
		0, 121, 119, 1, 0, 0, 0, 121, 122, 1, 0, 0, 0, 122, 124, 1, 0, 0, 0, 123,
		121, 1, 0, 0, 0, 124, 125, 5, 34, 0, 0, 125, 40, 1, 0, 0, 0, 126, 128,
		7, 1, 0, 0, 127, 126, 1, 0, 0, 0, 128, 129, 1, 0, 0, 0, 129, 127, 1, 0,
		0, 0, 129, 130, 1, 0, 0, 0, 130, 137, 1, 0, 0, 0, 131, 133, 5, 46, 0, 0,
		132, 134, 7, 1, 0, 0, 133, 132, 1, 0, 0, 0, 134, 135, 1, 0, 0, 0, 135,
		133, 1, 0, 0, 0, 135, 136, 1, 0, 0, 0, 136, 138, 1, 0, 0, 0, 137, 131,
		1, 0, 0, 0, 137, 138, 1, 0, 0, 0, 138, 42, 1, 0, 0, 0, 139, 143, 7, 2,
		0, 0, 140, 142, 7, 3, 0, 0, 141, 140, 1, 0, 0, 0, 142, 145, 1, 0, 0, 0,
		143, 141, 1, 0, 0, 0, 143, 144, 1, 0, 0, 0, 144, 44, 1, 0, 0, 0, 145, 143,
		1, 0, 0, 0, 146, 148, 7, 4, 0, 0, 147, 146, 1, 0, 0, 0, 148, 149, 1, 0,
		0, 0, 149, 147, 1, 0, 0, 0, 149, 150, 1, 0, 0, 0, 150, 151, 1, 0, 0, 0,
		151, 152, 6, 22, 0, 0, 152, 46, 1, 0, 0, 0, 10, 0, 107, 109, 119, 121,
		129, 135, 137, 143, 149, 1, 6, 0, 0,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// QueryLexerInit initializes any static state used to implement QueryLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewQueryLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func QueryLexerInit() {
	staticData := &QueryLexerLexerStaticData
	staticData.once.Do(querylexerLexerInit)
}

// NewQueryLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewQueryLexer(input antlr.CharStream) *QueryLexer {
	QueryLexerInit()
	l := new(QueryLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &QueryLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "Query.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// QueryLexer tokens.
const (
	QueryLexerT__0                 = 1
	QueryLexerT__1                 = 2
	QueryLexerT__2                 = 3
	QueryLexerT__3                 = 4
	QueryLexerT__4                 = 5
	QueryLexerT__5                 = 6
	QueryLexerT__6                 = 7
	QueryLexerT__7                 = 8
	QueryLexerT__8                 = 9
	QueryLexerT__9                 = 10
	QueryLexerT__10                = 11
	QueryLexerT__11                = 12
	QueryLexerT__12                = 13
	QueryLexerT__13                = 14
	QueryLexerT__14                = 15
	QueryLexerT__15                = 16
	QueryLexerT__16                = 17
	QueryLexerT__17                = 18
	QueryLexerSTRING               = 19
	QueryLexerSTRING_WITH_WILDCARD = 20
	QueryLexerNUMBER               = 21
	QueryLexerIDENTIFIER           = 22
	QueryLexerWS                   = 23
)

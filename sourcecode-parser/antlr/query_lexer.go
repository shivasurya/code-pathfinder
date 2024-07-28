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
		"", "'FIND'", "'WHERE'", "','", "'AS'", "'||'", "'&&'", "'('", "')'",
		"'.'", "'=='", "'!='", "'<'", "'>'", "'<='", "'>='", "'LIKE'", "'IN'",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "",
		"", "STRING", "STRING_WITH_WILDCARD", "NUMBER", "IDENTIFIER", "WS",
	}
	staticData.RuleNames = []string{
		"T__0", "T__1", "T__2", "T__3", "T__4", "T__5", "T__6", "T__7", "T__8",
		"T__9", "T__10", "T__11", "T__12", "T__13", "T__14", "T__15", "T__16",
		"STRING", "STRING_WITH_WILDCARD", "NUMBER", "IDENTIFIER", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 22, 147, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5,
		1, 5, 1, 6, 1, 6, 1, 7, 1, 7, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9, 1, 10, 1, 10,
		1, 10, 1, 11, 1, 11, 1, 12, 1, 12, 1, 13, 1, 13, 1, 13, 1, 14, 1, 14, 1,
		14, 1, 15, 1, 15, 1, 15, 1, 15, 1, 15, 1, 16, 1, 16, 1, 16, 1, 17, 1, 17,
		1, 17, 1, 17, 5, 17, 102, 8, 17, 10, 17, 12, 17, 105, 9, 17, 1, 17, 1,
		17, 1, 18, 1, 18, 1, 18, 1, 18, 1, 18, 5, 18, 114, 8, 18, 10, 18, 12, 18,
		117, 9, 18, 1, 18, 1, 18, 1, 19, 4, 19, 122, 8, 19, 11, 19, 12, 19, 123,
		1, 19, 1, 19, 4, 19, 128, 8, 19, 11, 19, 12, 19, 129, 3, 19, 132, 8, 19,
		1, 20, 1, 20, 5, 20, 136, 8, 20, 10, 20, 12, 20, 139, 9, 20, 1, 21, 4,
		21, 142, 8, 21, 11, 21, 12, 21, 143, 1, 21, 1, 21, 0, 0, 22, 1, 1, 3, 2,
		5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25,
		13, 27, 14, 29, 15, 31, 16, 33, 17, 35, 18, 37, 19, 39, 20, 41, 21, 43,
		22, 1, 0, 5, 2, 0, 34, 34, 92, 92, 1, 0, 48, 57, 3, 0, 65, 90, 95, 95,
		97, 122, 4, 0, 48, 57, 65, 90, 95, 95, 97, 122, 3, 0, 9, 10, 13, 13, 32,
		32, 156, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1,
		0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15,
		1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0,
		23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0,
		0, 31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0, 0, 0, 37, 1, 0, 0,
		0, 0, 39, 1, 0, 0, 0, 0, 41, 1, 0, 0, 0, 0, 43, 1, 0, 0, 0, 1, 45, 1, 0,
		0, 0, 3, 50, 1, 0, 0, 0, 5, 56, 1, 0, 0, 0, 7, 58, 1, 0, 0, 0, 9, 61, 1,
		0, 0, 0, 11, 64, 1, 0, 0, 0, 13, 67, 1, 0, 0, 0, 15, 69, 1, 0, 0, 0, 17,
		71, 1, 0, 0, 0, 19, 73, 1, 0, 0, 0, 21, 76, 1, 0, 0, 0, 23, 79, 1, 0, 0,
		0, 25, 81, 1, 0, 0, 0, 27, 83, 1, 0, 0, 0, 29, 86, 1, 0, 0, 0, 31, 89,
		1, 0, 0, 0, 33, 94, 1, 0, 0, 0, 35, 97, 1, 0, 0, 0, 37, 108, 1, 0, 0, 0,
		39, 121, 1, 0, 0, 0, 41, 133, 1, 0, 0, 0, 43, 141, 1, 0, 0, 0, 45, 46,
		5, 70, 0, 0, 46, 47, 5, 73, 0, 0, 47, 48, 5, 78, 0, 0, 48, 49, 5, 68, 0,
		0, 49, 2, 1, 0, 0, 0, 50, 51, 5, 87, 0, 0, 51, 52, 5, 72, 0, 0, 52, 53,
		5, 69, 0, 0, 53, 54, 5, 82, 0, 0, 54, 55, 5, 69, 0, 0, 55, 4, 1, 0, 0,
		0, 56, 57, 5, 44, 0, 0, 57, 6, 1, 0, 0, 0, 58, 59, 5, 65, 0, 0, 59, 60,
		5, 83, 0, 0, 60, 8, 1, 0, 0, 0, 61, 62, 5, 124, 0, 0, 62, 63, 5, 124, 0,
		0, 63, 10, 1, 0, 0, 0, 64, 65, 5, 38, 0, 0, 65, 66, 5, 38, 0, 0, 66, 12,
		1, 0, 0, 0, 67, 68, 5, 40, 0, 0, 68, 14, 1, 0, 0, 0, 69, 70, 5, 41, 0,
		0, 70, 16, 1, 0, 0, 0, 71, 72, 5, 46, 0, 0, 72, 18, 1, 0, 0, 0, 73, 74,
		5, 61, 0, 0, 74, 75, 5, 61, 0, 0, 75, 20, 1, 0, 0, 0, 76, 77, 5, 33, 0,
		0, 77, 78, 5, 61, 0, 0, 78, 22, 1, 0, 0, 0, 79, 80, 5, 60, 0, 0, 80, 24,
		1, 0, 0, 0, 81, 82, 5, 62, 0, 0, 82, 26, 1, 0, 0, 0, 83, 84, 5, 60, 0,
		0, 84, 85, 5, 61, 0, 0, 85, 28, 1, 0, 0, 0, 86, 87, 5, 62, 0, 0, 87, 88,
		5, 61, 0, 0, 88, 30, 1, 0, 0, 0, 89, 90, 5, 76, 0, 0, 90, 91, 5, 73, 0,
		0, 91, 92, 5, 75, 0, 0, 92, 93, 5, 69, 0, 0, 93, 32, 1, 0, 0, 0, 94, 95,
		5, 73, 0, 0, 95, 96, 5, 78, 0, 0, 96, 34, 1, 0, 0, 0, 97, 103, 5, 34, 0,
		0, 98, 102, 8, 0, 0, 0, 99, 100, 5, 92, 0, 0, 100, 102, 9, 0, 0, 0, 101,
		98, 1, 0, 0, 0, 101, 99, 1, 0, 0, 0, 102, 105, 1, 0, 0, 0, 103, 101, 1,
		0, 0, 0, 103, 104, 1, 0, 0, 0, 104, 106, 1, 0, 0, 0, 105, 103, 1, 0, 0,
		0, 106, 107, 5, 34, 0, 0, 107, 36, 1, 0, 0, 0, 108, 115, 5, 34, 0, 0, 109,
		114, 8, 0, 0, 0, 110, 111, 5, 92, 0, 0, 111, 114, 9, 0, 0, 0, 112, 114,
		5, 37, 0, 0, 113, 109, 1, 0, 0, 0, 113, 110, 1, 0, 0, 0, 113, 112, 1, 0,
		0, 0, 114, 117, 1, 0, 0, 0, 115, 113, 1, 0, 0, 0, 115, 116, 1, 0, 0, 0,
		116, 118, 1, 0, 0, 0, 117, 115, 1, 0, 0, 0, 118, 119, 5, 34, 0, 0, 119,
		38, 1, 0, 0, 0, 120, 122, 7, 1, 0, 0, 121, 120, 1, 0, 0, 0, 122, 123, 1,
		0, 0, 0, 123, 121, 1, 0, 0, 0, 123, 124, 1, 0, 0, 0, 124, 131, 1, 0, 0,
		0, 125, 127, 5, 46, 0, 0, 126, 128, 7, 1, 0, 0, 127, 126, 1, 0, 0, 0, 128,
		129, 1, 0, 0, 0, 129, 127, 1, 0, 0, 0, 129, 130, 1, 0, 0, 0, 130, 132,
		1, 0, 0, 0, 131, 125, 1, 0, 0, 0, 131, 132, 1, 0, 0, 0, 132, 40, 1, 0,
		0, 0, 133, 137, 7, 2, 0, 0, 134, 136, 7, 3, 0, 0, 135, 134, 1, 0, 0, 0,
		136, 139, 1, 0, 0, 0, 137, 135, 1, 0, 0, 0, 137, 138, 1, 0, 0, 0, 138,
		42, 1, 0, 0, 0, 139, 137, 1, 0, 0, 0, 140, 142, 7, 4, 0, 0, 141, 140, 1,
		0, 0, 0, 142, 143, 1, 0, 0, 0, 143, 141, 1, 0, 0, 0, 143, 144, 1, 0, 0,
		0, 144, 145, 1, 0, 0, 0, 145, 146, 6, 21, 0, 0, 146, 44, 1, 0, 0, 0, 10,
		0, 101, 103, 113, 115, 123, 129, 131, 137, 143, 1, 6, 0, 0,
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
	QueryLexerSTRING               = 18
	QueryLexerSTRING_WITH_WILDCARD = 19
	QueryLexerNUMBER               = 20
	QueryLexerIDENTIFIER           = 21
	QueryLexerWS                   = 22
)

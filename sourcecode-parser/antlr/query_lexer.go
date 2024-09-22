// Code generated from Query.g4 by ANTLR 4.13.2. DO NOT EDIT.

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
		"", "'('", "')'", "'{'", "'}'", "','", "'||'", "'&&'", "'=='", "'!='",
		"'<'", "'>'", "'<='", "'>='", "'+'", "'-'", "'*'", "'/'", "'!'", "'.'",
		"'['", "']'", "'LIKE'", "'in'", "", "", "", "'predicate'", "'FROM'",
		"'WHERE'", "'AS'", "'SELECT'",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "",
		"", "", "", "", "", "", "", "STRING", "STRING_WITH_WILDCARD", "NUMBER",
		"PREDICATE", "FROM", "WHERE", "AS", "SELECT", "IDENTIFIER", "WS",
	}
	staticData.RuleNames = []string{
		"T__0", "T__1", "T__2", "T__3", "T__4", "T__5", "T__6", "T__7", "T__8",
		"T__9", "T__10", "T__11", "T__12", "T__13", "T__14", "T__15", "T__16",
		"T__17", "T__18", "T__19", "T__20", "T__21", "T__22", "STRING", "STRING_WITH_WILDCARD",
		"NUMBER", "PREDICATE", "FROM", "WHERE", "AS", "SELECT", "IDENTIFIER",
		"WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 33, 204, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2,
		31, 7, 31, 2, 32, 7, 32, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1, 2, 1, 3, 1, 3,
		1, 4, 1, 4, 1, 5, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 1, 8,
		1, 8, 1, 8, 1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 1, 12, 1, 12,
		1, 12, 1, 13, 1, 13, 1, 14, 1, 14, 1, 15, 1, 15, 1, 16, 1, 16, 1, 17, 1,
		17, 1, 18, 1, 18, 1, 19, 1, 19, 1, 20, 1, 20, 1, 21, 1, 21, 1, 21, 1, 21,
		1, 21, 1, 22, 1, 22, 1, 22, 1, 23, 1, 23, 1, 23, 1, 23, 5, 23, 128, 8,
		23, 10, 23, 12, 23, 131, 9, 23, 1, 23, 1, 23, 1, 24, 1, 24, 1, 24, 1, 24,
		1, 24, 5, 24, 140, 8, 24, 10, 24, 12, 24, 143, 9, 24, 1, 24, 1, 24, 1,
		25, 4, 25, 148, 8, 25, 11, 25, 12, 25, 149, 1, 25, 1, 25, 4, 25, 154, 8,
		25, 11, 25, 12, 25, 155, 3, 25, 158, 8, 25, 1, 26, 1, 26, 1, 26, 1, 26,
		1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 27, 1, 27, 1, 27, 1, 27, 1,
		27, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 29, 1, 29, 1, 29, 1, 30,
		1, 30, 1, 30, 1, 30, 1, 30, 1, 30, 1, 30, 1, 31, 1, 31, 5, 31, 193, 8,
		31, 10, 31, 12, 31, 196, 9, 31, 1, 32, 4, 32, 199, 8, 32, 11, 32, 12, 32,
		200, 1, 32, 1, 32, 0, 0, 33, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13, 7,
		15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25, 13, 27, 14, 29, 15, 31, 16, 33,
		17, 35, 18, 37, 19, 39, 20, 41, 21, 43, 22, 45, 23, 47, 24, 49, 25, 51,
		26, 53, 27, 55, 28, 57, 29, 59, 30, 61, 31, 63, 32, 65, 33, 1, 0, 5, 2,
		0, 34, 34, 92, 92, 1, 0, 48, 57, 3, 0, 65, 90, 95, 95, 97, 122, 4, 0, 48,
		57, 65, 90, 95, 95, 97, 122, 3, 0, 9, 10, 13, 13, 32, 32, 213, 0, 1, 1,
		0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1,
		0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17,
		1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0,
		25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0,
		0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0, 0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0, 0,
		0, 0, 41, 1, 0, 0, 0, 0, 43, 1, 0, 0, 0, 0, 45, 1, 0, 0, 0, 0, 47, 1, 0,
		0, 0, 0, 49, 1, 0, 0, 0, 0, 51, 1, 0, 0, 0, 0, 53, 1, 0, 0, 0, 0, 55, 1,
		0, 0, 0, 0, 57, 1, 0, 0, 0, 0, 59, 1, 0, 0, 0, 0, 61, 1, 0, 0, 0, 0, 63,
		1, 0, 0, 0, 0, 65, 1, 0, 0, 0, 1, 67, 1, 0, 0, 0, 3, 69, 1, 0, 0, 0, 5,
		71, 1, 0, 0, 0, 7, 73, 1, 0, 0, 0, 9, 75, 1, 0, 0, 0, 11, 77, 1, 0, 0,
		0, 13, 80, 1, 0, 0, 0, 15, 83, 1, 0, 0, 0, 17, 86, 1, 0, 0, 0, 19, 89,
		1, 0, 0, 0, 21, 91, 1, 0, 0, 0, 23, 93, 1, 0, 0, 0, 25, 96, 1, 0, 0, 0,
		27, 99, 1, 0, 0, 0, 29, 101, 1, 0, 0, 0, 31, 103, 1, 0, 0, 0, 33, 105,
		1, 0, 0, 0, 35, 107, 1, 0, 0, 0, 37, 109, 1, 0, 0, 0, 39, 111, 1, 0, 0,
		0, 41, 113, 1, 0, 0, 0, 43, 115, 1, 0, 0, 0, 45, 120, 1, 0, 0, 0, 47, 123,
		1, 0, 0, 0, 49, 134, 1, 0, 0, 0, 51, 147, 1, 0, 0, 0, 53, 159, 1, 0, 0,
		0, 55, 169, 1, 0, 0, 0, 57, 174, 1, 0, 0, 0, 59, 180, 1, 0, 0, 0, 61, 183,
		1, 0, 0, 0, 63, 190, 1, 0, 0, 0, 65, 198, 1, 0, 0, 0, 67, 68, 5, 40, 0,
		0, 68, 2, 1, 0, 0, 0, 69, 70, 5, 41, 0, 0, 70, 4, 1, 0, 0, 0, 71, 72, 5,
		123, 0, 0, 72, 6, 1, 0, 0, 0, 73, 74, 5, 125, 0, 0, 74, 8, 1, 0, 0, 0,
		75, 76, 5, 44, 0, 0, 76, 10, 1, 0, 0, 0, 77, 78, 5, 124, 0, 0, 78, 79,
		5, 124, 0, 0, 79, 12, 1, 0, 0, 0, 80, 81, 5, 38, 0, 0, 81, 82, 5, 38, 0,
		0, 82, 14, 1, 0, 0, 0, 83, 84, 5, 61, 0, 0, 84, 85, 5, 61, 0, 0, 85, 16,
		1, 0, 0, 0, 86, 87, 5, 33, 0, 0, 87, 88, 5, 61, 0, 0, 88, 18, 1, 0, 0,
		0, 89, 90, 5, 60, 0, 0, 90, 20, 1, 0, 0, 0, 91, 92, 5, 62, 0, 0, 92, 22,
		1, 0, 0, 0, 93, 94, 5, 60, 0, 0, 94, 95, 5, 61, 0, 0, 95, 24, 1, 0, 0,
		0, 96, 97, 5, 62, 0, 0, 97, 98, 5, 61, 0, 0, 98, 26, 1, 0, 0, 0, 99, 100,
		5, 43, 0, 0, 100, 28, 1, 0, 0, 0, 101, 102, 5, 45, 0, 0, 102, 30, 1, 0,
		0, 0, 103, 104, 5, 42, 0, 0, 104, 32, 1, 0, 0, 0, 105, 106, 5, 47, 0, 0,
		106, 34, 1, 0, 0, 0, 107, 108, 5, 33, 0, 0, 108, 36, 1, 0, 0, 0, 109, 110,
		5, 46, 0, 0, 110, 38, 1, 0, 0, 0, 111, 112, 5, 91, 0, 0, 112, 40, 1, 0,
		0, 0, 113, 114, 5, 93, 0, 0, 114, 42, 1, 0, 0, 0, 115, 116, 5, 76, 0, 0,
		116, 117, 5, 73, 0, 0, 117, 118, 5, 75, 0, 0, 118, 119, 5, 69, 0, 0, 119,
		44, 1, 0, 0, 0, 120, 121, 5, 105, 0, 0, 121, 122, 5, 110, 0, 0, 122, 46,
		1, 0, 0, 0, 123, 129, 5, 34, 0, 0, 124, 128, 8, 0, 0, 0, 125, 126, 5, 92,
		0, 0, 126, 128, 9, 0, 0, 0, 127, 124, 1, 0, 0, 0, 127, 125, 1, 0, 0, 0,
		128, 131, 1, 0, 0, 0, 129, 127, 1, 0, 0, 0, 129, 130, 1, 0, 0, 0, 130,
		132, 1, 0, 0, 0, 131, 129, 1, 0, 0, 0, 132, 133, 5, 34, 0, 0, 133, 48,
		1, 0, 0, 0, 134, 141, 5, 34, 0, 0, 135, 140, 8, 0, 0, 0, 136, 137, 5, 92,
		0, 0, 137, 140, 9, 0, 0, 0, 138, 140, 5, 37, 0, 0, 139, 135, 1, 0, 0, 0,
		139, 136, 1, 0, 0, 0, 139, 138, 1, 0, 0, 0, 140, 143, 1, 0, 0, 0, 141,
		139, 1, 0, 0, 0, 141, 142, 1, 0, 0, 0, 142, 144, 1, 0, 0, 0, 143, 141,
		1, 0, 0, 0, 144, 145, 5, 34, 0, 0, 145, 50, 1, 0, 0, 0, 146, 148, 7, 1,
		0, 0, 147, 146, 1, 0, 0, 0, 148, 149, 1, 0, 0, 0, 149, 147, 1, 0, 0, 0,
		149, 150, 1, 0, 0, 0, 150, 157, 1, 0, 0, 0, 151, 153, 5, 46, 0, 0, 152,
		154, 7, 1, 0, 0, 153, 152, 1, 0, 0, 0, 154, 155, 1, 0, 0, 0, 155, 153,
		1, 0, 0, 0, 155, 156, 1, 0, 0, 0, 156, 158, 1, 0, 0, 0, 157, 151, 1, 0,
		0, 0, 157, 158, 1, 0, 0, 0, 158, 52, 1, 0, 0, 0, 159, 160, 5, 112, 0, 0,
		160, 161, 5, 114, 0, 0, 161, 162, 5, 101, 0, 0, 162, 163, 5, 100, 0, 0,
		163, 164, 5, 105, 0, 0, 164, 165, 5, 99, 0, 0, 165, 166, 5, 97, 0, 0, 166,
		167, 5, 116, 0, 0, 167, 168, 5, 101, 0, 0, 168, 54, 1, 0, 0, 0, 169, 170,
		5, 70, 0, 0, 170, 171, 5, 82, 0, 0, 171, 172, 5, 79, 0, 0, 172, 173, 5,
		77, 0, 0, 173, 56, 1, 0, 0, 0, 174, 175, 5, 87, 0, 0, 175, 176, 5, 72,
		0, 0, 176, 177, 5, 69, 0, 0, 177, 178, 5, 82, 0, 0, 178, 179, 5, 69, 0,
		0, 179, 58, 1, 0, 0, 0, 180, 181, 5, 65, 0, 0, 181, 182, 5, 83, 0, 0, 182,
		60, 1, 0, 0, 0, 183, 184, 5, 83, 0, 0, 184, 185, 5, 69, 0, 0, 185, 186,
		5, 76, 0, 0, 186, 187, 5, 69, 0, 0, 187, 188, 5, 67, 0, 0, 188, 189, 5,
		84, 0, 0, 189, 62, 1, 0, 0, 0, 190, 194, 7, 2, 0, 0, 191, 193, 7, 3, 0,
		0, 192, 191, 1, 0, 0, 0, 193, 196, 1, 0, 0, 0, 194, 192, 1, 0, 0, 0, 194,
		195, 1, 0, 0, 0, 195, 64, 1, 0, 0, 0, 196, 194, 1, 0, 0, 0, 197, 199, 7,
		4, 0, 0, 198, 197, 1, 0, 0, 0, 199, 200, 1, 0, 0, 0, 200, 198, 1, 0, 0,
		0, 200, 201, 1, 0, 0, 0, 201, 202, 1, 0, 0, 0, 202, 203, 6, 32, 0, 0, 203,
		66, 1, 0, 0, 0, 10, 0, 127, 129, 139, 141, 149, 155, 157, 194, 200, 1,
		6, 0, 0,
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
	QueryLexerT__18                = 19
	QueryLexerT__19                = 20
	QueryLexerT__20                = 21
	QueryLexerT__21                = 22
	QueryLexerT__22                = 23
	QueryLexerSTRING               = 24
	QueryLexerSTRING_WITH_WILDCARD = 25
	QueryLexerNUMBER               = 26
	QueryLexerPREDICATE            = 27
	QueryLexerFROM                 = 28
	QueryLexerWHERE                = 29
	QueryLexerAS                   = 30
	QueryLexerSELECT               = 31
	QueryLexerIDENTIFIER           = 32
	QueryLexerWS                   = 33
)

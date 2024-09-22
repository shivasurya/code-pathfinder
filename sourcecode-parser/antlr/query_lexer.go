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
		"'WHERE'", "'AS'",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "",
		"", "", "", "", "", "", "", "STRING", "STRING_WITH_WILDCARD", "NUMBER",
		"PREDICATE", "FROM", "WHERE", "AS", "IDENTIFIER", "WS",
	}
	staticData.RuleNames = []string{
		"T__0", "T__1", "T__2", "T__3", "T__4", "T__5", "T__6", "T__7", "T__8",
		"T__9", "T__10", "T__11", "T__12", "T__13", "T__14", "T__15", "T__16",
		"T__17", "T__18", "T__19", "T__20", "T__21", "T__22", "STRING", "STRING_WITH_WILDCARD",
		"NUMBER", "PREDICATE", "FROM", "WHERE", "AS", "IDENTIFIER", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 32, 195, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2,
		31, 7, 31, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1, 2, 1, 3, 1, 3, 1, 4, 1, 4,
		1, 5, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 1, 8, 1, 8, 1, 8,
		1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 1, 12, 1, 12, 1, 12, 1,
		13, 1, 13, 1, 14, 1, 14, 1, 15, 1, 15, 1, 16, 1, 16, 1, 17, 1, 17, 1, 18,
		1, 18, 1, 19, 1, 19, 1, 20, 1, 20, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1,
		22, 1, 22, 1, 22, 1, 23, 1, 23, 1, 23, 1, 23, 5, 23, 126, 8, 23, 10, 23,
		12, 23, 129, 9, 23, 1, 23, 1, 23, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 5,
		24, 138, 8, 24, 10, 24, 12, 24, 141, 9, 24, 1, 24, 1, 24, 1, 25, 4, 25,
		146, 8, 25, 11, 25, 12, 25, 147, 1, 25, 1, 25, 4, 25, 152, 8, 25, 11, 25,
		12, 25, 153, 3, 25, 156, 8, 25, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26,
		1, 26, 1, 26, 1, 26, 1, 26, 1, 27, 1, 27, 1, 27, 1, 27, 1, 27, 1, 28, 1,
		28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 29, 1, 29, 1, 29, 1, 30, 1, 30, 5, 30,
		184, 8, 30, 10, 30, 12, 30, 187, 9, 30, 1, 31, 4, 31, 190, 8, 31, 11, 31,
		12, 31, 191, 1, 31, 1, 31, 0, 0, 32, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11,
		6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25, 13, 27, 14, 29, 15,
		31, 16, 33, 17, 35, 18, 37, 19, 39, 20, 41, 21, 43, 22, 45, 23, 47, 24,
		49, 25, 51, 26, 53, 27, 55, 28, 57, 29, 59, 30, 61, 31, 63, 32, 1, 0, 5,
		2, 0, 34, 34, 92, 92, 1, 0, 48, 57, 3, 0, 65, 90, 95, 95, 97, 122, 4, 0,
		48, 57, 65, 90, 95, 95, 97, 122, 3, 0, 9, 10, 13, 13, 32, 32, 204, 0, 1,
		1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9,
		1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0,
		17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0,
		0, 25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0,
		0, 0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0, 0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0,
		0, 0, 0, 41, 1, 0, 0, 0, 0, 43, 1, 0, 0, 0, 0, 45, 1, 0, 0, 0, 0, 47, 1,
		0, 0, 0, 0, 49, 1, 0, 0, 0, 0, 51, 1, 0, 0, 0, 0, 53, 1, 0, 0, 0, 0, 55,
		1, 0, 0, 0, 0, 57, 1, 0, 0, 0, 0, 59, 1, 0, 0, 0, 0, 61, 1, 0, 0, 0, 0,
		63, 1, 0, 0, 0, 1, 65, 1, 0, 0, 0, 3, 67, 1, 0, 0, 0, 5, 69, 1, 0, 0, 0,
		7, 71, 1, 0, 0, 0, 9, 73, 1, 0, 0, 0, 11, 75, 1, 0, 0, 0, 13, 78, 1, 0,
		0, 0, 15, 81, 1, 0, 0, 0, 17, 84, 1, 0, 0, 0, 19, 87, 1, 0, 0, 0, 21, 89,
		1, 0, 0, 0, 23, 91, 1, 0, 0, 0, 25, 94, 1, 0, 0, 0, 27, 97, 1, 0, 0, 0,
		29, 99, 1, 0, 0, 0, 31, 101, 1, 0, 0, 0, 33, 103, 1, 0, 0, 0, 35, 105,
		1, 0, 0, 0, 37, 107, 1, 0, 0, 0, 39, 109, 1, 0, 0, 0, 41, 111, 1, 0, 0,
		0, 43, 113, 1, 0, 0, 0, 45, 118, 1, 0, 0, 0, 47, 121, 1, 0, 0, 0, 49, 132,
		1, 0, 0, 0, 51, 145, 1, 0, 0, 0, 53, 157, 1, 0, 0, 0, 55, 167, 1, 0, 0,
		0, 57, 172, 1, 0, 0, 0, 59, 178, 1, 0, 0, 0, 61, 181, 1, 0, 0, 0, 63, 189,
		1, 0, 0, 0, 65, 66, 5, 40, 0, 0, 66, 2, 1, 0, 0, 0, 67, 68, 5, 41, 0, 0,
		68, 4, 1, 0, 0, 0, 69, 70, 5, 123, 0, 0, 70, 6, 1, 0, 0, 0, 71, 72, 5,
		125, 0, 0, 72, 8, 1, 0, 0, 0, 73, 74, 5, 44, 0, 0, 74, 10, 1, 0, 0, 0,
		75, 76, 5, 124, 0, 0, 76, 77, 5, 124, 0, 0, 77, 12, 1, 0, 0, 0, 78, 79,
		5, 38, 0, 0, 79, 80, 5, 38, 0, 0, 80, 14, 1, 0, 0, 0, 81, 82, 5, 61, 0,
		0, 82, 83, 5, 61, 0, 0, 83, 16, 1, 0, 0, 0, 84, 85, 5, 33, 0, 0, 85, 86,
		5, 61, 0, 0, 86, 18, 1, 0, 0, 0, 87, 88, 5, 60, 0, 0, 88, 20, 1, 0, 0,
		0, 89, 90, 5, 62, 0, 0, 90, 22, 1, 0, 0, 0, 91, 92, 5, 60, 0, 0, 92, 93,
		5, 61, 0, 0, 93, 24, 1, 0, 0, 0, 94, 95, 5, 62, 0, 0, 95, 96, 5, 61, 0,
		0, 96, 26, 1, 0, 0, 0, 97, 98, 5, 43, 0, 0, 98, 28, 1, 0, 0, 0, 99, 100,
		5, 45, 0, 0, 100, 30, 1, 0, 0, 0, 101, 102, 5, 42, 0, 0, 102, 32, 1, 0,
		0, 0, 103, 104, 5, 47, 0, 0, 104, 34, 1, 0, 0, 0, 105, 106, 5, 33, 0, 0,
		106, 36, 1, 0, 0, 0, 107, 108, 5, 46, 0, 0, 108, 38, 1, 0, 0, 0, 109, 110,
		5, 91, 0, 0, 110, 40, 1, 0, 0, 0, 111, 112, 5, 93, 0, 0, 112, 42, 1, 0,
		0, 0, 113, 114, 5, 76, 0, 0, 114, 115, 5, 73, 0, 0, 115, 116, 5, 75, 0,
		0, 116, 117, 5, 69, 0, 0, 117, 44, 1, 0, 0, 0, 118, 119, 5, 105, 0, 0,
		119, 120, 5, 110, 0, 0, 120, 46, 1, 0, 0, 0, 121, 127, 5, 34, 0, 0, 122,
		126, 8, 0, 0, 0, 123, 124, 5, 92, 0, 0, 124, 126, 9, 0, 0, 0, 125, 122,
		1, 0, 0, 0, 125, 123, 1, 0, 0, 0, 126, 129, 1, 0, 0, 0, 127, 125, 1, 0,
		0, 0, 127, 128, 1, 0, 0, 0, 128, 130, 1, 0, 0, 0, 129, 127, 1, 0, 0, 0,
		130, 131, 5, 34, 0, 0, 131, 48, 1, 0, 0, 0, 132, 139, 5, 34, 0, 0, 133,
		138, 8, 0, 0, 0, 134, 135, 5, 92, 0, 0, 135, 138, 9, 0, 0, 0, 136, 138,
		5, 37, 0, 0, 137, 133, 1, 0, 0, 0, 137, 134, 1, 0, 0, 0, 137, 136, 1, 0,
		0, 0, 138, 141, 1, 0, 0, 0, 139, 137, 1, 0, 0, 0, 139, 140, 1, 0, 0, 0,
		140, 142, 1, 0, 0, 0, 141, 139, 1, 0, 0, 0, 142, 143, 5, 34, 0, 0, 143,
		50, 1, 0, 0, 0, 144, 146, 7, 1, 0, 0, 145, 144, 1, 0, 0, 0, 146, 147, 1,
		0, 0, 0, 147, 145, 1, 0, 0, 0, 147, 148, 1, 0, 0, 0, 148, 155, 1, 0, 0,
		0, 149, 151, 5, 46, 0, 0, 150, 152, 7, 1, 0, 0, 151, 150, 1, 0, 0, 0, 152,
		153, 1, 0, 0, 0, 153, 151, 1, 0, 0, 0, 153, 154, 1, 0, 0, 0, 154, 156,
		1, 0, 0, 0, 155, 149, 1, 0, 0, 0, 155, 156, 1, 0, 0, 0, 156, 52, 1, 0,
		0, 0, 157, 158, 5, 112, 0, 0, 158, 159, 5, 114, 0, 0, 159, 160, 5, 101,
		0, 0, 160, 161, 5, 100, 0, 0, 161, 162, 5, 105, 0, 0, 162, 163, 5, 99,
		0, 0, 163, 164, 5, 97, 0, 0, 164, 165, 5, 116, 0, 0, 165, 166, 5, 101,
		0, 0, 166, 54, 1, 0, 0, 0, 167, 168, 5, 70, 0, 0, 168, 169, 5, 82, 0, 0,
		169, 170, 5, 79, 0, 0, 170, 171, 5, 77, 0, 0, 171, 56, 1, 0, 0, 0, 172,
		173, 5, 87, 0, 0, 173, 174, 5, 72, 0, 0, 174, 175, 5, 69, 0, 0, 175, 176,
		5, 82, 0, 0, 176, 177, 5, 69, 0, 0, 177, 58, 1, 0, 0, 0, 178, 179, 5, 65,
		0, 0, 179, 180, 5, 83, 0, 0, 180, 60, 1, 0, 0, 0, 181, 185, 7, 2, 0, 0,
		182, 184, 7, 3, 0, 0, 183, 182, 1, 0, 0, 0, 184, 187, 1, 0, 0, 0, 185,
		183, 1, 0, 0, 0, 185, 186, 1, 0, 0, 0, 186, 62, 1, 0, 0, 0, 187, 185, 1,
		0, 0, 0, 188, 190, 7, 4, 0, 0, 189, 188, 1, 0, 0, 0, 190, 191, 1, 0, 0,
		0, 191, 189, 1, 0, 0, 0, 191, 192, 1, 0, 0, 0, 192, 193, 1, 0, 0, 0, 193,
		194, 6, 31, 0, 0, 194, 64, 1, 0, 0, 0, 10, 0, 125, 127, 137, 139, 147,
		153, 155, 185, 191, 1, 6, 0, 0,
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
	QueryLexerIDENTIFIER           = 31
	QueryLexerWS                   = 32
)

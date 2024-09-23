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
		"'<'", "'>'", "'<='", "'>='", "' in '", "'+'", "'-'", "'*'", "'/'",
		"'!'", "'.'", "'['", "']'", "'LIKE'", "'in'", "", "", "", "'predicate'",
		"'FROM'", "'WHERE'", "'AS'", "'SELECT'",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "",
		"", "", "", "", "", "", "", "", "STRING", "STRING_WITH_WILDCARD", "NUMBER",
		"PREDICATE", "FROM", "WHERE", "AS", "SELECT", "IDENTIFIER", "WS",
	}
	staticData.RuleNames = []string{
		"T__0", "T__1", "T__2", "T__3", "T__4", "T__5", "T__6", "T__7", "T__8",
		"T__9", "T__10", "T__11", "T__12", "T__13", "T__14", "T__15", "T__16",
		"T__17", "T__18", "T__19", "T__20", "T__21", "T__22", "T__23", "STRING",
		"STRING_WITH_WILDCARD", "NUMBER", "PREDICATE", "FROM", "WHERE", "AS",
		"SELECT", "IDENTIFIER", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 34, 211, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2,
		31, 7, 31, 2, 32, 7, 32, 2, 33, 7, 33, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1,
		2, 1, 3, 1, 3, 1, 4, 1, 4, 1, 5, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6, 1, 7, 1,
		7, 1, 7, 1, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11,
		1, 12, 1, 12, 1, 12, 1, 13, 1, 13, 1, 13, 1, 13, 1, 13, 1, 14, 1, 14, 1,
		15, 1, 15, 1, 16, 1, 16, 1, 17, 1, 17, 1, 18, 1, 18, 1, 19, 1, 19, 1, 20,
		1, 20, 1, 21, 1, 21, 1, 22, 1, 22, 1, 22, 1, 22, 1, 22, 1, 23, 1, 23, 1,
		23, 1, 24, 1, 24, 1, 24, 1, 24, 5, 24, 135, 8, 24, 10, 24, 12, 24, 138,
		9, 24, 1, 24, 1, 24, 1, 25, 1, 25, 1, 25, 1, 25, 1, 25, 5, 25, 147, 8,
		25, 10, 25, 12, 25, 150, 9, 25, 1, 25, 1, 25, 1, 26, 4, 26, 155, 8, 26,
		11, 26, 12, 26, 156, 1, 26, 1, 26, 4, 26, 161, 8, 26, 11, 26, 12, 26, 162,
		3, 26, 165, 8, 26, 1, 27, 1, 27, 1, 27, 1, 27, 1, 27, 1, 27, 1, 27, 1,
		27, 1, 27, 1, 27, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 29, 1, 29, 1, 29,
		1, 29, 1, 29, 1, 29, 1, 30, 1, 30, 1, 30, 1, 31, 1, 31, 1, 31, 1, 31, 1,
		31, 1, 31, 1, 31, 1, 32, 1, 32, 5, 32, 200, 8, 32, 10, 32, 12, 32, 203,
		9, 32, 1, 33, 4, 33, 206, 8, 33, 11, 33, 12, 33, 207, 1, 33, 1, 33, 0,
		0, 34, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10,
		21, 11, 23, 12, 25, 13, 27, 14, 29, 15, 31, 16, 33, 17, 35, 18, 37, 19,
		39, 20, 41, 21, 43, 22, 45, 23, 47, 24, 49, 25, 51, 26, 53, 27, 55, 28,
		57, 29, 59, 30, 61, 31, 63, 32, 65, 33, 67, 34, 1, 0, 5, 2, 0, 34, 34,
		92, 92, 1, 0, 48, 57, 3, 0, 65, 90, 95, 95, 97, 122, 4, 0, 48, 57, 65,
		90, 95, 95, 97, 122, 3, 0, 9, 10, 13, 13, 32, 32, 220, 0, 1, 1, 0, 0, 0,
		0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0,
		0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0,
		0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0,
		0, 0, 0, 27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0, 33, 1,
		0, 0, 0, 0, 35, 1, 0, 0, 0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0, 0, 0, 0, 41,
		1, 0, 0, 0, 0, 43, 1, 0, 0, 0, 0, 45, 1, 0, 0, 0, 0, 47, 1, 0, 0, 0, 0,
		49, 1, 0, 0, 0, 0, 51, 1, 0, 0, 0, 0, 53, 1, 0, 0, 0, 0, 55, 1, 0, 0, 0,
		0, 57, 1, 0, 0, 0, 0, 59, 1, 0, 0, 0, 0, 61, 1, 0, 0, 0, 0, 63, 1, 0, 0,
		0, 0, 65, 1, 0, 0, 0, 0, 67, 1, 0, 0, 0, 1, 69, 1, 0, 0, 0, 3, 71, 1, 0,
		0, 0, 5, 73, 1, 0, 0, 0, 7, 75, 1, 0, 0, 0, 9, 77, 1, 0, 0, 0, 11, 79,
		1, 0, 0, 0, 13, 82, 1, 0, 0, 0, 15, 85, 1, 0, 0, 0, 17, 88, 1, 0, 0, 0,
		19, 91, 1, 0, 0, 0, 21, 93, 1, 0, 0, 0, 23, 95, 1, 0, 0, 0, 25, 98, 1,
		0, 0, 0, 27, 101, 1, 0, 0, 0, 29, 106, 1, 0, 0, 0, 31, 108, 1, 0, 0, 0,
		33, 110, 1, 0, 0, 0, 35, 112, 1, 0, 0, 0, 37, 114, 1, 0, 0, 0, 39, 116,
		1, 0, 0, 0, 41, 118, 1, 0, 0, 0, 43, 120, 1, 0, 0, 0, 45, 122, 1, 0, 0,
		0, 47, 127, 1, 0, 0, 0, 49, 130, 1, 0, 0, 0, 51, 141, 1, 0, 0, 0, 53, 154,
		1, 0, 0, 0, 55, 166, 1, 0, 0, 0, 57, 176, 1, 0, 0, 0, 59, 181, 1, 0, 0,
		0, 61, 187, 1, 0, 0, 0, 63, 190, 1, 0, 0, 0, 65, 197, 1, 0, 0, 0, 67, 205,
		1, 0, 0, 0, 69, 70, 5, 40, 0, 0, 70, 2, 1, 0, 0, 0, 71, 72, 5, 41, 0, 0,
		72, 4, 1, 0, 0, 0, 73, 74, 5, 123, 0, 0, 74, 6, 1, 0, 0, 0, 75, 76, 5,
		125, 0, 0, 76, 8, 1, 0, 0, 0, 77, 78, 5, 44, 0, 0, 78, 10, 1, 0, 0, 0,
		79, 80, 5, 124, 0, 0, 80, 81, 5, 124, 0, 0, 81, 12, 1, 0, 0, 0, 82, 83,
		5, 38, 0, 0, 83, 84, 5, 38, 0, 0, 84, 14, 1, 0, 0, 0, 85, 86, 5, 61, 0,
		0, 86, 87, 5, 61, 0, 0, 87, 16, 1, 0, 0, 0, 88, 89, 5, 33, 0, 0, 89, 90,
		5, 61, 0, 0, 90, 18, 1, 0, 0, 0, 91, 92, 5, 60, 0, 0, 92, 20, 1, 0, 0,
		0, 93, 94, 5, 62, 0, 0, 94, 22, 1, 0, 0, 0, 95, 96, 5, 60, 0, 0, 96, 97,
		5, 61, 0, 0, 97, 24, 1, 0, 0, 0, 98, 99, 5, 62, 0, 0, 99, 100, 5, 61, 0,
		0, 100, 26, 1, 0, 0, 0, 101, 102, 5, 32, 0, 0, 102, 103, 5, 105, 0, 0,
		103, 104, 5, 110, 0, 0, 104, 105, 5, 32, 0, 0, 105, 28, 1, 0, 0, 0, 106,
		107, 5, 43, 0, 0, 107, 30, 1, 0, 0, 0, 108, 109, 5, 45, 0, 0, 109, 32,
		1, 0, 0, 0, 110, 111, 5, 42, 0, 0, 111, 34, 1, 0, 0, 0, 112, 113, 5, 47,
		0, 0, 113, 36, 1, 0, 0, 0, 114, 115, 5, 33, 0, 0, 115, 38, 1, 0, 0, 0,
		116, 117, 5, 46, 0, 0, 117, 40, 1, 0, 0, 0, 118, 119, 5, 91, 0, 0, 119,
		42, 1, 0, 0, 0, 120, 121, 5, 93, 0, 0, 121, 44, 1, 0, 0, 0, 122, 123, 5,
		76, 0, 0, 123, 124, 5, 73, 0, 0, 124, 125, 5, 75, 0, 0, 125, 126, 5, 69,
		0, 0, 126, 46, 1, 0, 0, 0, 127, 128, 5, 105, 0, 0, 128, 129, 5, 110, 0,
		0, 129, 48, 1, 0, 0, 0, 130, 136, 5, 34, 0, 0, 131, 135, 8, 0, 0, 0, 132,
		133, 5, 92, 0, 0, 133, 135, 9, 0, 0, 0, 134, 131, 1, 0, 0, 0, 134, 132,
		1, 0, 0, 0, 135, 138, 1, 0, 0, 0, 136, 134, 1, 0, 0, 0, 136, 137, 1, 0,
		0, 0, 137, 139, 1, 0, 0, 0, 138, 136, 1, 0, 0, 0, 139, 140, 5, 34, 0, 0,
		140, 50, 1, 0, 0, 0, 141, 148, 5, 34, 0, 0, 142, 147, 8, 0, 0, 0, 143,
		144, 5, 92, 0, 0, 144, 147, 9, 0, 0, 0, 145, 147, 5, 37, 0, 0, 146, 142,
		1, 0, 0, 0, 146, 143, 1, 0, 0, 0, 146, 145, 1, 0, 0, 0, 147, 150, 1, 0,
		0, 0, 148, 146, 1, 0, 0, 0, 148, 149, 1, 0, 0, 0, 149, 151, 1, 0, 0, 0,
		150, 148, 1, 0, 0, 0, 151, 152, 5, 34, 0, 0, 152, 52, 1, 0, 0, 0, 153,
		155, 7, 1, 0, 0, 154, 153, 1, 0, 0, 0, 155, 156, 1, 0, 0, 0, 156, 154,
		1, 0, 0, 0, 156, 157, 1, 0, 0, 0, 157, 164, 1, 0, 0, 0, 158, 160, 5, 46,
		0, 0, 159, 161, 7, 1, 0, 0, 160, 159, 1, 0, 0, 0, 161, 162, 1, 0, 0, 0,
		162, 160, 1, 0, 0, 0, 162, 163, 1, 0, 0, 0, 163, 165, 1, 0, 0, 0, 164,
		158, 1, 0, 0, 0, 164, 165, 1, 0, 0, 0, 165, 54, 1, 0, 0, 0, 166, 167, 5,
		112, 0, 0, 167, 168, 5, 114, 0, 0, 168, 169, 5, 101, 0, 0, 169, 170, 5,
		100, 0, 0, 170, 171, 5, 105, 0, 0, 171, 172, 5, 99, 0, 0, 172, 173, 5,
		97, 0, 0, 173, 174, 5, 116, 0, 0, 174, 175, 5, 101, 0, 0, 175, 56, 1, 0,
		0, 0, 176, 177, 5, 70, 0, 0, 177, 178, 5, 82, 0, 0, 178, 179, 5, 79, 0,
		0, 179, 180, 5, 77, 0, 0, 180, 58, 1, 0, 0, 0, 181, 182, 5, 87, 0, 0, 182,
		183, 5, 72, 0, 0, 183, 184, 5, 69, 0, 0, 184, 185, 5, 82, 0, 0, 185, 186,
		5, 69, 0, 0, 186, 60, 1, 0, 0, 0, 187, 188, 5, 65, 0, 0, 188, 189, 5, 83,
		0, 0, 189, 62, 1, 0, 0, 0, 190, 191, 5, 83, 0, 0, 191, 192, 5, 69, 0, 0,
		192, 193, 5, 76, 0, 0, 193, 194, 5, 69, 0, 0, 194, 195, 5, 67, 0, 0, 195,
		196, 5, 84, 0, 0, 196, 64, 1, 0, 0, 0, 197, 201, 7, 2, 0, 0, 198, 200,
		7, 3, 0, 0, 199, 198, 1, 0, 0, 0, 200, 203, 1, 0, 0, 0, 201, 199, 1, 0,
		0, 0, 201, 202, 1, 0, 0, 0, 202, 66, 1, 0, 0, 0, 203, 201, 1, 0, 0, 0,
		204, 206, 7, 4, 0, 0, 205, 204, 1, 0, 0, 0, 206, 207, 1, 0, 0, 0, 207,
		205, 1, 0, 0, 0, 207, 208, 1, 0, 0, 0, 208, 209, 1, 0, 0, 0, 209, 210,
		6, 33, 0, 0, 210, 68, 1, 0, 0, 0, 10, 0, 134, 136, 146, 148, 156, 162,
		164, 201, 207, 1, 6, 0, 0,
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
	QueryLexerT__23                = 24
	QueryLexerSTRING               = 25
	QueryLexerSTRING_WITH_WILDCARD = 26
	QueryLexerNUMBER               = 27
	QueryLexerPREDICATE            = 28
	QueryLexerFROM                 = 29
	QueryLexerWHERE                = 30
	QueryLexerAS                   = 31
	QueryLexerSELECT               = 32
	QueryLexerIDENTIFIER           = 33
	QueryLexerWS                   = 34
)

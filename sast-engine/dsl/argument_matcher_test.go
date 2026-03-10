package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// --- Comparator tests ---

func TestMatchesArgumentValue_Lt(t *testing.T) {
	constraint := ArgumentConstraint{Value: float64(2048), Comparator: "lt"}
	if !MatchesArgumentValue("1024", constraint) {
		t.Error("Expected 1024 < 2048 to match")
	}
	if MatchesArgumentValue("2048", constraint) {
		t.Error("Expected 2048 < 2048 to NOT match")
	}
	if MatchesArgumentValue("4096", constraint) {
		t.Error("Expected 4096 < 2048 to NOT match")
	}
}

func TestMatchesArgumentValue_Gt(t *testing.T) {
	constraint := ArgumentConstraint{Value: float64(100), Comparator: "gt"}
	if !MatchesArgumentValue("200", constraint) {
		t.Error("Expected 200 > 100 to match")
	}
	if MatchesArgumentValue("50", constraint) {
		t.Error("Expected 50 > 100 to NOT match")
	}
}

func TestMatchesArgumentValue_Lte(t *testing.T) {
	constraint := ArgumentConstraint{Value: float64(1024), Comparator: "lte"}
	if !MatchesArgumentValue("1024", constraint) {
		t.Error("Expected 1024 <= 1024 to match")
	}
	if !MatchesArgumentValue("512", constraint) {
		t.Error("Expected 512 <= 1024 to match")
	}
	if MatchesArgumentValue("2048", constraint) {
		t.Error("Expected 2048 <= 1024 to NOT match")
	}
}

func TestMatchesArgumentValue_Gte(t *testing.T) {
	constraint := ArgumentConstraint{Value: float64(256), Comparator: "gte"}
	if !MatchesArgumentValue("256", constraint) {
		t.Error("Expected 256 >= 256 to match")
	}
	if !MatchesArgumentValue("512", constraint) {
		t.Error("Expected 512 >= 256 to match")
	}
	if MatchesArgumentValue("128", constraint) {
		t.Error("Expected 128 >= 256 to NOT match")
	}
}

func TestMatchesArgumentValue_Regex(t *testing.T) {
	constraint := ArgumentConstraint{Value: "http://.*", Comparator: "regex"}
	if !MatchesArgumentValue("http://example.com", constraint) {
		t.Error("Expected http://example.com to match regex")
	}
	if MatchesArgumentValue("https://example.com", constraint) {
		t.Error("Expected https://example.com to NOT match http:// regex")
	}
}

func TestMatchesArgumentValue_Missing(t *testing.T) {
	constraint := ArgumentConstraint{Comparator: "missing"}
	// "missing" always returns false at value level — handled at keyword level
	if MatchesArgumentValue("anything", constraint) {
		t.Error("Expected missing comparator to return false at value level")
	}
}

func TestMatchesArgumentValue_ExactMatch(t *testing.T) {
	constraint := ArgumentConstraint{Value: "hello"}
	if !MatchesArgumentValue("hello", constraint) {
		t.Error("Expected exact match")
	}
	if MatchesArgumentValue("world", constraint) {
		t.Error("Expected no match for different value")
	}
}

func TestMatchesArgumentValue_BooleanMatch(t *testing.T) {
	constraint := ArgumentConstraint{Value: true}
	if !MatchesArgumentValue("True", constraint) {
		t.Error("Expected True to match true")
	}
	if MatchesArgumentValue("False", constraint) {
		t.Error("Expected False to NOT match true")
	}
}

// --- Missing keyword tests ---

func TestMatchesKeywordArguments_Missing(t *testing.T) {
	args := []core.Argument{
		{Value: "url", Position: 0},
		// No "timeout" keyword
	}

	keywordArgs := map[string]ArgumentConstraint{
		"timeout": {Comparator: "missing"},
	}

	if !MatchesKeywordArguments(args, keywordArgs) {
		t.Error("Expected missing() to pass when keyword is absent")
	}
}

func TestMatchesKeywordArguments_MissingButPresent(t *testing.T) {
	args := []core.Argument{
		{Value: "url", Position: 0},
		{Value: "timeout=30", Position: 1},
	}

	keywordArgs := map[string]ArgumentConstraint{
		"timeout": {Comparator: "missing"},
	}

	if MatchesKeywordArguments(args, keywordArgs) {
		t.Error("Expected missing() to fail when keyword IS present")
	}
}

// --- MatchesArguments integration ---

func TestMatchesArguments_PositionalAndKeyword(t *testing.T) {
	cs := &core.CallSite{
		Arguments: []core.Argument{
			{Value: `"SELECT * FROM users"`, Position: 0},
			{Value: "shell=True", Position: 1},
		},
	}

	positional := map[string]ArgumentConstraint{
		"0": {Value: "SELECT * FROM users"},
	}
	keyword := map[string]ArgumentConstraint{
		"shell": {Value: true},
	}

	if !MatchesArguments(cs, positional, keyword) {
		t.Error("Expected both positional and keyword to match")
	}
}

func TestMatchesArguments_Empty(t *testing.T) {
	cs := &core.CallSite{}
	if !MatchesArguments(cs, nil, nil) {
		t.Error("Expected empty constraints to match")
	}
}

// --- deriveReceiverFromFQN tests ---

func TestDeriveReceiverFromFQN(t *testing.T) {
	tests := []struct {
		targetFQN string
		target    string
		want      string
	}{
		{"os.system", "os.system", "os"},
		{"pickle.loads", "pickle.loads", "pickle"},
		{"django.shortcuts.redirect", "django.shortcuts.redirect", "django.shortcuts"},
		{"subprocess.call", "subprocess.call", "subprocess"},
		{"os.system", "system", "os"},
		{"", "system", ""},
		{"os.system", "os.exec", ""}, // FQN doesn't end with method name
	}

	for _, tt := range tests {
		t.Run(tt.targetFQN, func(t *testing.T) {
			got := deriveReceiverFromFQN(tt.targetFQN, tt.target)
			if got != tt.want {
				t.Errorf("deriveReceiverFromFQN(%q, %q) = %q, want %q",
					tt.targetFQN, tt.target, got, tt.want)
			}
		})
	}
}

// --- TypeConstrainedCallExecutor multi-receiver/method tests ---

func TestTypeConstrainedCallExecutor_MultiReceiverTypes(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.db": {
				{
					Target:                   "cursor.execute",
					Location:                 core.Location{File: "db.py", Line: 10},
					ResolvedViaTypeInference: true,
					InferredType:             "psycopg2.extensions.cursor",
					TypeConfidence:           0.9,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"sqlite3.Cursor", "psycopg2.extensions.cursor"},
			MethodNames:   []string{"execute"},
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match with multi-receiver, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_MultiMethodNames(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.db": {
				{
					Target:                   "cursor.executemany",
					Location:                 core.Location{File: "db.py", Line: 10},
					ResolvedViaTypeInference: true,
					InferredType:             "sqlite3.Cursor",
					TypeConfidence:           0.9,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"sqlite3.Cursor"},
			MethodNames:   []string{"execute", "executemany"},
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match with multi-method, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_ReceiverPatterns(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.db": {
				{
					Target:                   "cursor.execute",
					Location:                 core.Location{File: "db.py", Line: 10},
					ResolvedViaTypeInference: true,
					InferredType:             "mysql.connector.Cursor",
					TypeConfidence:           0.9,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes:    []string{"sqlite3.Cursor"},
			ReceiverPatterns: []string{"*.Cursor"},
			MethodNames:      []string{"execute"},
			MinConfidence:    0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match via receiver pattern, got %d", len(results))
	}
}

// --- FQN Bridge tests ---

func TestTypeConstrainedCallExecutor_FQNBridge(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.utils": {
				{
					Target:                   "os.system",
					TargetFQN:                "os.system",
					Location:                 core.Location{File: "utils.py", Line: 5},
					ResolvedViaTypeInference: false, // Module, not class
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"os"},
			MethodNames:   []string{"system"},
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match via FQN bridge, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_FQNBridge_Subprocess(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.run": {
				{
					Target:                   "subprocess.call",
					TargetFQN:                "subprocess.call",
					Location:                 core.Location{File: "run.py", Line: 3},
					ResolvedViaTypeInference: false,
					Arguments: []core.Argument{
						{Value: `"ls"`, Position: 0},
						{Value: "shell=True", Position: 1},
					},
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"subprocess"},
			MethodNames:   []string{"call", "run", "Popen"},
			MinConfidence: 0.5,
			KeywordArgs: map[string]ArgumentConstraint{
				"shell": {Value: true},
			},
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match via FQN bridge + arg match, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_FQNBridge_NoMatch_WrongArg(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.run": {
				{
					Target:                   "subprocess.call",
					TargetFQN:                "subprocess.call",
					Location:                 core.Location{File: "run.py", Line: 3},
					ResolvedViaTypeInference: false,
					Arguments: []core.Argument{
						{Value: `"ls"`, Position: 0},
						// No shell=True
					},
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"subprocess"},
			MethodNames:   []string{"call"},
			MinConfidence: 0.5,
			KeywordArgs: map[string]ArgumentConstraint{
				"shell": {Value: true},
			},
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches (no shell=True), got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_FQNBridge_DjangoShortcuts(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views": {
				{
					Target:                   "django.shortcuts.redirect",
					TargetFQN:                "django.shortcuts.redirect",
					Location:                 core.Location{File: "views.py", Line: 8},
					ResolvedViaTypeInference: false,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"django.shortcuts"},
			MethodNames:   []string{"redirect"},
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match for django.shortcuts.redirect, got %d", len(results))
	}
}

// --- Backward compatibility test ---

func TestTypeConstrainedCallExecutor_BackwardCompat_SingleFields(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.test": {
				{
					Target:                   "obj.method",
					Location:                 core.Location{File: "test.py", Line: 5},
					ResolvedViaTypeInference: true,
					InferredType:             "django.views.View",
					TypeConfidence:           0.95,
				},
			},
		},
	}

	// Old-style IR with single ReceiverType and MethodName
	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "django.views.View",
			MethodName:    "method",
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected backward compat to work, got %d results", len(results))
	}
}

// --- Comparator with TypeConstrainedCall ---

func TestTypeConstrainedCallExecutor_ArgComparator_Lt(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.crypto": {
				{
					Target:                   "rsa.generate_private_key",
					TargetFQN:                "cryptography.hazmat.primitives.asymmetric.rsa.generate_private_key",
					Location:                 core.Location{File: "crypto.py", Line: 5},
					ResolvedViaTypeInference: false,
					Arguments: []core.Argument{
						{Value: "key_size=1024", Position: 0},
					},
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"cryptography.hazmat.primitives.asymmetric.rsa"},
			MethodNames:   []string{"generate_private_key"},
			MinConfidence: 0.5,
			KeywordArgs: map[string]ArgumentConstraint{
				"key_size": {Value: float64(2048), Comparator: "lt"},
			},
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match for key_size < 2048, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_ArgComparator_Lt_NoMatch(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.crypto": {
				{
					Target:                   "rsa.generate_private_key",
					TargetFQN:                "cryptography.hazmat.primitives.asymmetric.rsa.generate_private_key",
					Location:                 core.Location{File: "crypto.py", Line: 5},
					ResolvedViaTypeInference: false,
					Arguments: []core.Argument{
						{Value: "key_size=4096", Position: 0},
					},
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"cryptography.hazmat.primitives.asymmetric.rsa"},
			MethodNames:   []string{"generate_private_key"},
			MinConfidence: 0.5,
			KeywordArgs: map[string]ArgumentConstraint{
				"key_size": {Value: float64(2048), Comparator: "lt"},
			},
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches for key_size=4096 < 2048, got %d", len(results))
	}
}

// --- parseTupleIndex tests ---

func TestParseTupleIndex_Shared(t *testing.T) {
	tests := []struct {
		name         string
		posStr       string
		wantPos      int
		wantTupleIdx int
		wantIsTuple  bool
		wantValid    bool
	}{
		{"simple position", "0", 0, 0, false, true},
		{"simple position 3", "3", 3, 0, false, true},
		{"tuple index", "0[1]", 0, 1, true, true},
		{"tuple index pos2", "2[0]", 2, 0, true, true},
		{"invalid non-numeric", "abc", 0, 0, false, false},
		{"malformed open bracket only", "0[", 0, 0, false, true},
		{"parse error in tuple index", "0[abc]", 0, 0, false, false},
		{"malformed bracket with bad pos", "abc[", 0, 0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, tupleIdx, isTuple, valid := parseTupleIndex(tt.posStr)
			if pos != tt.wantPos || tupleIdx != tt.wantTupleIdx || isTuple != tt.wantIsTuple || valid != tt.wantValid {
				t.Errorf("parseTupleIndex(%q) = (%d, %d, %v, %v), want (%d, %d, %v, %v)",
					tt.posStr, pos, tupleIdx, isTuple, valid,
					tt.wantPos, tt.wantTupleIdx, tt.wantIsTuple, tt.wantValid)
			}
		})
	}
}

// --- extractTupleElement tests ---

func TestExtractTupleElement_Shared(t *testing.T) {
	tests := []struct {
		name     string
		tupleStr string
		index    int
		wantVal  string
		wantOk   bool
	}{
		{"tuple index 0", `("0.0.0.0", 8080)`, 0, "0.0.0.0", true},
		{"tuple index 1", `("0.0.0.0", 8080)`, 1, "8080", true},
		{"index out of bounds", `("a", "b")`, 5, "", false},
		{"empty tuple", `()`, 0, "", false},
		{"not a tuple index 0", `plain_value`, 0, "plain_value", true},
		{"not a tuple index 1", `plain_value`, 1, "", false},
		{"list syntax", `["a", "b"]`, 0, "a", true},
		{"list syntax index 1", `["a", "b"]`, 1, "b", true},
		{"single element tuple", `("only")`, 0, "only", true},
		{"whitespace around tuple", `  ("x", "y")  `, 0, "x", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := extractTupleElement(tt.tupleStr, tt.index)
			if val != tt.wantVal || ok != tt.wantOk {
				t.Errorf("extractTupleElement(%q, %d) = (%q, %v), want (%q, %v)",
					tt.tupleStr, tt.index, val, ok, tt.wantVal, tt.wantOk)
			}
		})
	}
}

// --- MatchesPositionalArguments with tuple indexing ---

func TestMatchesPositionalArguments_TupleIndex(t *testing.T) {
	args := []core.Argument{
		{Value: `("0.0.0.0", 8080)`, Position: 0},
		{Value: `"hello"`, Position: 1},
	}

	// Match tuple element at position 0, index 1 (8080)
	positional := map[string]ArgumentConstraint{
		"0[1]": {Value: float64(8080)},
	}
	if !MatchesPositionalArguments(args, positional) {
		t.Error("Expected tuple index 0[1] to match 8080")
	}

	// Match tuple element at position 0, index 0 ("0.0.0.0")
	positional2 := map[string]ArgumentConstraint{
		"0[0]": {Value: "0.0.0.0"},
	}
	if !MatchesPositionalArguments(args, positional2) {
		t.Error("Expected tuple index 0[0] to match 0.0.0.0")
	}

	// Tuple index out of bounds
	positional3 := map[string]ArgumentConstraint{
		"0[5]": {Value: "anything"},
	}
	if MatchesPositionalArguments(args, positional3) {
		t.Error("Expected tuple index out of bounds to fail")
	}

	// Invalid position string
	positional4 := map[string]ArgumentConstraint{
		"abc": {Value: "anything"},
	}
	if MatchesPositionalArguments(args, positional4) {
		t.Error("Expected invalid position string to fail")
	}

	// Position out of bounds
	positional5 := map[string]ArgumentConstraint{
		"10": {Value: "anything"},
	}
	if MatchesPositionalArguments(args, positional5) {
		t.Error("Expected position out of bounds to fail")
	}
}

// --- CleanValue edge cases ---

func TestCleanValue_Shared(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"double quoted", `"hello"`, "hello"},
		{"single quoted", `'hello'`, "hello"},
		{"no quotes", "hello", "hello"},
		{"short string single char", "x", "x"},
		{"empty string", "", ""},
		{"whitespace", "  hello  ", "hello"},
		{"mismatched quotes", `"hello'`, `"hello'`},
		{"just quotes", `""`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanValue(tt.input)
			if got != tt.want {
				t.Errorf("CleanValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- NormalizeValue edge cases ---

func TestNormalizeValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"True to lowercase", "True", "true"},
		{"FALSE to lowercase", "FALSE", "false"},
		{"None to none", "None", "none"},
		{"null to none", "null", "none"},
		{"nil to none", "nil", "none"},
		{"NULL to none", "NULL", "none"},
		{"regular string unchanged", "MyValue", "MyValue"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeValue(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- compareNumeric edge cases ---

func TestCompareNumeric(t *testing.T) {
	lt := func(a, b float64) bool { return a < b }

	tests := []struct {
		name     string
		actual   string
		expected any
		want     bool
	}{
		{"float actual", "3.14", float64(4.0), true},
		{"non-numeric actual", "abc", float64(10), false},
		{"expected as int", "5", int(10), true},
		{"expected as int64", "5", int64(10), true},
		{"expected as unsupported string type", "5", "10", false},
		{"integer actual", "42", float64(100), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareNumeric(tt.actual, tt.expected, lt)
			if got != tt.want {
				t.Errorf("compareNumeric(%q, %v, lt) = %v, want %v",
					tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}

// --- matchesBooleanShared tests ---

func TestMatchesBooleanShared(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected bool
		want     bool
	}{
		{"1 for true", "1", true, true},
		{"0 for false", "0", false, true},
		{"false matches false", "false", false, true},
		{"true matches true", "true", true, true},
		{"True matches true", "True", true, true},
		{"FALSE matches false", "FALSE", false, true},
		{"0 does not match true", "0", true, false},
		{"1 does not match false", "1", false, false},
		{"random string not boolean", "yes", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesBooleanShared(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("matchesBooleanShared(%q, %v) = %v, want %v",
					tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}

// --- matchesNumberShared tests ---

func TestMatchesNumberShared(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected float64
		want     bool
	}{
		{"float comparison", "3.14", 3.14, true},
		{"int comparison", "42", 42.0, true},
		{"non-numeric", "abc", 42.0, false},
		{"octal", "0o777", 511.0, true},
		{"hex", "0xFF", 255.0, true},
		{"mismatch", "10", 20.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesNumberShared(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("matchesNumberShared(%q, %v) = %v, want %v",
					tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}

// --- matchesSingleValueShared tests ---

func TestMatchesSingleValueShared(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected any
		wildcard bool
		want     bool
	}{
		{"nil matches None", "None", nil, false, true},
		{"nil matches nil", "nil", nil, false, true},
		{"nil matches null", "null", nil, false, true},
		{"nil does not match other", "something", nil, false, false},
		{"unsupported type returns false", "hello", struct{}{}, false, false},
		{"wildcard string match", "hello_world", "hello*", true, true},
		{"wildcard no match", "hello_world", "foo*", true, false},
		{"exact string match", "hello", "hello", false, true},
		{"exact string no match", "hello", "world", false, false},
		{"bool true", "True", true, false, true},
		{"float64 match", "42", float64(42), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSingleValueShared(tt.actual, tt.expected, tt.wildcard)
			if got != tt.want {
				t.Errorf("matchesSingleValueShared(%q, %v, %v) = %v, want %v",
					tt.actual, tt.expected, tt.wildcard, got, tt.want)
			}
		})
	}
}

// --- wildcardMatchShared tests ---

func TestWildcardMatchShared(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		pattern string
		want    bool
	}{
		{"question mark matching", "abc", "a?c", true},
		{"question mark no match", "abcd", "a?c", false},
		{"star in middle", "abcdef", "ab*ef", true},
		{"star in middle no match", "abcdef", "ab*xy", false},
		{"backtracking needed", "abcabcabd", "abc*abd", true},
		{"backtracking fails", "abcabcabc", "abc*abd", false},
		{"exact match no wildcards", "hello", "hello", true},
		{"no match no wildcards", "hello", "world", false},
		{"star matches empty", "abc", "abc*", true},
		{"star matches everything", "anything", "*", true},
		{"multiple stars", "abc", "a**c", true},
		{"question at end", "ab", "a?", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wildcardMatchShared(tt.str, tt.pattern)
			if got != tt.want {
				t.Errorf("wildcardMatchShared(%q, %q) = %v, want %v",
					tt.str, tt.pattern, got, tt.want)
			}
		})
	}
}

// --- MatchesArgumentValue with regex where Value is non-string ---

func TestMatchesArgumentValue_RegexNonStringValue(t *testing.T) {
	constraint := ArgumentConstraint{Value: 123, Comparator: "regex"}
	if MatchesArgumentValue("anything", constraint) {
		t.Error("Expected regex with non-string Value to return false")
	}
}

// --- MatchesArgumentValue with nil expected ---

func TestMatchesArgumentValue_NilExpected(t *testing.T) {
	constraint := ArgumentConstraint{Value: nil}
	if !MatchesArgumentValue("None", constraint) {
		t.Error("Expected None to match nil constraint")
	}
	if !MatchesArgumentValue("null", constraint) {
		t.Error("Expected null to match nil constraint")
	}
	if !MatchesArgumentValue("nil", constraint) {
		t.Error("Expected nil to match nil constraint")
	}
	if MatchesArgumentValue("something", constraint) {
		t.Error("Expected something to NOT match nil constraint")
	}
}

// --- MatchesArgumentValue with unsupported type ---

func TestMatchesArgumentValue_UnsupportedType(t *testing.T) {
	constraint := ArgumentConstraint{Value: struct{ X int }{42}}
	if MatchesArgumentValue("anything", constraint) {
		t.Error("Expected unsupported type to return false")
	}
}

// --- MatchesArgumentValue with wildcard ---

func TestMatchesArgumentValue_Wildcard(t *testing.T) {
	constraint := ArgumentConstraint{Value: "0.0.*", Wildcard: true}
	if !MatchesArgumentValue("0.0.0.0", constraint) {
		t.Error("Expected 0.0.0.0 to match wildcard 0.0.*")
	}
	if MatchesArgumentValue("1.2.3.4", constraint) {
		t.Error("Expected 1.2.3.4 to NOT match wildcard 0.0.*")
	}
}

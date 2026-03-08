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

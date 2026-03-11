package dsl

import (
	"slices"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// --- Mock InheritanceChecker ---

type mockInheritanceChecker struct {
	modules map[string]bool
	classes map[string]mockClassInfo // key: "module.className"
}

type mockClassInfo struct {
	mro []string
}

func (m *mockInheritanceChecker) HasModule(moduleName string) bool {
	return m.modules[moduleName]
}

func (m *mockInheritanceChecker) IsSubclassSimple(moduleName, className, parentFQN string) bool {
	key := moduleName + "." + className
	info, ok := m.classes[key]
	if !ok {
		return false
	}
	return slices.Contains(info.mro, parentFQN)
}

func (m *mockInheritanceChecker) GetClassMRO(moduleName, className string) []string {
	key := moduleName + "." + className
	if info, ok := m.classes[key]; ok {
		return info.mro
	}
	return nil
}

func newMockChecker() *mockInheritanceChecker {
	return &mockInheritanceChecker{
		modules: map[string]bool{
			"django": true,
		},
		classes: map[string]mockClassInfo{
			"django.views.View": {
				mro: []string{"django.views.View", "builtins.object"},
			},
			"django.views.generic.ListView": {
				mro: []string{"django.views.generic.ListView", "django.views.View", "builtins.object"},
			},
			"django.http.HttpRequest": {
				mro: []string{"django.http.HttpRequest", "builtins.object"},
			},
		},
	}
}

// --- matchesReceiverType tests ---

func TestMatchesReceiverType_ExactMatch(t *testing.T) {
	if !matchesReceiverType("django.views.View", "django.views.View", nil) {
		t.Error("Expected exact match to succeed")
	}
}

func TestMatchesReceiverType_NoMatch(t *testing.T) {
	if matchesReceiverType("django.views.View", "flask.views.View", nil) {
		t.Error("Expected different FQNs to not match")
	}
}

func TestMatchesReceiverType_ShortName(t *testing.T) {
	if !matchesReceiverType("django.views.View", "View", nil) {
		t.Error("Expected short name 'View' to match 'django.views.View'")
	}
}

func TestMatchesReceiverType_ShortNameNoMatch(t *testing.T) {
	if matchesReceiverType("django.views.View", "Model", nil) {
		t.Error("Expected short name 'Model' to not match 'django.views.View'")
	}
}

func TestMatchesReceiverType_WildcardPrefix(t *testing.T) {
	if !matchesReceiverType("sqlite3.Cursor", "*Cursor", nil) {
		t.Error("Expected *Cursor to match sqlite3.Cursor")
	}
}

func TestMatchesReceiverType_WildcardSuffix(t *testing.T) {
	if !matchesReceiverType("sqlite3.Cursor", "sqlite3.*", nil) {
		t.Error("Expected sqlite3.* to match sqlite3.Cursor")
	}
}

func TestMatchesReceiverType_WildcardAll(t *testing.T) {
	if !matchesReceiverType("anything", "*", nil) {
		t.Error("Expected * to match anything")
	}
}

func TestMatchesReceiverType_WildcardContains(t *testing.T) {
	if !matchesReceiverType("django.views.generic.ListView", "*View*", nil) {
		t.Error("Expected *View* to match ListView")
	}
}

func TestMatchesReceiverType_EmptyActual(t *testing.T) {
	if matchesReceiverType("", "View", nil) {
		t.Error("Expected empty actual to not match")
	}
}

func TestMatchesReceiverType_EmptyPattern(t *testing.T) {
	if matchesReceiverType("django.views.View", "", nil) {
		t.Error("Expected empty pattern to not match")
	}
}

// --- Inheritance-aware matching ---

func TestMatchesReceiverType_InheritanceDirect(t *testing.T) {
	checker := newMockChecker()
	// ListView's MRO includes django.views.View
	if !matchesReceiverType("django.views.generic.ListView", "django.views.View", checker) {
		t.Error("Expected ListView to match View via inheritance")
	}
}

func TestMatchesReceiverType_InheritanceShortName(t *testing.T) {
	checker := newMockChecker()
	// ListView's MRO includes django.views.View, short name "View" should match
	if !matchesReceiverType("django.views.generic.ListView", "View", checker) {
		t.Error("Expected ListView to match short name 'View' via MRO")
	}
}

func TestMatchesReceiverType_InheritanceNoMatch(t *testing.T) {
	checker := newMockChecker()
	// View's MRO does NOT include "django.db.models.Model"
	if matchesReceiverType("django.views.View", "django.db.models.Model", checker) {
		t.Error("Expected View to NOT match Model")
	}
}

func TestMatchesReceiverType_InheritanceUnknownModule(t *testing.T) {
	checker := newMockChecker()
	// flask module not in checker
	if matchesReceiverType("flask.views.View", "django.views.View", checker) {
		t.Error("Expected unknown module to not match via inheritance")
	}
}

// --- TypeConstrainedCallExecutor tests ---

func TestTypeConstrainedCallExecutor_BasicMatch(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.TestView.get": {
				{
					Target:                   "response.json",
					Location:                 core.Location{File: "views.py", Line: 10},
					ResolvedViaTypeInference: true,
					InferredType:             "django.views.View",
					TypeConfidence:           0.95,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "django.views.View",
			MethodName:    "json",
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(results))
	}
	if results[0].SinkCall != "response.json" {
		t.Errorf("Expected SinkCall 'response.json', got %q", results[0].SinkCall)
	}
}

func TestTypeConstrainedCallExecutor_InheritanceMatch(t *testing.T) {
	checker := newMockChecker()
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.MyListView.get": {
				{
					Target:                   "self.get_queryset",
					Location:                 core.Location{File: "views.py", Line: 15},
					ResolvedViaTypeInference: true,
					InferredType:             "django.views.generic.ListView",
					TypeConfidence:           0.90,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "django.views.View",
			MethodName:    "get_queryset",
			MinConfidence: 0.5,
		},
		CallGraph:        cg,
		ThirdPartyRemote: checker,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 inheritance match, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_NoMatchWrongMethod(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.TestView.get": {
				{
					Target:                   "response.json",
					Location:                 core.Location{File: "views.py", Line: 10},
					ResolvedViaTypeInference: true,
					InferredType:             "django.views.View",
					TypeConfidence:           0.95,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "django.views.View",
			MethodName:    "post", // Wrong method
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches for wrong method, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_LowConfidence(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.TestView.get": {
				{
					Target:                   "response.json",
					Location:                 core.Location{File: "views.py", Line: 10},
					ResolvedViaTypeInference: true,
					InferredType:             "django.views.View",
					TypeConfidence:           0.3, // Below threshold
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "django.views.View",
			MethodName:    "json",
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches for low confidence, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_FallbackName(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.TestView.get": {
				{
					Target:                   "response.json",
					Location:                 core.Location{File: "views.py", Line: 10},
					ResolvedViaTypeInference: false, // No type info
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType: "django.views.View",
			MethodName:   "json",
			FallbackMode: "name", // Fall back to name match
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match with fallback=name, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_FallbackNone(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.TestView.get": {
				{
					Target:                   "response.json",
					Location:                 core.Location{File: "views.py", Line: 10},
					ResolvedViaTypeInference: false, // No type info
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType: "django.views.View",
			MethodName:   "json",
			FallbackMode: "none", // Strict: require type info
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches with fallback=none, got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_NoFalsePositive(t *testing.T) {
	checker := newMockChecker()
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.utils.SomeClass.get": {
				{
					Target:                   "dict.get",
					Location:                 core.Location{File: "utils.py", Line: 5},
					ResolvedViaTypeInference: true,
					InferredType:             "builtins.dict",
					TypeConfidence:           1.0,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "django.views.View",
			MethodName:    "get",
			MinConfidence: 0.5,
		},
		CallGraph:        cg,
		ThirdPartyRemote: checker,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches for dict.get (not a View), got %d", len(results))
	}
}

func TestTypeConstrainedCallExecutor_EmptyMethodName(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.TestView.get": {
				{
					Target:                   "any_call",
					Location:                 core.Location{File: "views.py", Line: 10},
					ResolvedViaTypeInference: true,
					InferredType:             "django.views.View",
					TypeConfidence:           0.95,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "django.views.View",
			MethodName:    "", // Match any method
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match with empty method name, got %d", len(results))
	}
}

// --- TypeConstrainedAttributeExecutor tests ---

func TestTypeConstrainedAttributeExecutor_BasicMatch(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.TestView.get": {
				{
					Target:                   "request.GET",
					Location:                 core.Location{File: "views.py", Line: 12},
					ResolvedViaTypeInference: true,
					InferredType:             "django.http.HttpRequest",
					TypeConfidence:           0.90,
				},
			},
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "django.http.HttpRequest",
			AttributeName: "GET",
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 attribute match, got %d", len(results))
	}
	if results[0].SinkCall != "request.GET" {
		t.Errorf("Expected SinkCall 'request.GET', got %q", results[0].SinkCall)
	}
}

func TestTypeConstrainedAttributeExecutor_WrongAttribute(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.TestView.get": {
				{
					Target:                   "request.POST",
					Location:                 core.Location{File: "views.py", Line: 12},
					ResolvedViaTypeInference: true,
					InferredType:             "django.http.HttpRequest",
					TypeConfidence:           0.90,
				},
			},
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "django.http.HttpRequest",
			AttributeName: "GET", // Looking for GET, not POST
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches for wrong attribute, got %d", len(results))
	}
}

func TestTypeConstrainedAttributeExecutor_InheritanceMatch(t *testing.T) {
	checker := newMockChecker()
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.TestView.get": {
				{
					Target:                   "self.request.GET",
					Location:                 core.Location{File: "views.py", Line: 12},
					ResolvedViaTypeInference: true,
					InferredType:             "django.http.HttpRequest",
					TypeConfidence:           0.90,
				},
			},
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "HttpRequest", // Short name
			AttributeName: "GET",
			MinConfidence: 0.5,
		},
		CallGraph:        cg,
		ThirdPartyRemote: checker,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match with short name, got %d", len(results))
	}
}

func TestTypeConstrainedAttributeExecutor_EmptyAttributeName(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.test": {
				{
					Target:                   "request.GET",
					Location:                 core.Location{File: "test.py", Line: 1},
					ResolvedViaTypeInference: true,
					InferredType:             "django.http.HttpRequest",
					TypeConfidence:           0.90,
				},
			},
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "django.http.HttpRequest",
			AttributeName: "", // Empty
			MinConfidence: 0.5,
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches with empty attribute name, got %d", len(results))
	}
}

// --- matchesWildcardType tests ---

func TestMatchesWildcardType(t *testing.T) {
	tests := []struct {
		actual, pattern string
		want            bool
	}{
		{"sqlite3.Cursor", "*Cursor", true},
		{"sqlite3.Cursor", "sqlite3.*", true},
		{"anything", "*", true},
		{"django.views.ListView", "*View*", true},
		{"django.views.View", "*Model*", false},
		{"exact.match", "exact.match", true},
		{"not.match", "different", false},
	}

	for _, tt := range tests {
		t.Run(tt.actual+"_"+tt.pattern, func(t *testing.T) {
			got := matchesWildcardType(tt.actual, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesWildcardType(%q, %q) = %v, want %v", tt.actual, tt.pattern, got, tt.want)
			}
		})
	}
}

// --- splitTypeModuleAndClass tests ---

func TestSplitTypeModuleAndClass(t *testing.T) {
	tests := []struct {
		fqn        string
		wantModule string
		wantClass  string
	}{
		{"django.views.View", "django", "views.View"},
		{"builtins.str", "builtins", "str"},
		{"solo", "solo", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fqn, func(t *testing.T) {
			mod, cls := splitTypeModuleAndClass(tt.fqn)
			if mod != tt.wantModule || cls != tt.wantClass {
				t.Errorf("splitTypeModuleAndClass(%q) = (%q, %q), want (%q, %q)",
					tt.fqn, mod, cls, tt.wantModule, tt.wantClass)
			}
		})
	}
}

// --- Loader routing tests ---

func TestExecuteRule_TypeConstrainedCall(t *testing.T) {
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

	loader := NewRuleLoader("")
	rule := &RuleIR{
		Matcher: map[string]any{
			"type":          "type_constrained_call",
			"receiverType":  "django.views.View",
			"methodName":    "method",
			"minConfidence": 0.5,
		},
	}

	results, err := loader.ExecuteRule(rule, cg)
	if err != nil {
		t.Fatalf("ExecuteRule failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

// TestExecuteRule_TypeConstrainedCall_InheritanceViaLoader tests that MRO-based
// inheritance matching works when going through the RuleLoader path.
// This was BUG-1: ThirdPartyRemote was never passed to the executor, making
// inheritance matching dead code in production.
func TestExecuteRule_TypeConstrainedCall_InheritanceViaLoader(t *testing.T) {
	checker := newMockChecker()

	// Call site has InferredType="django.views.generic.ListView"
	// Rule targets "django.views.View" — should match via MRO inheritance
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.MyListView.get": {
				{
					Target:                   "self.get_queryset",
					Location:                 core.Location{File: "views.py", Line: 15},
					ResolvedViaTypeInference: true,
					InferredType:             "django.views.generic.ListView",
					TypeConfidence:           0.90,
				},
			},
		},
		ThirdPartyRemote: checker, // BUG-1 fix: stored on CallGraph
	}

	loader := NewRuleLoader("")
	rule := &RuleIR{
		Matcher: map[string]any{
			"type":          "type_constrained_call",
			"receiverType":  "django.views.View", // parent class
			"methodName":    "get_queryset",
			"minConfidence": 0.5,
			"fallbackMode":  "none", // no fallback — must match via inheritance
		},
	}

	results, err := loader.ExecuteRule(rule, cg)
	if err != nil {
		t.Fatalf("ExecuteRule failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 inheritance match via RuleLoader path, got %d (BUG-1 not fixed)", len(results))
	}
}

// TestExecuteRule_TypeConstrainedCall_InheritanceViaLoader_NoMatch verifies that
// inheritance matching correctly rejects non-subclass types.
func TestExecuteRule_TypeConstrainedCall_InheritanceViaLoader_NoMatch(t *testing.T) {
	checker := newMockChecker()

	// Call site has InferredType="flask.views.MethodView" — NOT a django subclass
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.views.FlaskView.get": {
				{
					Target:                   "self.get_queryset",
					Location:                 core.Location{File: "views.py", Line: 15},
					ResolvedViaTypeInference: true,
					InferredType:             "flask.views.MethodView",
					TypeConfidence:           0.90,
				},
			},
		},
		ThirdPartyRemote: checker,
	}

	loader := NewRuleLoader("")
	rule := &RuleIR{
		Matcher: map[string]any{
			"type":          "type_constrained_call",
			"receiverType":  "django.views.View",
			"methodName":    "get_queryset",
			"minConfidence": 0.5,
			"fallbackMode":  "none",
		},
	}

	results, err := loader.ExecuteRule(rule, cg)
	if err != nil {
		t.Fatalf("ExecuteRule failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Expected 0 results (flask.views.MethodView is not a django subclass), got %d", len(results))
	}
}

func TestExecuteRule_TypeConstrainedAttribute(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.test": {
				{
					Target:                   "request.GET",
					Location:                 core.Location{File: "test.py", Line: 5},
					ResolvedViaTypeInference: true,
					InferredType:             "django.http.HttpRequest",
					TypeConfidence:           0.95,
				},
			},
		},
	}

	loader := NewRuleLoader("")
	rule := &RuleIR{
		Matcher: map[string]any{
			"type":          "type_constrained_attribute",
			"receiverType":  "django.http.HttpRequest",
			"attributeName": "GET",
			"minConfidence": 0.5,
		},
	}

	results, err := loader.ExecuteRule(rule, cg)
	if err != nil {
		t.Fatalf("ExecuteRule failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

// BUG-6 Tests: Attribute executor FQN bridge

func TestAttributeExecutor_FQNBridge(t *testing.T) {
	// Attribute access resolved via FQN (no type inference)
	cg := core.NewCallGraph()
	cg.CallSites["app.view"] = []core.CallSite{
		{
			Target:    "request.GET",
			TargetFQN: "django.http.HttpRequest.GET",
			Location:  core.Location{File: "views.py", Line: 5},
			// NOT resolved via type inference
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "django.http.HttpRequest",
			AttributeName: "GET",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match via FQN bridge, got %d (BUG-6)", len(results))
	}
}

func TestAttributeExecutor_FQNPrefix(t *testing.T) {
	// Attribute access where FQN prefix matches receiver
	cg := core.NewCallGraph()
	cg.CallSites["app.view"] = []core.CallSite{
		{
			Target:    "request.GET",
			TargetFQN: "django.http.HttpRequest.GET",
			Location:  core.Location{File: "views.py", Line: 5},
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "django.http.HttpRequest",
			AttributeName: "GET",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match via FQN prefix, got %d (BUG-6)", len(results))
	}
}

func TestAttributeExecutor_NoFQN_NoTypeInference_Rejects(t *testing.T) {
	// No FQN, no type inference, fallback=none → should NOT match
	cg := core.NewCallGraph()
	cg.CallSites["app.view"] = []core.CallSite{
		{
			Target:   "request.GET",
			Location: core.Location{File: "views.py", Line: 5},
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "django.http.HttpRequest",
			AttributeName: "GET",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches with no FQN/type info and fallback=none, got %d", len(results))
	}
}

// BUG-4 Tests: MatchMethod tracking

func TestMatchMethod_TypeInference(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:                   "cursor.execute",
			Location:                 core.Location{File: "app.py", Line: 5},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "sqlite3.Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].MatchMethod != "type_inference" {
		t.Errorf("Expected MatchMethod=type_inference, got %q", results[0].MatchMethod)
	}
}

func TestMatchMethod_FQNBridge(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:    "cursor.execute",
			TargetFQN: "sqlite3.Cursor.execute",
			Location:  core.Location{File: "app.py", Line: 5},
			// NOT resolved via type inference — uses FQN bridge
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "sqlite3.Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].MatchMethod != "fqn_bridge" {
		t.Errorf("Expected MatchMethod=fqn_bridge, got %q", results[0].MatchMethod)
	}
}

func TestMatchMethod_FQNPrefix(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:    "request.args.get",
			TargetFQN: "flask.request.args.get",
			Location:  core.Location{File: "app.py", Line: 5},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "flask.request",
			MethodName:    "get",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].MatchMethod != "fqn_prefix" {
		t.Errorf("Expected MatchMethod=fqn_prefix, got %q", results[0].MatchMethod)
	}
}

func TestMatchMethod_NameFallback(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:   "something.execute",
			Location: core.Location{File: "app.py", Line: 5},
			// No type inference, no FQN — falls back to name
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "sqlite3.Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "name",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 result via fallback, got %d", len(results))
	}
	if results[0].MatchMethod != "name_fallback" {
		t.Errorf("Expected MatchMethod=name_fallback, got %q (BUG-4)", results[0].MatchMethod)
	}
}

func TestMatchMethod_NoneRejectsFallback(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:   "something.execute",
			Location: core.Location{File: "app.py", Line: 5},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "sqlite3.Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 results with fallbackMode=none, got %d", len(results))
	}
}

// BUG-7 Tests: Userland FQN format sensitivity

func TestBUG7_FQNPrefixMismatch_Demonstrated(t *testing.T) {
	// Demonstrates the FQN prefix mismatch between import-style and file-discovery-style FQNs.
	// Rule uses import-style FQN, call site has file-discovery-style (stripped) FQN.
	cg := core.NewCallGraph()
	cg.CallSites["core.utils.helper"] = []core.CallSite{
		{
			Target:    "obj.method",
			TargetFQN: "core.utils.MyClass.method", // stripped by resolveWithPrefixStripping
			Location:  core.Location{File: "core/utils.py", Line: 5},
		},
	}

	// Rule with FULL import path — won't match stripped FQN
	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "label_studio.core.utils.MyClass", // full import path
			MethodName:    "method",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	// This SHOULD match but currently DOESN'T because:
	// - FQN bridge: derives "core.utils.MyClass" from TargetFQN, doesn't match "label_studio.core.utils.MyClass"
	// - FQN prefix: "core.utils.MyClass.method" doesn't start with "label_studio.core.utils.MyClass"
	if len(results) != 0 {
		t.Log("BUG-7: Unexpectedly matched — prefix stripping may have been applied elsewhere")
	} else {
		t.Log("BUG-7 CONFIRMED: Full import path doesn't match stripped FQN (known edge case for mono-repo projects)")
	}
}

func TestBUG7_StrippedFQN_WorksWithStrippedRule(t *testing.T) {
	// When rule uses the SAME stripped FQN format as the call site, it works fine.
	cg := core.NewCallGraph()
	cg.CallSites["core.utils.helper"] = []core.CallSite{
		{
			Target:    "obj.method",
			TargetFQN: "core.utils.MyClass.method",
			Location:  core.Location{File: "core/utils.py", Line: 5},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "core.utils.MyClass", // matches stripped FQN
			MethodName:    "method",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph: cg,
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Errorf("Expected 1 match when rule uses stripped FQN, got %d", len(results))
	}
}

// BUG-3 Tests: Stdlib MRO/Inheritance Support

func TestCompositeInheritanceChecker_BothRegistries(t *testing.T) {
	// Simulate third-party checker (django)
	thirdParty := newMockChecker()

	// Simulate stdlib checker (io module with MRO data)
	stdlib := &mockInheritanceChecker{
		modules: map[string]bool{
			"io": true,
		},
		classes: map[string]mockClassInfo{
			"io.FileIO": {
				mro: []string{"io.FileIO", "io.RawIOBase", "io.IOBase", "builtins.object"},
			},
			"io.BufferedReader": {
				mro: []string{"io.BufferedReader", "io.BufferedIOBase", "io.IOBase", "builtins.object"},
			},
		},
	}

	composite := &compositeInheritanceChecker{checkers: []InheritanceChecker{thirdParty, stdlib}}

	// Third-party module check
	if !composite.HasModule("django") {
		t.Error("Should find django module via third-party checker")
	}
	// Stdlib module check
	if !composite.HasModule("io") {
		t.Error("Should find io module via stdlib checker")
	}
	// Unknown module
	if composite.HasModule("nonexistent") {
		t.Error("Should not find nonexistent module")
	}

	// Third-party subclass check
	if !composite.IsSubclassSimple("django", "views.generic.ListView", "django.views.View") {
		t.Error("Should detect ListView as subclass of View via third-party")
	}
	// Stdlib subclass check
	if !composite.IsSubclassSimple("io", "FileIO", "io.IOBase") {
		t.Error("Should detect FileIO as subclass of IOBase via stdlib")
	}
	if !composite.IsSubclassSimple("io", "BufferedReader", "io.IOBase") {
		t.Error("Should detect BufferedReader as subclass of IOBase via stdlib")
	}

	// Stdlib MRO retrieval
	mro := composite.GetClassMRO("io", "FileIO")
	if len(mro) != 4 {
		t.Errorf("Expected 4-entry MRO for FileIO, got %d", len(mro))
	}
}

func TestStdlibMRO_MatchesReceiverType_ViaComposite(t *testing.T) {
	// Rule targets io.IOBase, call site has InferredType=io.FileIO
	// Should match via MRO inheritance through stdlib checker

	stdlib := &mockInheritanceChecker{
		modules: map[string]bool{"io": true},
		classes: map[string]mockClassInfo{
			"io.FileIO": {
				mro: []string{"io.FileIO", "io.RawIOBase", "io.IOBase", "builtins.object"},
			},
		},
	}
	composite := &compositeInheritanceChecker{checkers: []InheritanceChecker{stdlib}}

	// matchesReceiverType should find io.FileIO is a subclass of io.IOBase
	if !matchesReceiverType("io.FileIO", "io.IOBase", composite) {
		t.Error("io.FileIO should match io.IOBase via MRO inheritance (BUG-3 fix)")
	}
}

func TestStdlibMRO_E2E_TypeConstrainedCall(t *testing.T) {
	// End-to-end: rule for io.IOBase.read() should match call site with InferredType=io.FileIO

	stdlib := &mockInheritanceChecker{
		modules: map[string]bool{"io": true},
		classes: map[string]mockClassInfo{
			"io.FileIO": {
				mro: []string{"io.FileIO", "io.RawIOBase", "io.IOBase", "builtins.object"},
			},
		},
	}

	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:                   "f.read",
			Location:                 core.Location{File: "app.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "io.FileIO",
			TypeConfidence:           0.90,
		},
	}
	// Store stdlib checker — simulates CDN with MRO data populated
	cg.StdlibRemote = stdlib

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverType:  "io.IOBase",
			MethodName:    "read",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph:        cg,
		ThirdPartyRemote: extractInheritanceChecker(cg),
	}

	results := executor.Execute()
	if len(results) != 1 {
		t.Fatalf("Expected 1 match for io.IOBase.read() via FileIO inheritance, got %d (BUG-3)", len(results))
	}
	if results[0].SinkCall != "f.read" {
		t.Errorf("Expected SinkCall=f.read, got %s", results[0].SinkCall)
	}
}

func TestStdlibMRO_EmptyMRO_NoFalsePositive(t *testing.T) {
	// When CDN doesn't populate MRO (current state), no false matches

	stdlib := &mockInheritanceChecker{
		modules: map[string]bool{"io": true},
		classes: map[string]mockClassInfo{
			// FileIO exists but has EMPTY MRO (current CDN state)
			"io.FileIO": {mro: []string{}},
		},
	}

	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:                   "f.read",
			Location:                 core.Location{File: "app.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "io.FileIO",
			TypeConfidence:           0.90,
		},
	}
	cg.StdlibRemote = stdlib

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverType:  "io.IOBase",
			MethodName:    "read",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph:        cg,
		ThirdPartyRemote: extractInheritanceChecker(cg),
	}

	results := executor.Execute()
	if len(results) != 0 {
		t.Errorf("Expected 0 matches when MRO is empty (current CDN state), got %d", len(results))
	}
}

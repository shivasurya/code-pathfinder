package dsl

import (
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
	for _, ancestor := range info.mro {
		if ancestor == parentFQN {
			return true
		}
	}
	return false
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

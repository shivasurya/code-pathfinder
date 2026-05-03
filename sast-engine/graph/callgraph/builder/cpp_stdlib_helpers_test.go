package builder

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func newTestStdlibFn(fqn, returnType string) *core.CStdlibFunction {
	return &core.CStdlibFunction{FQN: fqn, ReturnType: returnType}
}

func TestCanonicalizeStdlibType(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain class", "std::string", "std::string"},
		{"single template arg", "std::vector<int>", "std::vector"},
		{"nested template", "std::map<std::string, std::vector<int>>", "std::map"},
		{"primitive", "int", "int"},
		{"empty string", "", ""},
		{"leading bracket is non-canonical, treated as plain", "<bad", "<bad"},
		{"whitespace before angle", "std::vector <int>", "std::vector"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, canonicalizeStdlibType(tt.in))
		})
	}
}

func TestParseTemplateArgs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"no template", "std::string", nil},
		{"unmatched open", "std::vector<int", nil},
		{"close before open", "int>std::vector<", nil},
		{"single arg", "std::vector<int>", []string{"int"}},
		{"two args", "std::map<std::string, int>", []string{"std::string", "int"}},
		{"nested template arg", "std::map<int, std::vector<int>>", []string{"int", "std::vector<int>"}},
		{"deeply nested", "A<B<C, D<E, F>>, G>", []string{"B<C, D<E, F>>", "G"}},
		{"trailing whitespace trimmed", "std::pair< int , long >", []string{"int", "long"}},
		{"empty body", "x<>", []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseTemplateArgs(tt.in))
		})
	}
}

func TestApplyTemplateSubstitution(t *testing.T) {
	tests := []struct {
		name       string
		returnType string
		args       []string
		want       string
	}{
		{"single T", "T", []string{"int"}, "int"},
		{"reference to T", "T&", []string{"int"}, "int&"},
		{"const T pointer", "const T*", []string{"std::string"}, "const std::string*"},
		{"pair of T and U", "std::pair<T, U>", []string{"int", "long"}, "std::pair<int, long>"},
		{"K alias for first arg", "K", []string{"std::string", "int"}, "std::string"},
		{"V third arg", "std::tuple<T, U, V>", []string{"int", "long", "char"}, "std::tuple<int, long, char>"},
		{"non-placeholder T-prefix preserved", "Type", []string{"int"}, "Type"},
		{"only V with one arg leaves V untouched", "V", []string{"int"}, "V"},
		{"no placeholders", "iterator", []string{"int"}, "iterator"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, applyTemplateSubstitution(tt.returnType, tt.args))
		})
	}
}

func TestReplaceWholeWord(t *testing.T) {
	tests := []struct {
		name string
		s    string
		from string
		to   string
		want string
	}{
		{"empty needle returns input", "anything", "", "X", "anything"},
		{"start of string", "T&", "T", "int", "int&"},
		{"end of string", "&T", "T", "int", "&int"},
		{"middle word", "const T&", "T", "int", "const int&"},
		{"not a word boundary", "Type", "T", "X", "Type"},
		{"multiple replacements", "T, T", "T", "int", "int, int"},
		{"adjacent identifiers preserved", "const TX", "T", "int", "const TX"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, replaceWholeWord(tt.s, tt.from, tt.to))
		})
	}
}

func TestSubstituteTemplateMethodReturn_NoArgs(t *testing.T) {
	method := newTestStdlibFn("std::vector::push_back", "void")
	got := substituteTemplateMethodReturn(method, "std::vector")
	assert.Same(t, method, got, "no template args => return the same pointer (no allocation)")
}

func TestSubstituteTemplateMethodReturn_ShallowCopyOnSubstitution(t *testing.T) {
	method := newTestStdlibFn("std::vector::front", "T&")
	got := substituteTemplateMethodReturn(method, "std::vector<int>")
	assert.NotSame(t, method, got, "substitution must not mutate the registry record")
	assert.Equal(t, "T&", method.ReturnType, "original record stays unchanged")
	assert.Equal(t, "int&", got.ReturnType)
}

func TestLookupCppStdlibMethod_NilTypeEngine(t *testing.T) {
	cs := &CallSiteInternal{CallerFQN: "src/main.cpp::main", FunctionName: "push_back", ObjectName: "vec"}
	fqn, fn := lookupCppStdlibMethod(cs, core.NewCppModuleRegistry("/proj"), nil)
	assert.Empty(t, fqn)
	assert.Nil(t, fn)
}

func TestLookupCppStdlibFreeFunction_RequiresQualifiedName(t *testing.T) {
	cs := &CallSiteInternal{CallerFQN: "src/main.cpp::main", FunctionName: "move"} // bare, no namespace
	cReg := core.NewCModuleRegistry("/proj")
	cReg.FileToPrefix["/proj/src/main.cpp"] = "src/main.cpp"
	fqn, fn := lookupCppStdlibFreeFunction(cs, cReg, nil)
	assert.Empty(t, fqn)
	assert.Nil(t, fn)
}

func TestLookupCppStdlibFreeFunction_FileNotInRegistry(t *testing.T) {
	cs := &CallSiteInternal{CallerFile: "/unknown.cpp", CallerFQN: "x::y", FunctionName: "std::move"}
	fqn, fn := lookupCppStdlibFreeFunction(cs, core.NewCModuleRegistry("/proj"), nil)
	assert.Empty(t, fqn)
	assert.Nil(t, fn)
}

func TestLookupCStdlib_FileNotInRegistry(t *testing.T) {
	fqn, fn := lookupCStdlib("/unknown.c", "printf", core.NewCModuleRegistry("/proj"))
	assert.Empty(t, fqn)
	assert.Nil(t, fn)
}

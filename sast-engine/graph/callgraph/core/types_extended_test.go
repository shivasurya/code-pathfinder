package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CONCRETE TYPE TESTS
// =============================================================================

func TestNewConcreteType_FullyQualified(t *testing.T) {
	ct := NewConcreteType("myapp.models.User", 0.9)

	assert.Equal(t, "User", ct.Name)
	assert.Equal(t, "myapp.models", ct.Module)
	assert.Equal(t, "myapp.models.User", ct.FQN())
	assert.Equal(t, 0.9, ct.Confidence())
}

func TestNewConcreteType_BuiltinNoModule(t *testing.T) {
	ct := NewConcreteType("str", 0.95)

	assert.Equal(t, "str", ct.Name)
	assert.Equal(t, "", ct.Module)
	assert.Equal(t, "str", ct.FQN())
}

func TestConcreteType_Equals(t *testing.T) {
	t1 := NewConcreteType("myapp.User", 0.9)
	t2 := NewConcreteType("myapp.User", 0.8) // Different confidence
	t3 := NewConcreteType("myapp.Admin", 0.9)

	assert.True(t, t1.Equals(t2), "Same FQN should be equal regardless of confidence")
	assert.False(t, t1.Equals(t3), "Different names should not be equal")
	assert.False(t, t1.Equals(&NoneType{}), "Different types should not be equal")
}

func TestConcreteType_String(t *testing.T) {
	tests := []struct {
		name     string
		ct       *ConcreteType
		expected string
	}{
		{"with module", NewConcreteType("myapp.models.User", 0.9), "myapp.models.User"},
		{"builtin", NewConcreteType("str", 0.95), "str"},
		{"nested module", NewConcreteType("a.b.c.d.Class", 0.8), "a.b.c.d.Class"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ct.String())
		})
	}
}

// =============================================================================
// TYPE VARIABLE TESTS
// =============================================================================

func TestTypeVariable_String(t *testing.T) {
	tv1 := &TypeVariable{ID: 1, Name: ""}
	tv2 := &TypeVariable{ID: 2, Name: "T"}

	assert.Equal(t, "$T_1", tv1.String())
	assert.Equal(t, "$T_2", tv2.String())
}

func TestTypeVariable_Equals(t *testing.T) {
	tv1 := &TypeVariable{ID: 1}
	tv2 := &TypeVariable{ID: 1}
	tv3 := &TypeVariable{ID: 2}

	assert.True(t, tv1.Equals(tv2))
	assert.False(t, tv1.Equals(tv3))
	assert.False(t, tv1.Equals(&NoneType{}))
}

func TestTypeVariable_Confidence(t *testing.T) {
	tv := &TypeVariable{ID: 1}
	assert.Equal(t, 0.0, tv.Confidence(), "Unresolved type variable has zero confidence")
}

// =============================================================================
// UNION TYPE TESTS
// =============================================================================

func TestNewUnionType_Basic(t *testing.T) {
	types := []Type{
		NewConcreteType("str", 0.9),
		&NoneType{},
	}
	union := NewUnionType(types, 0.85)

	assert.Len(t, union.Types, 2)
	assert.Equal(t, 0.85, union.Confidence())
	assert.Contains(t, union.String(), "Union[")
}

func TestNewUnionType_FlattenNested(t *testing.T) {
	inner := NewUnionType([]Type{
		NewConcreteType("str", 0.9),
		NewConcreteType("int", 0.9),
	}, 0.9)

	outer := NewUnionType([]Type{
		inner,
		NewConcreteType("float", 0.9),
	}, 0.85)

	assert.Len(t, outer.Types, 3, "Nested unions should be flattened")
}

func TestNewUnionType_DeduplicateTypes(t *testing.T) {
	types := []Type{
		NewConcreteType("str", 0.9),
		NewConcreteType("str", 0.8), // Duplicate
		NewConcreteType("int", 0.9),
	}
	union := NewUnionType(types, 0.85)

	assert.Len(t, union.Types, 2, "Duplicate types should be removed")
}

func TestUnionType_Equals(t *testing.T) {
	u1 := NewUnionType([]Type{
		NewConcreteType("str", 0.9),
		NewConcreteType("int", 0.9),
	}, 0.9)

	u2 := NewUnionType([]Type{
		NewConcreteType("int", 0.8), // Different order
		NewConcreteType("str", 0.7),
	}, 0.8)

	u3 := NewUnionType([]Type{
		NewConcreteType("str", 0.9),
		NewConcreteType("float", 0.9),
	}, 0.9)

	assert.True(t, u1.Equals(u2), "Unions with same members should be equal regardless of order")
	assert.False(t, u1.Equals(u3), "Unions with different members should not be equal")
}

// =============================================================================
// ANY TYPE TESTS
// =============================================================================

func TestAnyType_String(t *testing.T) {
	a1 := &AnyType{}
	a2 := &AnyType{Reason: "dynamic getattr"}

	assert.Equal(t, "Any", a1.String())
	assert.Equal(t, "Any<dynamic getattr>", a2.String())
}

func TestAnyType_FQN(t *testing.T) {
	a := &AnyType{}
	assert.Equal(t, "typing.Any", a.FQN())
}

func TestAnyType_Confidence(t *testing.T) {
	a := &AnyType{}
	assert.Equal(t, 0.0, a.Confidence())
}

// =============================================================================
// NONE TYPE TESTS
// =============================================================================

func TestNoneType(t *testing.T) {
	n := &NoneType{}

	assert.Equal(t, "None", n.String())
	assert.Equal(t, "builtins.NoneType", n.FQN())
	assert.Equal(t, 1.0, n.Confidence())
	assert.True(t, n.Equals(&NoneType{}))
	assert.False(t, n.Equals(&AnyType{}))
}

// =============================================================================
// FUNCTION TYPE TESTS
// =============================================================================

func TestFunctionType_String(t *testing.T) {
	ft := &FunctionType{
		Parameters: []Type{
			NewConcreteType("str", 0.9),
			NewConcreteType("int", 0.9),
		},
		ReturnType: NewConcreteType("bool", 0.9),
		confidence: 0.9,
	}

	assert.Equal(t, "Callable[[str, int], bool]", ft.String())
}

func TestFunctionType_NoParams(t *testing.T) {
	ft := &FunctionType{
		Parameters: []Type{},
		ReturnType: NewConcreteType("str", 0.9),
		confidence: 0.9,
	}

	assert.Equal(t, "Callable[[], str]", ft.String())
}

func TestFunctionType_NilReturnType(t *testing.T) {
	ft := &FunctionType{
		Parameters: []Type{NewConcreteType("str", 0.9)},
		ReturnType: nil,
		confidence: 0.9,
	}

	assert.Equal(t, "Callable[[str], None]", ft.String())
}

func TestFunctionType_Equals(t *testing.T) {
	ft1 := &FunctionType{
		Parameters: []Type{NewConcreteType("str", 0.9)},
		ReturnType: NewConcreteType("int", 0.9),
	}
	ft2 := &FunctionType{
		Parameters: []Type{NewConcreteType("str", 0.8)},
		ReturnType: NewConcreteType("int", 0.8),
	}
	ft3 := &FunctionType{
		Parameters: []Type{NewConcreteType("str", 0.9)},
		ReturnType: NewConcreteType("bool", 0.9),
	}

	assert.True(t, ft1.Equals(ft2))
	assert.False(t, ft1.Equals(ft3))
}

// =============================================================================
// CONFIDENCE SCORING TESTS
// =============================================================================

func TestConfidenceScore(t *testing.T) {
	tests := []struct {
		source   ConfidenceSource
		expected float64
	}{
		{ConfidenceAnnotation, 1.0},
		{ConfidenceLiteral, 0.95},
		{ConfidenceConstructor, 0.95},
		{ConfidenceReturnType, 0.9},
		{ConfidenceAssignment, 0.85},
		{ConfidenceAttribute, 0.8},
		{ConfidenceFluentHeuristic, 0.7},
		{ConfidenceUnknown, 0.0},
		{"invalid", 0.0},
	}

	for _, tt := range tests {
		t.Run(string(tt.source), func(t *testing.T) {
			assert.Equal(t, tt.expected, ConfidenceScore(tt.source))
		})
	}
}

func TestCombineConfidence(t *testing.T) {
	assert.Equal(t, 0.0, CombineConfidence())
	assert.Equal(t, 0.9, CombineConfidence(0.9))
	assert.InDelta(t, 0.81, CombineConfidence(0.9, 0.9), 0.001)
	assert.InDelta(t, 0.729, CombineConfidence(0.9, 0.9, 0.9), 0.001)
}

// =============================================================================
// TYPE UTILITY TESTS
// =============================================================================

func TestIsNoneType(t *testing.T) {
	assert.True(t, IsNoneType(&NoneType{}))
	assert.False(t, IsNoneType(&AnyType{}))
	assert.False(t, IsNoneType(NewConcreteType("str", 0.9)))
}

func TestIsAnyType(t *testing.T) {
	assert.True(t, IsAnyType(&AnyType{}))
	assert.False(t, IsAnyType(&NoneType{}))
	assert.False(t, IsAnyType(NewConcreteType("str", 0.9)))
}

func TestIsConcreteType(t *testing.T) {
	assert.True(t, IsConcreteType(NewConcreteType("str", 0.9)))
	assert.False(t, IsConcreteType(&NoneType{}))
	assert.False(t, IsConcreteType(&AnyType{}))
}

func TestExtractConcreteType(t *testing.T) {
	ct := NewConcreteType("myapp.User", 0.9)

	extracted, ok := ExtractConcreteType(ct)
	require.True(t, ok)
	assert.Equal(t, "User", extracted.Name)

	_, ok = ExtractConcreteType(&NoneType{})
	assert.False(t, ok)
}

func TestSimplifyUnion_EmptyUnion(t *testing.T) {
	empty := &UnionType{Types: []Type{}}
	result := SimplifyUnion(empty)

	assert.True(t, IsAnyType(result))
}

func TestSimplifyUnion_SingleType(t *testing.T) {
	single := NewUnionType([]Type{NewConcreteType("str", 0.9)}, 0.9)
	result := SimplifyUnion(single)

	ct, ok := ExtractConcreteType(result)
	require.True(t, ok)
	assert.Equal(t, "str", ct.Name)
}

func TestSimplifyUnion_AllSameType(t *testing.T) {
	same := NewUnionType([]Type{
		NewConcreteType("str", 0.9),
		NewConcreteType("str", 0.8),
	}, 0.85)
	result := SimplifyUnion(same)

	// After deduplication in NewUnionType, there's only one type
	ct, ok := ExtractConcreteType(result)
	require.True(t, ok)
	assert.Equal(t, "str", ct.Name)
}

func TestSimplifyUnion_ContainsAny(t *testing.T) {
	withAny := NewUnionType([]Type{
		NewConcreteType("str", 0.9),
		&AnyType{},
	}, 0.85)
	result := SimplifyUnion(withAny)

	assert.True(t, IsAnyType(result))
}

// =============================================================================
// TYPE INTERFACE COMPLIANCE TESTS
// =============================================================================

func TestAllTypesImplementTypeInterface(t *testing.T) {
	// Compile-time check that all types implement Type
	var _ Type = &ConcreteType{}
	var _ Type = &TypeVariable{}
	var _ Type = &UnionType{}
	var _ Type = &AnyType{}
	var _ Type = &NoneType{}
	var _ Type = &FunctionType{}
}

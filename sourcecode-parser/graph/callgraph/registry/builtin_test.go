package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewBuiltinRegistry tests registry initialization.
func TestNewBuiltinRegistry(t *testing.T) {
	registry := NewBuiltinRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.Types)

	// Verify all builtin types are registered
	expectedTypes := []string{
		"builtins.str",
		"builtins.list",
		"builtins.dict",
		"builtins.set",
		"builtins.tuple",
		"builtins.int",
		"builtins.float",
		"builtins.bool",
		"builtins.bytes",
	}

	for _, typeFQN := range expectedTypes {
		builtinType := registry.GetType(typeFQN)
		assert.NotNil(t, builtinType, "Type %s should be registered", typeFQN)
		assert.Equal(t, typeFQN, builtinType.FQN)
		assert.NotNil(t, builtinType.Methods)
	}
}

// TestBuiltinRegistry_GetType tests type retrieval.
func TestBuiltinRegistry_GetType(t *testing.T) {
	registry := NewBuiltinRegistry()

	tests := []struct {
		name     string
		typeFQN  string
		expected bool
	}{
		{name: "str type exists", typeFQN: "builtins.str", expected: true},
		{name: "list type exists", typeFQN: "builtins.list", expected: true},
		{name: "dict type exists", typeFQN: "builtins.dict", expected: true},
		{name: "unknown type", typeFQN: "builtins.unknown", expected: false},
		{name: "user type", typeFQN: "myapp.models.User", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.GetType(tt.typeFQN)
			if tt.expected {
				assert.NotNil(t, result)
				assert.Equal(t, tt.typeFQN, result.FQN)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

// TestBuiltinRegistry_GetMethod tests method retrieval.
func TestBuiltinRegistry_GetMethod(t *testing.T) {
	registry := NewBuiltinRegistry()

	tests := []struct {
		name         string
		typeFQN      string
		methodName   string
		expectedType string
		shouldExist  bool
	}{
		{
			name:         "str.upper returns str",
			typeFQN:      "builtins.str",
			methodName:   "upper",
			expectedType: "builtins.str",
			shouldExist:  true,
		},
		{
			name:         "str.isdigit returns bool",
			typeFQN:      "builtins.str",
			methodName:   "isdigit",
			expectedType: "builtins.bool",
			shouldExist:  true,
		},
		{
			name:         "list.append returns None",
			typeFQN:      "builtins.list",
			methodName:   "append",
			expectedType: "builtins.NoneType",
			shouldExist:  true,
		},
		{
			name:         "dict.keys returns dict_keys",
			typeFQN:      "builtins.dict",
			methodName:   "keys",
			expectedType: "builtins.dict_keys",
			shouldExist:  true,
		},
		{
			name:        "nonexistent method",
			typeFQN:     "builtins.str",
			methodName:  "nonexistent",
			shouldExist: false,
		},
		{
			name:        "method on nonexistent type",
			typeFQN:     "builtins.unknown",
			methodName:  "foo",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := registry.GetMethod(tt.typeFQN, tt.methodName)
			if tt.shouldExist {
				assert.NotNil(t, method)
				assert.Equal(t, tt.methodName, method.Name)
				assert.NotNil(t, method.ReturnType)
				assert.Equal(t, tt.expectedType, method.ReturnType.TypeFQN)
			} else {
				assert.Nil(t, method)
			}
		})
	}
}

// TestBuiltinRegistry_InferLiteralType_Strings tests string literal inference.
func TestBuiltinRegistry_InferLiteralType_Strings(t *testing.T) {
	registry := NewBuiltinRegistry()

	tests := []struct {
		name     string
		literal  string
		expected string
	}{
		{name: "double quoted string", literal: `"hello"`, expected: "builtins.str"},
		{name: "single quoted string", literal: `'world'`, expected: "builtins.str"},
		{name: "triple double quoted", literal: `"""multiline"""`, expected: "builtins.str"},
		{name: "triple single quoted", literal: `'''multiline'''`, expected: "builtins.str"},
		{name: "empty string", literal: `""`, expected: "builtins.str"},
		{name: "string with spaces", literal: `"  hello  "`, expected: "builtins.str"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.InferLiteralType(tt.literal)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.TypeFQN)
			assert.Equal(t, float32(1.0), result.Confidence)
			assert.Equal(t, "literal", result.Source)
		})
	}
}

// TestBuiltinRegistry_InferLiteralType_Numbers tests numeric literal inference.
func TestBuiltinRegistry_InferLiteralType_Numbers(t *testing.T) {
	registry := NewBuiltinRegistry()

	tests := []struct {
		name     string
		literal  string
		expected string
	}{
		{name: "integer", literal: "42", expected: "builtins.int"},
		{name: "negative integer", literal: "-123", expected: "builtins.int"},
		{name: "positive integer", literal: "+456", expected: "builtins.int"},
		{name: "zero", literal: "0", expected: "builtins.int"},
		{name: "hex integer", literal: "0xff", expected: "builtins.int"},
		{name: "octal integer", literal: "0o77", expected: "builtins.int"},
		{name: "binary integer", literal: "0b1010", expected: "builtins.int"},
		{name: "integer with underscores", literal: "1_000_000", expected: "builtins.int"},
		{name: "float", literal: "3.14", expected: "builtins.float"},
		{name: "negative float", literal: "-2.5", expected: "builtins.float"},
		{name: "scientific notation", literal: "1.5e10", expected: "builtins.float"},
		{name: "negative scientific", literal: "-3.2E-5", expected: "builtins.float"},
		{name: "float with underscores", literal: "1_000.5", expected: "builtins.float"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.InferLiteralType(tt.literal)
			assert.NotNil(t, result, "Failed to infer type for: %s", tt.literal)
			assert.Equal(t, tt.expected, result.TypeFQN)
			assert.Equal(t, float32(1.0), result.Confidence)
			assert.Equal(t, "literal", result.Source)
		})
	}
}

// TestBuiltinRegistry_InferLiteralType_Collections tests collection literal inference.
func TestBuiltinRegistry_InferLiteralType_Collections(t *testing.T) {
	registry := NewBuiltinRegistry()

	tests := []struct {
		name     string
		literal  string
		expected string
	}{
		{name: "empty list", literal: "[]", expected: "builtins.list"},
		{name: "list with elements", literal: "[1, 2, 3]", expected: "builtins.list"},
		{name: "empty dict", literal: "{}", expected: "builtins.dict"},
		{name: "dict with items", literal: `{"key": "value"}`, expected: "builtins.dict"},
		{name: "set with elements", literal: "{1, 2, 3}", expected: "builtins.set"},
		{name: "empty tuple", literal: "()", expected: "builtins.tuple"},
		{name: "tuple with elements", literal: "(1, 2, 3)", expected: "builtins.tuple"},
		{name: "single element tuple", literal: "(1,)", expected: "builtins.tuple"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.InferLiteralType(tt.literal)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.TypeFQN)
			assert.Equal(t, float32(1.0), result.Confidence)
			assert.Equal(t, "literal", result.Source)
		})
	}
}

// TestBuiltinRegistry_InferLiteralType_Special tests special literals.
func TestBuiltinRegistry_InferLiteralType_Special(t *testing.T) {
	registry := NewBuiltinRegistry()

	tests := []struct {
		name     string
		literal  string
		expected string
	}{
		{name: "True", literal: "True", expected: "builtins.bool"},
		{name: "False", literal: "False", expected: "builtins.bool"},
		{name: "None", literal: "None", expected: "builtins.NoneType"},
		{name: "bytes literal", literal: `b"data"`, expected: "builtins.bytes"},
		{name: "bytes with single quote", literal: `b'data'`, expected: "builtins.bytes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.InferLiteralType(tt.literal)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.TypeFQN)
			assert.Equal(t, float32(1.0), result.Confidence)
			assert.Equal(t, "literal", result.Source)
		})
	}
}

// TestBuiltinRegistry_InferLiteralType_Invalid tests invalid literals.
func TestBuiltinRegistry_InferLiteralType_Invalid(t *testing.T) {
	registry := NewBuiltinRegistry()

	tests := []struct {
		name    string
		literal string
	}{
		{name: "identifier", literal: "variable_name"},
		{name: "function call", literal: "foo()"},
		{name: "empty string", literal: ""},
		{name: "whitespace", literal: "   "},
		{name: "partial string", literal: `"unclosed`},
		{name: "invalid number", literal: "12.34.56"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.InferLiteralType(tt.literal)
			assert.Nil(t, result, "Should not infer type for: %s", tt.literal)
		})
	}
}

// TestBuiltinType_StringMethods tests str type methods.
func TestBuiltinType_StringMethods(t *testing.T) {
	registry := NewBuiltinRegistry()
	strType := registry.GetType("builtins.str")

	assert.NotNil(t, strType)

	// Test methods that return str
	strReturnMethods := []string{"upper", "lower", "strip", "replace", "capitalize"}
	for _, methodName := range strReturnMethods {
		method := strType.Methods[methodName]
		assert.NotNil(t, method, "Method %s should exist", methodName)
		assert.Equal(t, "builtins.str", method.ReturnType.TypeFQN)
	}

	// Test methods that return bool
	boolReturnMethods := []string{"isdigit", "isalpha", "startswith", "endswith"}
	for _, methodName := range boolReturnMethods {
		method := strType.Methods[methodName]
		assert.NotNil(t, method, "Method %s should exist", methodName)
		assert.Equal(t, "builtins.bool", method.ReturnType.TypeFQN)
	}

	// Test methods that return int
	intReturnMethods := []string{"find", "count", "index"}
	for _, methodName := range intReturnMethods {
		method := strType.Methods[methodName]
		assert.NotNil(t, method, "Method %s should exist", methodName)
		assert.Equal(t, "builtins.int", method.ReturnType.TypeFQN)
	}

	// Test methods that return list
	listReturnMethods := []string{"split", "splitlines"}
	for _, methodName := range listReturnMethods {
		method := strType.Methods[methodName]
		assert.NotNil(t, method, "Method %s should exist", methodName)
		assert.Equal(t, "builtins.list", method.ReturnType.TypeFQN)
	}

	// Test encode returns bytes
	encodeMethod := strType.Methods["encode"]
	assert.NotNil(t, encodeMethod)
	assert.Equal(t, "builtins.bytes", encodeMethod.ReturnType.TypeFQN)
}

// TestBuiltinType_ListMethods tests list type methods.
func TestBuiltinType_ListMethods(t *testing.T) {
	registry := NewBuiltinRegistry()
	listType := registry.GetType("builtins.list")

	assert.NotNil(t, listType)

	// Test methods that return None (mutating methods)
	noneMethods := []string{"append", "extend", "insert", "remove", "clear"}
	for _, methodName := range noneMethods {
		method := listType.Methods[methodName]
		assert.NotNil(t, method, "Method %s should exist", methodName)
		assert.Equal(t, "builtins.NoneType", method.ReturnType.TypeFQN)
	}

	// Test methods that return int
	countMethod := listType.Methods["count"]
	assert.NotNil(t, countMethod)
	assert.Equal(t, "builtins.int", countMethod.ReturnType.TypeFQN)

	// Test copy returns list
	copyMethod := listType.Methods["copy"]
	assert.NotNil(t, copyMethod)
	assert.Equal(t, "builtins.list", copyMethod.ReturnType.TypeFQN)

	// Test pop returns unknown type (confidence 0.0)
	popMethod := listType.Methods["pop"]
	assert.NotNil(t, popMethod)
	assert.Equal(t, "", popMethod.ReturnType.TypeFQN)
	assert.Equal(t, float32(0.0), popMethod.ReturnType.Confidence)
}

// TestBuiltinType_DictMethods tests dict type methods.
func TestBuiltinType_DictMethods(t *testing.T) {
	registry := NewBuiltinRegistry()
	dictType := registry.GetType("builtins.dict")

	assert.NotNil(t, dictType)

	// Test methods that return dict views
	keysMethod := dictType.Methods["keys"]
	assert.NotNil(t, keysMethod)
	assert.Equal(t, "builtins.dict_keys", keysMethod.ReturnType.TypeFQN)

	valuesMethod := dictType.Methods["values"]
	assert.NotNil(t, valuesMethod)
	assert.Equal(t, "builtins.dict_values", valuesMethod.ReturnType.TypeFQN)

	itemsMethod := dictType.Methods["items"]
	assert.NotNil(t, itemsMethod)
	assert.Equal(t, "builtins.dict_items", itemsMethod.ReturnType.TypeFQN)

	// Test copy returns dict
	copyMethod := dictType.Methods["copy"]
	assert.NotNil(t, copyMethod)
	assert.Equal(t, "builtins.dict", copyMethod.ReturnType.TypeFQN)

	// Test methods that return None
	clearMethod := dictType.Methods["clear"]
	assert.NotNil(t, clearMethod)
	assert.Equal(t, "builtins.NoneType", clearMethod.ReturnType.TypeFQN)

	// Test get returns unknown type
	getMethod := dictType.Methods["get"]
	assert.NotNil(t, getMethod)
	assert.Equal(t, "", getMethod.ReturnType.TypeFQN)
	assert.Equal(t, float32(0.0), getMethod.ReturnType.Confidence)
}

// TestBuiltinType_SetMethods tests set type methods.
func TestBuiltinType_SetMethods(t *testing.T) {
	registry := NewBuiltinRegistry()
	setType := registry.GetType("builtins.set")

	assert.NotNil(t, setType)

	// Test methods that return None
	addMethod := setType.Methods["add"]
	assert.NotNil(t, addMethod)
	assert.Equal(t, "builtins.NoneType", addMethod.ReturnType.TypeFQN)

	// Test methods that return set
	unionMethod := setType.Methods["union"]
	assert.NotNil(t, unionMethod)
	assert.Equal(t, "builtins.set", unionMethod.ReturnType.TypeFQN)

	// Test methods that return bool
	issubsetMethod := setType.Methods["issubset"]
	assert.NotNil(t, issubsetMethod)
	assert.Equal(t, "builtins.bool", issubsetMethod.ReturnType.TypeFQN)
}

// TestBuiltinType_TupleMethods tests tuple type methods.
func TestBuiltinType_TupleMethods(t *testing.T) {
	registry := NewBuiltinRegistry()
	tupleType := registry.GetType("builtins.tuple")

	assert.NotNil(t, tupleType)

	// Test count returns int
	countMethod := tupleType.Methods["count"]
	assert.NotNil(t, countMethod)
	assert.Equal(t, "builtins.int", countMethod.ReturnType.TypeFQN)

	// Test index returns int
	indexMethod := tupleType.Methods["index"]
	assert.NotNil(t, indexMethod)
	assert.Equal(t, "builtins.int", indexMethod.ReturnType.TypeFQN)
}

// TestBuiltinType_IntMethods tests int type methods.
func TestBuiltinType_IntMethods(t *testing.T) {
	registry := NewBuiltinRegistry()
	intType := registry.GetType("builtins.int")

	assert.NotNil(t, intType)

	// Test bit_length returns int
	bitLengthMethod := intType.Methods["bit_length"]
	assert.NotNil(t, bitLengthMethod)
	assert.Equal(t, "builtins.int", bitLengthMethod.ReturnType.TypeFQN)

	// Test to_bytes returns bytes
	toBytesMethod := intType.Methods["to_bytes"]
	assert.NotNil(t, toBytesMethod)
	assert.Equal(t, "builtins.bytes", toBytesMethod.ReturnType.TypeFQN)
}

// TestBuiltinType_FloatMethods tests float type methods.
func TestBuiltinType_FloatMethods(t *testing.T) {
	registry := NewBuiltinRegistry()
	floatType := registry.GetType("builtins.float")

	assert.NotNil(t, floatType)

	// Test is_integer returns bool
	isIntegerMethod := floatType.Methods["is_integer"]
	assert.NotNil(t, isIntegerMethod)
	assert.Equal(t, "builtins.bool", isIntegerMethod.ReturnType.TypeFQN)

	// Test hex returns str
	hexMethod := floatType.Methods["hex"]
	assert.NotNil(t, hexMethod)
	assert.Equal(t, "builtins.str", hexMethod.ReturnType.TypeFQN)
}

// TestBuiltinType_BoolType tests bool type.
func TestBuiltinType_BoolType(t *testing.T) {
	registry := NewBuiltinRegistry()
	boolType := registry.GetType("builtins.bool")

	assert.NotNil(t, boolType)
	assert.Equal(t, "builtins.bool", boolType.FQN)
	// Bool has no unique methods (inherits from int)
	assert.NotNil(t, boolType.Methods)
}

// TestBuiltinType_BytesMethods tests bytes type methods.
func TestBuiltinType_BytesMethods(t *testing.T) {
	registry := NewBuiltinRegistry()
	bytesType := registry.GetType("builtins.bytes")

	assert.NotNil(t, bytesType)

	// Test methods that return bytes
	upperMethod := bytesType.Methods["upper"]
	assert.NotNil(t, upperMethod)
	assert.Equal(t, "builtins.bytes", upperMethod.ReturnType.TypeFQN)

	// Test methods that return bool
	isdigitMethod := bytesType.Methods["isdigit"]
	assert.NotNil(t, isdigitMethod)
	assert.Equal(t, "builtins.bool", isdigitMethod.ReturnType.TypeFQN)

	// Test decode returns str
	decodeMethod := bytesType.Methods["decode"]
	assert.NotNil(t, decodeMethod)
	assert.Equal(t, "builtins.str", decodeMethod.ReturnType.TypeFQN)

	// Test hex returns str
	hexMethod := bytesType.Methods["hex"]
	assert.NotNil(t, hexMethod)
	assert.Equal(t, "builtins.str", hexMethod.ReturnType.TypeFQN)
}

// TestIsNumericLiteral tests numeric literal validation.
// Note: isNumericLiteral is a private function in the callgraph package,
// so we test it indirectly through InferLiteralType in the tests above.

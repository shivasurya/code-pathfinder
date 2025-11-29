package registry

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// BuiltinMethod represents a method available on a builtin type.
type BuiltinMethod struct {
	Name       string          // Method name (e.g., "upper", "append")
	ReturnType *core.TypeInfo  // Return type of the method
}

// BuiltinType represents a Python builtin type with its available methods.
type BuiltinType struct {
	FQN     string                    // Fully qualified name (e.g., "builtins.str")
	Methods map[string]*BuiltinMethod // Method name -> method info
}

// BuiltinRegistry maintains information about Python builtin types and their methods.
// This enables type inference for literal values and builtin method calls.
type BuiltinRegistry struct {
	Types map[string]*BuiltinType // Type FQN -> builtin type info
}

// NewBuiltinRegistry creates and initializes a registry with Python builtin types.
// The registry is pre-populated with common types: str, list, dict, set, tuple,
// int, float, bool, bytes, and their associated methods.
//
// Returns:
//   - Initialized BuiltinRegistry with all builtin types
func NewBuiltinRegistry() *BuiltinRegistry {
	registry := &BuiltinRegistry{
		Types: make(map[string]*BuiltinType),
	}

	// Initialize builtin types
	registry.initStringType()
	registry.initListType()
	registry.initDictType()
	registry.initSetType()
	registry.initTupleType()
	registry.initIntType()
	registry.initFloatType()
	registry.initBoolType()
	registry.initBytesType()

	return registry
}

// GetType retrieves builtin type information by its fully qualified name.
//
// Parameters:
//   - typeFQN: fully qualified type name (e.g., "builtins.str")
//
// Returns:
//   - BuiltinType if found, nil otherwise
func (br *BuiltinRegistry) GetType(typeFQN string) *BuiltinType {
	return br.Types[typeFQN]
}

// GetMethod retrieves method information for a builtin type.
//
// Parameters:
//   - typeFQN: fully qualified type name
//   - methodName: name of the method
//
// Returns:
//   - BuiltinMethod if found, nil otherwise
func (br *BuiltinRegistry) GetMethod(typeFQN, methodName string) *BuiltinMethod {
	builtinType := br.GetType(typeFQN)
	if builtinType == nil {
		return nil
	}
	return builtinType.Methods[methodName]
}

// InferLiteralType infers the type of a Python literal value.
// Supports: strings, integers, floats, booleans, lists, dicts, sets, tuples.
//
// Parameters:
//   - literal: the literal value as a string
//
// Returns:
//   - TypeInfo with confidence 1.0 if recognized, nil otherwise
func (br *BuiltinRegistry) InferLiteralType(literal string) *core.TypeInfo {
	literal = strings.TrimSpace(literal)

	// String literals (single/double/triple quotes)
	if (strings.HasPrefix(literal, "'") && strings.HasSuffix(literal, "'")) ||
		(strings.HasPrefix(literal, "\"") && strings.HasSuffix(literal, "\"")) ||
		(strings.HasPrefix(literal, "'''") && strings.HasSuffix(literal, "'''")) ||
		(strings.HasPrefix(literal, "\"\"\"") && strings.HasSuffix(literal, "\"\"\"")) {
		return &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	// Bytes literals
	if (strings.HasPrefix(literal, "b'") || strings.HasPrefix(literal, "b\"")) {
		return &core.TypeInfo{
			TypeFQN:    "builtins.bytes",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	// Boolean literals
	if literal == "True" || literal == "False" {
		return &core.TypeInfo{
			TypeFQN:    "builtins.bool",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	// None (NoneType)
	if literal == "None" {
		return &core.TypeInfo{
			TypeFQN:    "builtins.NoneType",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	// List literals
	if strings.HasPrefix(literal, "[") && strings.HasSuffix(literal, "]") {
		return &core.TypeInfo{
			TypeFQN:    "builtins.list",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	// Dict literals
	if strings.HasPrefix(literal, "{") && strings.HasSuffix(literal, "}") {
		// Check if it's a set (would need element analysis for certainty)
		// For now, assume dict if it contains ':' and set otherwise
		if strings.Contains(literal, ":") || literal == "{}" {
			return &core.TypeInfo{
				TypeFQN:    "builtins.dict",
				Confidence: 1.0,
				Source:     "literal",
			}
		}
		return &core.TypeInfo{
			TypeFQN:    "builtins.set",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	// Tuple literals
	if strings.HasPrefix(literal, "(") && strings.HasSuffix(literal, ")") {
		return &core.TypeInfo{
			TypeFQN:    "builtins.tuple",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	// Numeric literals (int or float)
	if isNumericLiteral(literal) {
		if strings.Contains(literal, ".") || strings.Contains(literal, "e") || strings.Contains(literal, "E") {
			return &core.TypeInfo{
				TypeFQN:    "builtins.float",
				Confidence: 1.0,
				Source:     "literal",
			}
		}
		return &core.TypeInfo{
			TypeFQN:    "builtins.int",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	return nil
}

// isNumericLiteral checks if a string represents a numeric literal.
func isNumericLiteral(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Handle negative numbers
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
	}

	if len(s) == 0 {
		return false
	}

	// Check for hex, octal, binary prefixes
	if len(s) >= 2 {
		prefix := strings.ToLower(s[:2])
		if prefix == "0x" || prefix == "0o" || prefix == "0b" {
			return len(s) > 2
		}
	}

	hasDigit := false
	hasDot := false
	hasE := false
	skipNext := false

	for i, ch := range s {
		if skipNext {
			skipNext = false
			if ch == '+' || ch == '-' {
				continue
			}
		}

		switch {
		case ch >= '0' && ch <= '9':
			hasDigit = true
		case ch == '.':
			if hasDot || hasE {
				return false
			}
			hasDot = true
		case ch == 'e' || ch == 'E':
			if hasE || !hasDigit {
				return false
			}
			hasE = true
			// Next character can be +/-
			if i+1 < len(s) && (s[i+1] == '+' || s[i+1] == '-') {
				skipNext = true
			}
		case ch == '_':
			// Python allows underscores in numeric literals (e.g., 1_000_000)
			continue
		default:
			// +/- only allowed after 'e' or 'E', which is handled by skipNext
			return false
		}
	}

	return hasDigit
}

// initStringType initializes the builtin str type and its methods.
func (br *BuiltinRegistry) initStringType() {
	strType := &BuiltinType{
		FQN:     "builtins.str",
		Methods: make(map[string]*BuiltinMethod),
	}

	// String methods that return str
	stringReturnMethods := []string{
		"capitalize", "casefold", "center", "expandtabs", "format",
		"format_map", "join", "ljust", "lower", "lstrip", "replace",
		"rjust", "rstrip", "strip", "swapcase", "title", "translate",
		"upper", "zfill",
	}
	for _, method := range stringReturnMethods {
		strType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.str", Confidence: 1.0, Source: "builtin"},
		}
	}

	// String methods that return bool
	boolReturnMethods := []string{
		"isalnum", "isalpha", "isascii", "isdecimal", "isdigit",
		"isidentifier", "islower", "isnumeric", "isprintable",
		"isspace", "istitle", "isupper", "startswith", "endswith",
	}
	for _, method := range boolReturnMethods {
		strType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.bool", Confidence: 1.0, Source: "builtin"},
		}
	}

	// String methods that return int
	intReturnMethods := []string{"count", "find", "index", "rfind", "rindex"}
	for _, method := range intReturnMethods {
		strType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "builtin"},
		}
	}

	// String methods that return list
	listReturnMethods := []string{"split", "rsplit", "splitlines", "partition", "rpartition"}
	for _, method := range listReturnMethods {
		strType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.list", Confidence: 1.0, Source: "builtin"},
		}
	}

	// encode returns bytes
	strType.Methods["encode"] = &BuiltinMethod{
		Name:       "encode",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.bytes", Confidence: 1.0, Source: "builtin"},
	}

	br.Types["builtins.str"] = strType
}

// initListType initializes the builtin list type and its methods.
func (br *BuiltinRegistry) initListType() {
	listType := &BuiltinType{
		FQN:     "builtins.list",
		Methods: make(map[string]*BuiltinMethod),
	}

	// Methods that return None (mutating methods)
	noneMethods := []string{"append", "extend", "insert", "remove", "clear", "sort", "reverse"}
	for _, method := range noneMethods {
		listType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.NoneType", Confidence: 1.0, Source: "builtin"},
		}
	}

	// Methods that return int
	listType.Methods["count"] = &BuiltinMethod{
		Name:       "count",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "builtin"},
	}
	listType.Methods["index"] = &BuiltinMethod{
		Name:       "index",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "builtin"},
	}

	// copy returns list
	listType.Methods["copy"] = &BuiltinMethod{
		Name:       "copy",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.list", Confidence: 1.0, Source: "builtin"},
	}

	// pop returns the element (unknown type, use confidence 0.0)
	listType.Methods["pop"] = &BuiltinMethod{
		Name:       "pop",
		ReturnType: &core.TypeInfo{TypeFQN: "", Confidence: 0.0, Source: "builtin"},
	}

	br.Types["builtins.list"] = listType
}

// initDictType initializes the builtin dict type and its methods.
func (br *BuiltinRegistry) initDictType() {
	dictType := &BuiltinType{
		FQN:     "builtins.dict",
		Methods: make(map[string]*BuiltinMethod),
	}

	// Methods that return None
	noneMethods := []string{"clear", "update"}
	for _, method := range noneMethods {
		dictType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.NoneType", Confidence: 1.0, Source: "builtin"},
		}
	}

	// Methods that return dict views/iterables
	dictType.Methods["keys"] = &BuiltinMethod{
		Name:       "keys",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.dict_keys", Confidence: 1.0, Source: "builtin"},
	}
	dictType.Methods["values"] = &BuiltinMethod{
		Name:       "values",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.dict_values", Confidence: 1.0, Source: "builtin"},
	}
	dictType.Methods["items"] = &BuiltinMethod{
		Name:       "items",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.dict_items", Confidence: 1.0, Source: "builtin"},
	}

	// copy returns dict
	dictType.Methods["copy"] = &BuiltinMethod{
		Name:       "copy",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.dict", Confidence: 1.0, Source: "builtin"},
	}

	// get, pop, popitem, setdefault return unknown types
	dictType.Methods["get"] = &BuiltinMethod{
		Name:       "get",
		ReturnType: &core.TypeInfo{TypeFQN: "", Confidence: 0.0, Source: "builtin"},
	}
	dictType.Methods["pop"] = &BuiltinMethod{
		Name:       "pop",
		ReturnType: &core.TypeInfo{TypeFQN: "", Confidence: 0.0, Source: "builtin"},
	}
	dictType.Methods["popitem"] = &BuiltinMethod{
		Name:       "popitem",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.tuple", Confidence: 0.5, Source: "builtin"},
	}
	dictType.Methods["setdefault"] = &BuiltinMethod{
		Name:       "setdefault",
		ReturnType: &core.TypeInfo{TypeFQN: "", Confidence: 0.0, Source: "builtin"},
	}

	br.Types["builtins.dict"] = dictType
}

// initSetType initializes the builtin set type and its methods.
func (br *BuiltinRegistry) initSetType() {
	setType := &BuiltinType{
		FQN:     "builtins.set",
		Methods: make(map[string]*BuiltinMethod),
	}

	// Methods that return None
	noneMethods := []string{"add", "remove", "discard", "clear", "update",
		"intersection_update", "difference_update", "symmetric_difference_update"}
	for _, method := range noneMethods {
		setType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.NoneType", Confidence: 1.0, Source: "builtin"},
		}
	}

	// Methods that return set
	setReturnMethods := []string{"copy", "union", "intersection", "difference", "symmetric_difference"}
	for _, method := range setReturnMethods {
		setType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.set", Confidence: 1.0, Source: "builtin"},
		}
	}

	// Methods that return bool
	boolReturnMethods := []string{"isdisjoint", "issubset", "issuperset"}
	for _, method := range boolReturnMethods {
		setType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.bool", Confidence: 1.0, Source: "builtin"},
		}
	}

	// pop returns unknown element type
	setType.Methods["pop"] = &BuiltinMethod{
		Name:       "pop",
		ReturnType: &core.TypeInfo{TypeFQN: "", Confidence: 0.0, Source: "builtin"},
	}

	br.Types["builtins.set"] = setType
}

// initTupleType initializes the builtin tuple type and its methods.
func (br *BuiltinRegistry) initTupleType() {
	tupleType := &BuiltinType{
		FQN:     "builtins.tuple",
		Methods: make(map[string]*BuiltinMethod),
	}

	// Tuple methods that return int
	tupleType.Methods["count"] = &BuiltinMethod{
		Name:       "count",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "builtin"},
	}
	tupleType.Methods["index"] = &BuiltinMethod{
		Name:       "index",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "builtin"},
	}

	br.Types["builtins.tuple"] = tupleType
}

// initIntType initializes the builtin int type and its methods.
func (br *BuiltinRegistry) initIntType() {
	intType := &BuiltinType{
		FQN:     "builtins.int",
		Methods: make(map[string]*BuiltinMethod),
	}

	// Int methods that return int
	intReturnMethods := []string{"bit_length", "bit_count", "conjugate"}
	for _, method := range intReturnMethods {
		intType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "builtin"},
		}
	}

	// to_bytes returns bytes
	intType.Methods["to_bytes"] = &BuiltinMethod{
		Name:       "to_bytes",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.bytes", Confidence: 1.0, Source: "builtin"},
	}

	// from_bytes is a class method that returns int
	intType.Methods["from_bytes"] = &BuiltinMethod{
		Name:       "from_bytes",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "builtin"},
	}

	br.Types["builtins.int"] = intType
}

// initFloatType initializes the builtin float type and its methods.
func (br *BuiltinRegistry) initFloatType() {
	floatType := &BuiltinType{
		FQN:     "builtins.float",
		Methods: make(map[string]*BuiltinMethod),
	}

	// Float methods that return float
	floatReturnMethods := []string{"conjugate"}
	for _, method := range floatReturnMethods {
		floatType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.float", Confidence: 1.0, Source: "builtin"},
		}
	}

	// Methods that return bool
	boolReturnMethods := []string{"is_integer"}
	for _, method := range boolReturnMethods {
		floatType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.bool", Confidence: 1.0, Source: "builtin"},
		}
	}

	// hex returns str
	floatType.Methods["hex"] = &BuiltinMethod{
		Name:       "hex",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.str", Confidence: 1.0, Source: "builtin"},
	}

	// fromhex is a class method that returns float
	floatType.Methods["fromhex"] = &BuiltinMethod{
		Name:       "fromhex",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.float", Confidence: 1.0, Source: "builtin"},
	}

	br.Types["builtins.float"] = floatType
}

// initBoolType initializes the builtin bool type.
func (br *BuiltinRegistry) initBoolType() {
	boolType := &BuiltinType{
		FQN:     "builtins.bool",
		Methods: make(map[string]*BuiltinMethod),
	}
	// Bool has no unique methods (inherits from int)
	br.Types["builtins.bool"] = boolType
}

// initBytesType initializes the builtin bytes type and its methods.
func (br *BuiltinRegistry) initBytesType() {
	bytesType := &BuiltinType{
		FQN:     "builtins.bytes",
		Methods: make(map[string]*BuiltinMethod),
	}

	// Bytes methods that return bytes
	bytesReturnMethods := []string{
		"capitalize", "center", "expandtabs", "join", "ljust",
		"lower", "lstrip", "replace", "rjust", "rstrip", "strip",
		"swapcase", "title", "translate", "upper", "zfill",
	}
	for _, method := range bytesReturnMethods {
		bytesType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.bytes", Confidence: 1.0, Source: "builtin"},
		}
	}

	// Bytes methods that return bool
	boolReturnMethods := []string{
		"isalnum", "isalpha", "isascii", "isdigit", "islower",
		"isspace", "istitle", "isupper", "startswith", "endswith",
	}
	for _, method := range boolReturnMethods {
		bytesType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.bool", Confidence: 1.0, Source: "builtin"},
		}
	}

	// Bytes methods that return int
	intReturnMethods := []string{"count", "find", "index", "rfind", "rindex"}
	for _, method := range intReturnMethods {
		bytesType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "builtin"},
		}
	}

	// Bytes methods that return list
	listReturnMethods := []string{"split", "rsplit", "splitlines", "partition", "rpartition"}
	for _, method := range listReturnMethods {
		bytesType.Methods[method] = &BuiltinMethod{
			Name:       method,
			ReturnType: &core.TypeInfo{TypeFQN: "builtins.list", Confidence: 1.0, Source: "builtin"},
		}
	}

	// decode returns str
	bytesType.Methods["decode"] = &BuiltinMethod{
		Name:       "decode",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.str", Confidence: 1.0, Source: "builtin"},
	}

	// hex returns str
	bytesType.Methods["hex"] = &BuiltinMethod{
		Name:       "hex",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.str", Confidence: 1.0, Source: "builtin"},
	}

	// fromhex is a class method that returns bytes
	bytesType.Methods["fromhex"] = &BuiltinMethod{
		Name:       "fromhex",
		ReturnType: &core.TypeInfo{TypeFQN: "builtins.bytes", Confidence: 1.0, Source: "builtin"},
	}

	br.Types["builtins.bytes"] = bytesType
}

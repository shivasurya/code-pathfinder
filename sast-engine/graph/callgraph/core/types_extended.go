// Package core provides extended type system for bidirectional inference.
package core

import (
	"fmt"
	"slices"
	"strings"
)

// =============================================================================
// TYPE INTERFACE HIERARCHY
// =============================================================================

// Type represents any inferred type in the system.
// All concrete types must implement this interface.
type Type interface {
	isType()                // Marker method for type safety
	String() string         // Human-readable representation
	FQN() string            // Fully qualified name (e.g., "myapp.models.User")
	Equals(other Type) bool // Structural equality check
	Confidence() float64    // Confidence score 0.0-1.0
}

// ConcreteType represents a known, resolved type.
// Example: "myapp.models.User", "builtins.str".
type ConcreteType struct {
	Name       string  // Short name: "User"
	Module     string  // Module path: "myapp.models"
	confidence float64 // Inference confidence
}

func (t *ConcreteType) isType() {}

func (t *ConcreteType) String() string {
	if t.Module == "" {
		return t.Name
	}
	return fmt.Sprintf("%s.%s", t.Module, t.Name)
}

func (t *ConcreteType) FQN() string {
	return t.String()
}

func (t *ConcreteType) Equals(other Type) bool {
	if o, ok := other.(*ConcreteType); ok {
		return t.Module == o.Module && t.Name == o.Name
	}
	return false
}

func (t *ConcreteType) Confidence() float64 {
	return t.confidence
}

// NewConcreteType creates a ConcreteType from a fully qualified name.
func NewConcreteType(fqn string, confidence float64) *ConcreteType {
	parts := strings.Split(fqn, ".")
	if len(parts) == 1 {
		return &ConcreteType{Name: parts[0], Module: "", confidence: confidence}
	}
	name := parts[len(parts)-1]
	module := strings.Join(parts[:len(parts)-1], ".")
	return &ConcreteType{Name: name, Module: module, confidence: confidence}
}

// TypeVariable represents an unresolved type placeholder.
// Used during constraint solving before concrete resolution.
type TypeVariable struct {
	ID         int    // Unique identifier
	Name       string // Optional name hint (e.g., "T" for generics)
	Constraint Type   // Upper bound constraint, if any
}

func (t *TypeVariable) isType() {}

func (t *TypeVariable) String() string {
	if t.Name != "" {
		return fmt.Sprintf("$%s_%d", t.Name, t.ID)
	}
	return fmt.Sprintf("$T_%d", t.ID)
}

func (t *TypeVariable) FQN() string {
	return t.String()
}

func (t *TypeVariable) Equals(other Type) bool {
	if o, ok := other.(*TypeVariable); ok {
		return t.ID == o.ID
	}
	return false
}

func (t *TypeVariable) Confidence() float64 {
	return 0.0 // Unresolved = no confidence
}

// UnionType represents a type that could be one of several types.
// Example: Union[str, None] for Optional[str].
type UnionType struct {
	Types      []Type // Member types
	confidence float64
}

func (t *UnionType) isType() {}

func (t *UnionType) String() string {
	parts := make([]string, 0, len(t.Types))
	for _, typ := range t.Types {
		parts = append(parts, typ.String())
	}
	return fmt.Sprintf("Union[%s]", strings.Join(parts, ", "))
}

func (t *UnionType) FQN() string {
	return t.String()
}

func (t *UnionType) Equals(other Type) bool {
	if o, ok := other.(*UnionType); ok {
		if len(t.Types) != len(o.Types) {
			return false
		}
		// Order-independent comparison
		for _, tt := range t.Types {
			found := slices.ContainsFunc(o.Types, tt.Equals)
			if !found {
				return false
			}
		}
		return true
	}
	return false
}

func (t *UnionType) Confidence() float64 {
	return t.confidence
}

// NewUnionType creates a UnionType from member types.
// Automatically flattens nested unions and removes duplicates.
func NewUnionType(types []Type, confidence float64) *UnionType {
	var flattened []Type
	seen := make(map[string]bool)

	for _, typ := range types {
		if union, ok := typ.(*UnionType); ok {
			// Flatten nested union
			for _, inner := range union.Types {
				fqn := inner.FQN()
				if !seen[fqn] {
					seen[fqn] = true
					flattened = append(flattened, inner)
				}
			}
		} else {
			fqn := typ.FQN()
			if !seen[fqn] {
				seen[fqn] = true
				flattened = append(flattened, typ)
			}
		}
	}

	return &UnionType{Types: flattened, confidence: confidence}
}

// AnyType represents an unknown or unresolvable type.
// Used when inference fails or for dynamic patterns.
type AnyType struct {
	Reason string // Why inference failed
}

func (t *AnyType) isType() {}

func (t *AnyType) String() string {
	if t.Reason != "" {
		return fmt.Sprintf("Any<%s>", t.Reason)
	}
	return "Any"
}

func (t *AnyType) FQN() string {
	return "typing.Any"
}

func (t *AnyType) Equals(other Type) bool {
	_, ok := other.(*AnyType)
	return ok
}

func (t *AnyType) Confidence() float64 {
	return 0.0
}

// NoneType represents Python's None.
type NoneType struct{}

func (t *NoneType) isType() {}

func (t *NoneType) String() string {
	return "None"
}

func (t *NoneType) FQN() string {
	return "builtins.NoneType"
}

func (t *NoneType) Equals(other Type) bool {
	_, ok := other.(*NoneType)
	return ok
}

func (t *NoneType) Confidence() float64 {
	return 1.0
}

// FunctionType represents a callable type with signature.
type FunctionType struct {
	Parameters []Type // Parameter types
	ReturnType Type   // Return type
	confidence float64
}

func (t *FunctionType) isType() {}

func (t *FunctionType) String() string {
	params := make([]string, 0, len(t.Parameters))
	for _, p := range t.Parameters {
		params = append(params, p.String())
	}
	ret := "None"
	if t.ReturnType != nil {
		ret = t.ReturnType.String()
	}
	return fmt.Sprintf("Callable[[%s], %s]", strings.Join(params, ", "), ret)
}

func (t *FunctionType) FQN() string {
	return t.String()
}

func (t *FunctionType) Equals(other Type) bool {
	if o, ok := other.(*FunctionType); ok {
		if len(t.Parameters) != len(o.Parameters) {
			return false
		}
		for i, p := range t.Parameters {
			if !p.Equals(o.Parameters[i]) {
				return false
			}
		}
		if t.ReturnType == nil && o.ReturnType == nil {
			return true
		}
		if t.ReturnType == nil || o.ReturnType == nil {
			return false
		}
		return t.ReturnType.Equals(o.ReturnType)
	}
	return false
}

func (t *FunctionType) Confidence() float64 {
	return t.confidence
}

// =============================================================================
// CONFIDENCE SCORING
// =============================================================================

// ConfidenceSource represents where a type inference came from.
type ConfidenceSource string

const (
	ConfidenceAnnotation      ConfidenceSource = "annotation"  // Type annotation: 1.0
	ConfidenceLiteral         ConfidenceSource = "literal"     // Literal value: 0.95
	ConfidenceConstructor     ConfidenceSource = "constructor" // Constructor call: 0.95
	ConfidenceReturnType      ConfidenceSource = "return_type" // Function return: 0.9
	ConfidenceAssignment      ConfidenceSource = "assignment"  // Assignment tracking: 0.85
	ConfidenceAttribute       ConfidenceSource = "attribute"   // Attribute access: 0.8
	ConfidenceFluentHeuristic ConfidenceSource = "fluent"      // Fluent interface guess: 0.7
	ConfidenceUnknown         ConfidenceSource = "unknown"     // Unknown: 0.0
)

// ConfidenceScore returns the numeric confidence for a source.
func ConfidenceScore(source ConfidenceSource) float64 {
	switch source {
	case ConfidenceAnnotation:
		return 1.0
	case ConfidenceLiteral:
		return 0.95
	case ConfidenceConstructor:
		return 0.95
	case ConfidenceReturnType:
		return 0.9
	case ConfidenceAssignment:
		return 0.85
	case ConfidenceAttribute:
		return 0.8
	case ConfidenceFluentHeuristic:
		return 0.7
	default:
		return 0.0
	}
}

// CombineConfidence combines multiple confidence scores.
// Uses multiplication for sequential inferences (a.b.c).
func CombineConfidence(scores ...float64) float64 {
	if len(scores) == 0 {
		return 0.0
	}
	result := scores[0]
	for _, s := range scores[1:] {
		result *= s
	}
	return result
}

// =============================================================================
// TYPE UTILITIES
// =============================================================================

// IsNoneType checks if a type is None.
func IsNoneType(t Type) bool {
	_, ok := t.(*NoneType)
	return ok
}

// IsAnyType checks if a type is Any (unresolved).
func IsAnyType(t Type) bool {
	_, ok := t.(*AnyType)
	return ok
}

// IsConcreteType checks if a type is a resolved concrete type.
func IsConcreteType(t Type) bool {
	_, ok := t.(*ConcreteType)
	return ok
}

// ExtractConcreteType extracts the ConcreteType if applicable.
func ExtractConcreteType(t Type) (*ConcreteType, bool) {
	ct, ok := t.(*ConcreteType)
	return ct, ok
}

// SimplifyUnion simplifies a union type.
// - Single type: returns that type
// - All same type: returns that type
// - Contains Any: returns Any.
func SimplifyUnion(u *UnionType) Type {
	if len(u.Types) == 0 {
		return &AnyType{Reason: "empty union"}
	}
	if len(u.Types) == 1 {
		return u.Types[0]
	}

	// Check if all types are the same
	first := u.Types[0]
	allSame := true
	for _, t := range u.Types[1:] {
		if !first.Equals(t) {
			allSame = false
			break
		}
	}
	if allSame {
		return first
	}

	// Check for Any
	if slices.ContainsFunc(u.Types, IsAnyType) {
		return &AnyType{Reason: "union contains Any"}
	}

	return u
}

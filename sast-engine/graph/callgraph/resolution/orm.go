package resolution

import (
	"slices"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// ORMPattern represents a recognized ORM pattern (e.g., Django ORM, SQLAlchemy).
// These patterns are dynamically generated at runtime and won't be found in source code,
// but we can still resolve them by recognizing the pattern.
type ORMPattern struct {
	Name        string   // Pattern name (e.g., "Django ORM")
	MethodNames []string // Common ORM method names
	Description string   // Human-readable description
}

// Common Django ORM method names that are dynamically generated.
var djangoORMMethods = []string{
	"filter", "get", "create", "update", "delete",
	"all", "first", "last", "count", "exists",
	"exclude", "annotate", "aggregate", "values",
	"values_list", "select_related", "prefetch_related",
	"only", "defer", "using", "order_by", "reverse",
	"distinct", "latest", "earliest", "bulk_create",
	"bulk_update", "in_bulk", "iterator", "update_or_create",
	"get_or_create", "none",
}

// Common SQLAlchemy ORM method names.
var sqlalchemyORMMethods = []string{
	"filter", "filter_by", "get", "first", "all",
	"one", "one_or_none", "scalar", "count",
	"order_by", "limit", "offset", "join",
	"outerjoin", "group_by", "having", "distinct",
}

// IsDjangoORMPattern checks if a call target matches Django ORM pattern.
// Django ORM pattern: ModelName.objects.<method>
//
// Examples:
//   - "Task.objects.filter" → true
//   - "User.objects.get" → true
//   - "Annotation.objects.all" → true
//   - "task.save" → false (instance method, not manager)
//
// Parameters:
//   - target: call target string (e.g., "Task.objects.filter")
//
// Returns:
//   - true if it matches Django ORM pattern
//   - the method name if matched (e.g., "filter")
func IsDjangoORMPattern(target string) (bool, string) {
	// Must contain ".objects."
	if !strings.Contains(target, ".objects.") {
		// Also check for ".objects" suffix (e.g., "Task.objects")
		if !strings.HasSuffix(target, ".objects") {
			return false, ""
		}
	}

	// Extract method name after .objects.
	// Example: "Task.objects.filter" → "filter"
	parts := strings.Split(target, ".objects.")
	if len(parts) == 2 {
		methodName := parts[1]
		// Check if it's a known Django ORM method
		if slices.Contains(djangoORMMethods, methodName) {
			return true, methodName
		}
		// Even if not in our list, if it follows the pattern, mark it as ORM
		// This handles custom managers and less common methods
		return true, methodName
	}

	// Handle "Task.objects" without a method (returns the manager itself)
	if strings.HasSuffix(target, ".objects") {
		return true, "objects"
	}

	return false, ""
}

// IsSQLAlchemyORMPattern checks if a call target matches SQLAlchemy ORM pattern.
// SQLAlchemy patterns are more varied, but common ones include:
//   - session.query(Model).filter(...)
//   - db.session.query(Model).all()
//   - Model.query.filter_by(...)
//
// Parameters:
//   - target: call target string
//
// Returns:
//   - true if it matches SQLAlchemy ORM pattern
//   - the method name if matched
func IsSQLAlchemyORMPattern(target string) (bool, string) {
	// Check for .query. pattern (Flask-SQLAlchemy style)
	if strings.Contains(target, ".query.") {
		parts := strings.Split(target, ".query.")
		if len(parts) == 2 {
			methodName := parts[1]
			if slices.Contains(sqlalchemyORMMethods, methodName) {
				return true, methodName
			}
			return true, methodName
		}
	}

	return false, ""
}

// IsORMPattern checks if a call target matches any known ORM pattern.
//
// Parameters:
//   - target: call target string
//
// Returns:
//   - true if it matches any ORM pattern
//   - the ORM pattern name (e.g., "Django ORM")
//   - the method name (e.g., "filter")
func IsORMPattern(target string) (bool, string, string) {
	// Check Django ORM
	if isDjango, method := IsDjangoORMPattern(target); isDjango {
		return true, "Django ORM", method
	}

	// Check SQLAlchemy ORM
	if isSQLAlchemy, method := IsSQLAlchemyORMPattern(target); isSQLAlchemy {
		return true, "SQLAlchemy", method
	}

	return false, "", ""
}

// ValidateDjangoModel checks if a name is likely a Django model by examining
// the code graph for the class definition and checking if it inherits from
// django.db.models.Model or has "Model" in its name.
//
// This is a heuristic check since we can't always definitively determine
// if something is a Django model without runtime information.
//
// Parameters:
//   - modelName: the name to check (e.g., "Task", "User")
//   - codeGraph: the parsed code graph
//
// Returns:
//   - true if the name is likely a Django model
func ValidateDjangoModel(modelName string, codeGraph *graph.CodeGraph) bool {
	// Look for a class with this name in the code graph
	for _, node := range codeGraph.Nodes {
		if node.Type == "class_declaration" && node.Name == modelName {
			// Check if it has "Model" in superclass
			// Note: This is a heuristic - ideally we'd check the full inheritance chain
			if strings.Contains(node.SuperClass, "Model") {
				return true
			}
			// If it has "Model" suffix in name, likely a model
			if strings.HasSuffix(modelName, "Model") {
				return true
			}
		}
	}

	// Common Django model names pattern
	// Most Django models don't have "Model" suffix but are PascalCase nouns
	// This is a weak heuristic but better than nothing
	if len(modelName) > 0 {
		firstChar := modelName[0]
		// PascalCase name (starts with uppercase)
		if firstChar >= 'A' && firstChar <= 'Z' {
			// Not a known common non-model class pattern
			if !strings.HasPrefix(modelName, "Test") &&
				!strings.HasSuffix(modelName, "View") &&
				!strings.HasSuffix(modelName, "Serializer") &&
				!strings.HasSuffix(modelName, "Form") {
				// Likely a model
				return true
			}
		}
	}

	return false
}

// ResolveDjangoORMCall attempts to resolve a Django ORM call pattern.
// It constructs a synthetic FQN for the ORM method even though it doesn't
// exist in source code, because Django generates these methods at runtime.
//
// Parameters:
//   - target: the call target (e.g., "Task.objects.filter")
//   - modulePath: the current module path
//   - registry: module registry
//   - codeGraph: the parsed code graph (for model validation)
//
// Returns:
//   - fully qualified name for the ORM call
//   - true if successfully resolved as Django ORM
func ResolveDjangoORMCall(target string, modulePath string, registry *core.ModuleRegistry, codeGraph *graph.CodeGraph) (string, bool) {
	isDjango, method := IsDjangoORMPattern(target)
	if !isDjango {
		return target, false
	}

	// Extract model name from pattern
	// "Task.objects.filter" → "Task"
	parts := strings.Split(target, ".objects")
	if len(parts) == 0 {
		return target, false
	}

	modelName := parts[0]
	// Handle qualified names like "models.Task.objects.filter"
	if strings.Contains(modelName, ".") {
		modelParts := strings.Split(modelName, ".")
		modelName = modelParts[len(modelParts)-1]
	}

	// Validate if it's likely a Django model (optional heuristic)
	// Even if validation fails, we still resolve it as ORM pattern
	// because the pattern match is strong enough
	isModel := ValidateDjangoModel(modelName, codeGraph)

	// Construct synthetic FQN for the ORM method
	// Format: module.ModelName.objects.<method>
	// This doesn't exist in source but represents the runtime behavior
	fqn := modulePath + "." + modelName + ".objects." + method

	// If we validated it as a model, we're more confident
	// But we resolve it either way since the pattern is clear
	_ = isModel // Used for potential future confidence scoring

	return fqn, true
}

// ResolveSQLAlchemyORMCall attempts to resolve a SQLAlchemy ORM call pattern.
//
// Parameters:
//   - target: the call target
//   - modulePath: the current module path
//
// Returns:
//   - fully qualified name for the ORM call
//   - true if successfully resolved as SQLAlchemy ORM
func ResolveSQLAlchemyORMCall(target string, modulePath string) (string, bool) {
	isSQLAlchemy, _ := IsSQLAlchemyORMPattern(target)
	if !isSQLAlchemy {
		return target, false
	}

	// Construct synthetic FQN
	fqn := modulePath + "." + target

	return fqn, true
}

// ResolveORMCall attempts to resolve any ORM pattern.
//
// Parameters:
//   - target: the call target
//   - modulePath: the current module path
//   - registry: module registry
//   - codeGraph: the parsed code graph
//
// Returns:
//   - fully qualified name for the ORM call
//   - true if successfully resolved as any ORM pattern
func ResolveORMCall(target string, modulePath string, registry *core.ModuleRegistry, codeGraph *graph.CodeGraph) (string, bool) {
	// Try Django ORM first (most common)
	if fqn, resolved := ResolveDjangoORMCall(target, modulePath, registry, codeGraph); resolved {
		return fqn, true
	}

	// Try SQLAlchemy
	if fqn, resolved := ResolveSQLAlchemyORMCall(target, modulePath); resolved {
		return fqn, true
	}

	return target, false
}

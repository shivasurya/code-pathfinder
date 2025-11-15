package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
)

// IsDjangoORMPattern checks if a call target matches Django ORM pattern.
// Deprecated: Use resolution.IsDjangoORMPattern instead.
func IsDjangoORMPattern(target string) (bool, string) {
	return resolution.IsDjangoORMPattern(target)
}

// IsSQLAlchemyORMPattern checks if a call target matches SQLAlchemy ORM pattern.
// Deprecated: Use resolution.IsSQLAlchemyORMPattern instead.
func IsSQLAlchemyORMPattern(target string) (bool, string) {
	return resolution.IsSQLAlchemyORMPattern(target)
}

// IsORMPattern detects if target is any recognized ORM pattern.
// Deprecated: Use resolution.IsORMPattern instead.
func IsORMPattern(target string) (bool, string, string) {
	return resolution.IsORMPattern(target)
}

// ValidateDjangoModel validates that a Django model exists in the code graph.
// Deprecated: Use resolution.ValidateDjangoModel instead.
func ValidateDjangoModel(modelName string, codeGraph *graph.CodeGraph) bool {
	return resolution.ValidateDjangoModel(modelName, codeGraph)
}

// ResolveDjangoORMCall resolves Django ORM call to a synthetic FQN.
// Deprecated: Use resolution.ResolveDjangoORMCall instead.
func ResolveDjangoORMCall(target string, modulePath string, registry *core.ModuleRegistry, codeGraph *graph.CodeGraph) (string, bool) {
	return resolution.ResolveDjangoORMCall(target, modulePath, registry, codeGraph)
}

// ResolveSQLAlchemyORMCall resolves SQLAlchemy ORM call to a synthetic FQN.
// Deprecated: Use resolution.ResolveSQLAlchemyORMCall instead.
func ResolveSQLAlchemyORMCall(target string, modulePath string) (string, bool) {
	return resolution.ResolveSQLAlchemyORMCall(target, modulePath)
}

// ResolveORMCall detects and resolves ORM calls.
// Deprecated: Use resolution.ResolveORMCall instead.
func ResolveORMCall(target string, modulePath string, registry *core.ModuleRegistry, codeGraph *graph.CodeGraph) (string, bool) {
	return resolution.ResolveORMCall(target, modulePath, registry, codeGraph)
}

// Package resolution provides type information structures for type resolution and inference.
//
// This package defines the type system used by the type inference engine
// and registry packages. It contains data structures that track variable bindings
// and function scopes during type analysis.
//
// # Type Information
//
// The core type information is defined in the core package (core.TypeInfo), while
// this package focuses on scope and binding management:
//
//	typeInfo := &core.TypeInfo{
//	    TypeFQN:    "builtins.str",
//	    Source:     "literal",
//	    Confidence: 1.0,
//	}
//
//	binding := &resolution.VariableBinding{
//	    VarName: "username",
//	    Type:    typeInfo,
//	}
//
// # Function Scopes
//
// FunctionScope tracks variable bindings within a function:
//
//	scope := resolution.NewFunctionScope("myapp.views.login")
//	scope.AddVariable(&resolution.VariableBinding{
//	    VarName: "user",
//	    Type:    &core.TypeInfo{TypeFQN: "myapp.models.User"},
//	})
//
// # Breaking Circular Dependencies
//
// This package was created to resolve the circular dependency between
// builtin_registry.go and type_inference.go by providing shared type definitions
// that both packages can depend on without depending on each other.
package resolution

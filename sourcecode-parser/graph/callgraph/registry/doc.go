// Package registry provides module, type, and attribute registry functionality for Python code analysis.
//
// This package handles:
//   - Module discovery and path resolution
//   - Python builtin type registry
//   - Class attribute tracking
//   - Python version detection
//
// # Module Registry
//
// BuildModuleRegistry walks a directory tree to discover all Python files
// and build a mapping from module paths to file paths:
//
//	registry, err := registry.BuildModuleRegistry("/path/to/project")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	filePath, ok := registry.GetFilePath("myapp.views")
//
// The registry automatically skips common directories like venv, __pycache__, .git, etc.
//
// # Builtin Registry
//
// BuiltinRegistry provides type information for Python builtin types and functions:
//
//	builtins := registry.NewBuiltinRegistry()
//	typeInfo := builtins.GetBuiltinType("str")
//	// Returns: &core.TypeInfo{TypeFQN: "builtins.str", ...}
//
// It includes comprehensive coverage of:
//   - Builtin types (str, int, list, dict, etc.)
//   - Builtin functions (len, range, enumerate, etc.)
//   - Type methods (str.upper, list.append, etc.)
//
// # Attribute Registry
//
// AttributeRegistry tracks class attributes discovered during analysis:
//
//	attrReg := registry.NewAttributeRegistry()
//	attrReg.AddAttribute("myapp.User", &core.ClassAttribute{
//	    Name: "email",
//	    Type: &core.TypeInfo{TypeFQN: "builtins.str"},
//	})
//
// Thread-safe for concurrent access during multi-file analysis.
//
// # Python Version Detection
//
// The package can detect Python version from project files:
//   - .python-version files
//   - pyproject.toml dependencies
//   - Defaults to latest stable version
//
// This informs which builtin types and methods are available.
package registry

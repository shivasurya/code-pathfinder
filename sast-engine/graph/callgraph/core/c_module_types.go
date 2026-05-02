package core

// CModuleRegistry indexes C source files for call-graph construction.
//
// C has no module system: a translation unit's identity IS its file path.
// The registry therefore turns the project-relative file path into the
// "module prefix" used to compose fully-qualified names (FQNs), and
// records every function definition under that prefix so the call-graph
// builder can resolve cross-file references.
//
// Lifecycle: the registry is built once after parsing (see
// `registry.BuildCModuleRegistry`) and consumed read-only by the call-graph
// builder. It is not safe for concurrent mutation; callers should treat it
// as immutable after construction.
//
// FQN format:
//
//	"<relative-path>::<function-name>"
//
//	"src/net/socket.c::connect_to_server"
//	"include/buffer.h::create_buffer"
type CModuleRegistry struct {
	// FileToPrefix maps absolute file path to its project-relative prefix.
	// The prefix is the path used in every FQN derived from the file:
	//
	//	"/home/dev/proj/src/net/socket.c" -> "src/net/socket.c"
	//
	// Files outside the project root are intentionally absent (the build
	// step skips them) so consumers can rely on prefixes being relative.
	FileToPrefix map[string]string

	// Includes maps a project-relative file path to the relative paths of
	// the headers it includes via `#include "..."`. System includes
	// (`#include <...>`) are excluded — they have no project-local file.
	//
	//	"src/net/socket.c" -> ["include/buffer.h", "include/net.h"]
	Includes map[string][]string

	// FunctionIndex maps a bare function name to every FQN that defines
	// that name. A name may resolve to multiple FQNs when it is declared
	// in a header and defined in a .c file, or when distinct translation
	// units declare static helpers with the same name.
	//
	//	"create_buffer" -> [
	//	    "src/utils/buffer.c::create_buffer",
	//	    "include/buffer.h::create_buffer",
	//	]
	FunctionIndex map[string][]string

	// ProjectRoot is the absolute path used as the base for relative-path
	// computation. Stored so consumers can re-derive prefixes for ad-hoc
	// files (e.g. an include resolved at query time).
	ProjectRoot string
}

// NewCModuleRegistry returns an empty CModuleRegistry rooted at projectRoot.
// All maps are pre-allocated so callers may write directly without nil
// checks.
func NewCModuleRegistry(projectRoot string) *CModuleRegistry {
	return &CModuleRegistry{
		FileToPrefix:  make(map[string]string),
		Includes:      make(map[string][]string),
		FunctionIndex: make(map[string][]string),
		ProjectRoot:   projectRoot,
	}
}

// CppModuleRegistry extends CModuleRegistry with namespace and class
// indices required for C++ resolution. C++ FQNs include the namespace
// path and, for methods, the enclosing class:
//
//	"src/net/socket.cpp::mylib::Socket::connect"  // namespace + class + method
//	"src/main.cpp::main"                           // free function, no namespace
//	"src/app.cpp::App::run"                        // class method, no namespace
//
// CppModuleRegistry embeds CModuleRegistry so all C-level lookups
// (FileToPrefix, Includes, FunctionIndex) work uniformly across both
// languages.
type CppModuleRegistry struct {
	// CModuleRegistry provides the file-to-prefix, include, and function
	// indices shared with C. Free functions appear in FunctionIndex; the
	// namespace- and class-qualified forms below complement (not replace)
	// it.
	CModuleRegistry

	// NamespaceIndex maps a namespace-qualified key to its single
	// canonical FQN. Keys take one of three forms:
	//
	//	"mylib::process"            // namespaced free function
	//	"Socket::connect"           // class method, no namespace
	//	"mylib::Socket::connect"    // namespaced class method
	//
	// The map deliberately holds one FQN per key (the most recent
	// definition wins) — overload resolution happens later in the call
	// graph builder once parameter types are available.
	NamespaceIndex map[string]string

	// ClassIndex maps a bare class name to every FQN that declares the
	// class. Multiple FQNs are expected for forward declarations or for
	// classes declared in distinct namespaces sharing a name:
	//
	//	"Socket" -> [
	//	    "src/net/socket.cpp::mylib::Socket",
	//	    "include/socket.hpp::mylib::Socket",
	//	]
	ClassIndex map[string][]string
}

// NewCppModuleRegistry returns an empty CppModuleRegistry rooted at
// projectRoot. The embedded CModuleRegistry is initialised with the same
// root, so all maps are non-nil.
func NewCppModuleRegistry(projectRoot string) *CppModuleRegistry {
	return &CppModuleRegistry{
		CModuleRegistry: *NewCModuleRegistry(projectRoot),
		NamespaceIndex:  make(map[string]string),
		ClassIndex:      make(map[string][]string),
	}
}

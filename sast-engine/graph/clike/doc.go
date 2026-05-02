// Package clike contains shared helpers for parsing C and C++ source files.
//
// The parsing pipeline treats C and C++ as two distinct languages — separate
// tree-sitter grammars, separate Node.Language values ("c" and "cpp"),
// separate dispatchers in graph/parser.go — but the AST node structure for
// declarations, statements, and types is largely shared between the two
// grammars. Rather than duplicate the extraction logic in two parallel
// dispatchers, the cross-cutting primitives live here.
//
// # Detection (this PR)
//
//   - IsCSourceFile / IsCppSourceFile route a file to the correct grammar
//   - DetectCppInHeader is a best-effort heuristic for the .h ambiguity
//     (.h is shared between C and C++; the worker calls this once per file
//     and CacheHeaderLanguage stores the result for zero-I/O hot-path lookups)
//
// # Declarations
//
//   - FunctionInfo / ExtractFunctionInfo extract name, return type, params,
//     and the declaration-vs-definition flag from a function_definition node
//   - FieldInfo / ExtractStructFields extract struct/class field name+type
//
// # Types
//
//   - ExtractTypeString assembles a complete C/C++ type string from the
//     primitive_type / type_identifier / qualified_identifier / template_type
//     and pointer_declarator / reference_declarator / type_qualifier nodes
//     produced by tree-sitter (e.g. "const std::vector<int>&", "char*")
//
// # Helpers
//
//   - ExtractParameters extracts (names, types) from a parameter_list
//   - ExtractCallInfo extracts target, arguments, and call-shape metadata
//     (free function vs method vs qualified) from a call_expression
//   - IsCKeyword / IsCppKeyword are used by statement extraction to filter
//     reserved words out of identifier lists
//
// All helpers are pure AST operations: they take *sitter.Node and []byte and
// return plain values. They have no dependency on graph.Node, graph.CodeGraph,
// or any other higher-level type. The parsers in graph/parser_c.go (PR-03)
// and graph/parser_cpp.go (PR-04) consume them.
package clike

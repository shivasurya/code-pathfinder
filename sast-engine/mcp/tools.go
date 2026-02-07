package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
)

// LSP Symbol Kind constants (Language Server Protocol specification).
// Maps Python symbol types to standardized LSP SymbolKind integers.
// Reference: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#symbolKind
const (
	SymbolKindFile        = 1  // File
	SymbolKindModule      = 2  // Module
	SymbolKindNamespace   = 3  // Namespace (not used in Python)
	SymbolKindPackage     = 4  // Package
	SymbolKindClass       = 5  // Class
	SymbolKindMethod      = 6  // Method
	SymbolKindProperty    = 7  // Property
	SymbolKindField       = 8  // Field
	SymbolKindConstructor = 9  // Constructor
	SymbolKindEnum        = 10 // Enum
	SymbolKindInterface   = 11 // Interface
	SymbolKindFunction    = 12 // Function
	SymbolKindVariable    = 13 // Variable
	SymbolKindConstant    = 14 // Constant
	SymbolKindString      = 15 // String (not used for symbols)
	SymbolKindNumber      = 16 // Number (not used for symbols)
	SymbolKindBoolean     = 17 // Boolean (not used for symbols)
	SymbolKindArray       = 18 // Array (not used for symbols)
	SymbolKindObject      = 19 // Object (not used for symbols)
	SymbolKindKey         = 20 // Key (not used for symbols)
	SymbolKindNull        = 21 // Null (not used for symbols)
	SymbolKindEnumMember  = 22 // EnumMember
	SymbolKindStruct      = 23 // Struct (dataclass)
	SymbolKindEvent       = 24 // Event (not used in Python)
	SymbolKindOperator    = 25 // Operator (special methods)
	SymbolKindTypeParam   = 26 // TypeParameter
)

// getSymbolKind maps Python symbol types to LSP SymbolKind integers and names.
// Returns (kind int, kindName string) for the given symbol type.
func getSymbolKind(symbolType string) (int, string) {
	switch symbolType {
	// Function types
	case "function_definition":
		return SymbolKindFunction, "Function"
	case "method":
		return SymbolKindMethod, "Method"
	case "constructor":
		return SymbolKindConstructor, "Constructor"
	case "property":
		return SymbolKindProperty, "Property"
	case "special_method":
		return SymbolKindOperator, "Operator"

	// Class types
	case "class_definition":
		return SymbolKindClass, "Class"
	case "interface":
		return SymbolKindInterface, "Interface"
	case "enum":
		return SymbolKindEnum, "Enum"
	case "dataclass":
		return SymbolKindStruct, "Struct"

	// Variable types
	case "module_variable":
		return SymbolKindVariable, "Variable"
	case "constant":
		return SymbolKindConstant, "Constant"
	case "class_field":
		return SymbolKindField, "Field"

	// Java types (for compatibility)
	case "method_declaration":
		return SymbolKindMethod, "Method"
	case "class_declaration":
		return SymbolKindClass, "Class"
	case "variable_declaration":
		return SymbolKindVariable, "Variable"

	// Unknown/default
	default:
		return SymbolKindVariable, "Unknown"
	}
}

// getToolDefinitions returns the complete tool schemas.
func (s *Server) getToolDefinitions() []Tool {
	return []Tool{
		{
			Name: "get_index_info",
			Description: `Get comprehensive statistics about the indexed Python codebase. Use this FIRST to understand the project scope before making other queries.

Returns:
- Project info: project_path, python_version, indexed_at, build_time_seconds
- Overall stats: total_symbols, call_edges, modules, files, taint_summaries, class_fields
- symbols_by_type: Breakdown by all 12 Python types (function_definition, method, constructor, property, special_method, class_definition, interface, enum, dataclass, module_variable, constant, class_field)
- symbols_by_lsp_kind: Breakdown by LSP Symbol Kind (Function, Method, Constructor, Property, Operator, Class, Interface, Enum, Struct, Variable, Constant, Field)
- top_modules: Top 10 modules by function count
- health: Index health indicators (average functions per module, etc.)

Use when: Starting analysis, understanding project size and structure, verifying index quality, or exploring symbol distribution.`,
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name: "find_symbol",
			Description: `Search and filter Python symbols by name and/or type across 12 symbol types. Supports partial matching. Results are paginated.

Symbol Types Available:
- Functions: function_definition, method, constructor, property, special_method
- Classes: class_definition, interface (Protocol/ABC), enum, dataclass
- Variables: module_variable, constant (UPPERCASE), class_field

Returns: For ALL symbols: fqn, file, line, type, symbol_kind (LSP integer), symbol_kind_name (human-readable).
For functions/methods: return_type, parameters, decorators. For classes: superclass, interfaces. For fields: inferred_type, confidence, assigned_in.

LSP Symbol Kinds: Function(12), Method(6), Constructor(9), Property(7), Operator(25), Class(5), Interface(11), Enum(10), Struct(23), Variable(13), Constant(14), Field(8).

Filtering: At least ONE of name/type/types/module must be provided. Filters can be combined (e.g., name="get" + type="method" + module="core.utils").

Use when: Looking for symbols by name; filtering by type; exploring codebase structure; finding definitions; analyzing symbol types; drilling down into specific modules.

Examples:
- find_symbol(name="login") - finds all symbols named login
- find_symbol(type="method") - lists all methods
- find_symbol(types=["interface","enum"]) - lists all interfaces and enums
- find_symbol(name="get", type="method") - finds methods named "get"
- find_symbol(name="User", type="class_definition") - finds User class only
- find_symbol(module="core.settings") - finds all symbols in core.settings module
- find_symbol(type="constant", module="core.settings.base") - finds constants in specific module
- find_symbol(module="data_manager", type="method") - finds all methods in data_manager package`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name":   {Type: "string", Description: "Symbol name to find. Optional. Can be: short name ('login'), partial name ('auth'), or FQN ('myapp.auth.login')"},
					"type":   {Type: "string", Description: "Filter by single symbol type. Optional. One of: function_definition, method, constructor, property, special_method, class_definition, interface, enum, dataclass, module_variable, constant, class_field"},
					"types":  {Type: "array", Description: "Filter by multiple symbol types. Optional. Array of type strings. Alternative to 'type' parameter"},
					"module": {Type: "string", Description: "Filter by module. Optional. Matches symbols whose FQN starts with the module path (e.g., 'core.settings', 'data_manager.models'). Works with all symbol types"},
					"limit":  {Type: "integer", Description: "Max results to return (default: 50, max: 500)"},
					"cursor": {Type: "string", Description: "Pagination cursor from previous response"},
				},
				Required: []string{},
			},
		},
		{
			Name: "find_module",
			Description: `Search for Python modules by name. Returns module information including file path and symbol counts.

Returns: module_fqn, file_path, functions_count (number of functions/methods in the module), and match_type (exact/partial).

Use when: Finding module locations, understanding module structure, or navigating between modules.

Examples:
- find_module("myapp.auth") - find the auth module
- find_module("utils") - find all modules named utils
- find_module("models.user") - find user module in models package`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {Type: "string", Description: "Module name to find. Can be FQN ('myapp.auth') or short name ('auth')"},
				},
				Required: []string{"name"},
			},
		},
		{
			Name: "list_modules",
			Description: `List all Python modules in the indexed project. Returns comprehensive module information.

Returns: Array of modules with module_fqn, file_path, and functions_count for each. Includes total_modules count.

Use when: Exploring project structure, getting an overview of all modules, or discovering what modules exist.

Examples:
- list_modules() - get all modules in the project`,
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name: "get_callers",
			Description: `Find all functions that CALL a given function (reverse call graph / incoming edges). Answer: "Who uses this function?" Results are paginated.

Returns: Target function info and list of callers with their FQN, file, line number, and the specific call site location. Includes pagination info.

IMPORTANT - Function Parameter Requirements:
- Must be a FUNCTION or METHOD name, NOT a module path
- Use find_symbol first to get the exact FQN of functions
- Short names work ("sanitize_input") if unique in the codebase
- Full FQNs are preferred ("myapp.utils.sanitize_input") to avoid ambiguity
- Method names should include class ("MyClass.save" or "myapp.models.MyClass.save")
- Test files are NOT indexed - only production code is queryable

What WORKS:
✓ "myapp.auth.login" - function FQN
✓ "User.save" - method with class name
✓ "sanitize_input" - short function name (if unique)
✓ "myapp.models.Project.get_all_tasks" - full method FQN

What DOESN'T work:
✗ "myapp.auth" - this is a module, not a function
✗ "test_login" - test files are not indexed
✗ Module-level code without a function wrapper

Use when: Understanding function usage, impact analysis before refactoring, finding entry points, or tracing how data flows INTO a function.

Examples:
- get_callers("sanitize_input") - who calls the sanitize function?
- get_callers("myapp.auth.login") - who calls the login function?
- get_callers("User.save") - who calls the User.save method?`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"function": {Type: "string", Description: "Function to find callers for. Use short name ('login') or FQN ('myapp.auth.login')"},
					"limit":    {Type: "integer", Description: "Max results to return (default: 50, max: 500)"},
					"cursor":   {Type: "string", Description: "Pagination cursor from previous response"},
				},
				Required: []string{"function"},
			},
		},
		{
			Name: "get_callees",
			Description: `Find all functions CALLED BY a given function (forward call graph / outgoing edges). Answer: "What does this function depend on?" Results are paginated.

Returns: Source function info, list of callees with target name, call line, resolution status (resolved/unresolved), and type inference info if available. Includes pagination info.

IMPORTANT - Function Parameter Requirements:
- Must be a FUNCTION or METHOD name, NOT a module path
- Use find_symbol first to get the exact FQN of the function you want to analyze
- Short names work ("process_payment") if the function name is unique
- Full FQNs are preferred ("myapp.payment.process_payment") to avoid ambiguity
- Method names should include class ("Payment.process" or "myapp.models.Payment.process")
- Test files are NOT indexed - only production code is queryable

What WORKS:
✓ "myapp.payment.process_payment" - function FQN
✓ "Payment.process" - method with class name
✓ "process_payment" - short function name (if unique in codebase)
✓ "myapp.models.Task.update_status" - full method FQN

What DOESN'T work:
✗ "myapp.payment" - this is a module, not a function
✗ "test_process_payment" - test files are not indexed
✗ Module-level code that isn't wrapped in a function

Type Inference in Results:
- Callees resolved via cross-file import resolution include "type_inference" metadata
- Shows inferred_type (e.g., "django.db.models.JSONField") for method calls on imported classes
- Confidence score (0.0-1.0) indicates resolution certainty
- High confidence (>0.9) means the call was resolved using import analysis

Use when: Understanding function dependencies, analyzing what a function does, tracing data flow FROM a function, finding unresolved external calls, or analyzing cross-file method calls.

Examples:
- get_callees("process_payment") - what functions does payment processing call?
- get_callees("myapp.payment.process_payment") - dependencies with full FQN
- get_callees("User.save") - what does the User.save method call?
- get_callees("myapp.tasks.models.Annotation.save") - analyze cross-file method calls`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"function": {Type: "string", Description: "Function to find callees for. Use short name ('process') or FQN ('myapp.payment.process')"},
					"limit":    {Type: "integer", Description: "Max results to return (default: 50, max: 500)"},
					"cursor":   {Type: "string", Description: "Pagination cursor from previous response"},
				},
				Required: []string{"function"},
			},
		},
		{
			Name: "get_call_details",
			Description: `Get detailed information about a SPECIFIC call from one function to another. Most detailed view of a single call site.

Returns: Full call site info including caller FQN, target, exact location (file, line, column), arguments passed, and resolution details (resolved status, failure reason if unresolved, type inference info).

Use when: Investigating a specific function call, understanding how arguments are passed, debugging why a call wasn't resolved, or analyzing type inference.

Examples:
- get_call_details("handle_request", "authenticate") - how does handle_request call authenticate?
- get_call_details("save_user", "execute") - examine the database call in save_user`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"caller": {Type: "string", Description: "The function making the call (short name or FQN)"},
					"callee": {Type: "string", Description: "The function being called (short name, will match partially)"},
				},
				Required: []string{"caller", "callee"},
			},
		},
		{
			Name: "resolve_import",
			Description: `Resolve a Python import path to its actual file location in the project.

Returns: Import resolution with file_path, module_fqn, match_type (exact/short_name/partial/ambiguous), and alternatives if multiple matches exist.

Use when: Finding where a module is defined, understanding import structure, or locating source files for external references.

Examples:
- resolve_import("myapp.auth.users") - find the users module
- resolve_import("utils") - find modules named utils (may return multiple)
- resolve_import("database") - locate database module`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"import": {Type: "string", Description: "Import path to resolve. Can be FQN ('myapp.auth.users') or short name ('users')"},
				},
				Required: []string{"import"},
			},
		},
	}
}

// executeTool runs a tool and returns the result.
func (s *Server) executeTool(name string, args map[string]interface{}) (string, bool) {
	switch name {
	case "get_index_info":
		return s.toolGetIndexInfo()
	case "find_symbol":
		return s.toolFindSymbol(args)
	case "find_module":
		moduleName, _ := args["name"].(string)
		return s.toolFindModule(moduleName)
	case "list_modules":
		return s.toolListModules()
	case "get_callers":
		return s.toolGetCallers(args)
	case "get_callees":
		return s.toolGetCallees(args)
	case "get_call_details":
		caller, _ := args["caller"].(string)
		callee, _ := args["callee"].(string)
		return s.toolGetCallDetails(caller, callee)
	case "resolve_import":
		importPath, _ := args["import"].(string)
		return s.toolResolveImport(importPath)
	default:
		return fmt.Sprintf(`{"error": "Unknown tool: %s"}`, name), true
	}
}

// ============================================================================
// Tool Implementations
// ============================================================================

// toolGetIndexInfo returns comprehensive index statistics including symbol type breakdown.
func (s *Server) toolGetIndexInfo() (string, bool) {
	// Count symbols by type and LSP kind.
	symbolsByType := make(map[string]int)
	symbolsByLSPKind := make(map[string]int)

	for _, node := range s.callGraph.Functions {
		symbolsByType[node.Type]++

		// Get LSP kind for this symbol.
		_, kindName := getSymbolKind(node.Type)
		symbolsByLSPKind[kindName]++
	}

	// Count class attributes if available.
	classFieldsCount := 0
	if s.callGraph.Attributes != nil {
		if attrRegistry, ok := s.callGraph.Attributes.(*registry.AttributeRegistry); ok {
			for _, classAttrs := range attrRegistry.Classes {
				classFieldsCount += len(classAttrs.Attributes)
			}
		}
	}

	// Calculate module statistics.
	moduleStats := make([]map[string]interface{}, 0, len(s.moduleRegistry.Modules))
	totalFunctionsInModules := 0

	for moduleFQN, filePath := range s.moduleRegistry.Modules {
		functionsCount := 0
		for fqn := range s.callGraph.Functions {
			if strings.HasPrefix(fqn, moduleFQN+".") {
				functionsCount++
			}
		}
		totalFunctionsInModules += functionsCount

		moduleStats = append(moduleStats, map[string]interface{}{
			"module_fqn":      moduleFQN,
			"file_path":       filePath,
			"functions_count": functionsCount,
		})
	}

	// Build comprehensive result.
	result := map[string]interface{}{
		"project_path":       s.projectPath,
		"python_version":     s.pythonVersion,
		"indexed_at":         s.indexedAt.Format("2006-01-02T15:04:05Z07:00"),
		"build_time_seconds": s.buildTime.Seconds(),

		// Overall statistics.
		"stats": map[string]interface{}{
			"total_symbols":   len(s.callGraph.Functions),
			"call_edges":      len(s.callGraph.Edges),
			"modules":         len(s.moduleRegistry.Modules),
			"files":           len(s.moduleRegistry.FileToModule),
			"taint_summaries": len(s.callGraph.Summaries),
			"class_fields":    classFieldsCount,
		},

		// Symbol breakdown by Python type (12 types).
		"symbols_by_type": symbolsByType,

		// Symbol breakdown by LSP Symbol Kind (human-readable).
		"symbols_by_lsp_kind": symbolsByLSPKind,

		// Module statistics (top 10 by function count).
		"top_modules": getTopModules(moduleStats, 10),

		// Index health indicators.
		"health": map[string]interface{}{
			"indexed_symbols":          len(s.callGraph.Functions),
			"symbols_with_call_edges":  len(s.callGraph.Edges),
			"modules_indexed":          len(s.moduleRegistry.Modules),
			"average_functions_per_module": float64(totalFunctionsInModules) / float64(maxInt(len(s.moduleRegistry.Modules), 1)),
		},
	}

	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// getTopModules returns the top N modules by function count.
func getTopModules(moduleStats []map[string]interface{}, limit int) []map[string]interface{} {
	// Sort by functions_count descending.
	type moduleStat struct {
		data           map[string]interface{}
		functionsCount int
	}

	stats := make([]moduleStat, len(moduleStats))
	for i, m := range moduleStats {
		stats[i] = moduleStat{
			data:           m,
			functionsCount: m["functions_count"].(int),
		}
	}

	// Simple bubble sort for top N (good enough for small N).
	for i := 0; i < len(stats) && i < limit; i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].functionsCount > stats[i].functionsCount {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	// Return top N.
	result := make([]map[string]interface{}, 0, limit)
	for i := 0; i < len(stats) && i < limit; i++ {
		result = append(result, stats[i].data)
	}

	return result
}

// max returns the maximum of two integers.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// toolFindSymbol finds symbols by name with pagination support.
// Searches all 12 Python symbol types: functions, methods, constructors, properties,
// special methods, classes, interfaces, enums, dataclasses, module variables, constants, and class fields.
func (s *Server) toolFindSymbol(args map[string]interface{}) (string, bool) {
	name, _ := args["name"].(string)
	singleType, _ := args["type"].(string)
	moduleFilter, _ := args["module"].(string)

	// Handle types parameter (array).
	var typeFilter []string
	if typesParam, ok := args["types"].([]interface{}); ok {
		for _, t := range typesParam {
			if typeStr, ok := t.(string); ok {
				typeFilter = append(typeFilter, typeStr)
			}
		}
	}

	// Validation: at least one filter must be provided.
	if name == "" && singleType == "" && len(typeFilter) == 0 && moduleFilter == "" {
		return `{"error": "At least one filter required: provide 'name', 'type', 'types', or 'module' parameter"}`, true
	}

	// Validate type/types conflict.
	if singleType != "" && len(typeFilter) > 0 {
		return `{"error": "Cannot specify both 'type' and 'types' parameters. Use one or the other"}`, true
	}

	// Convert singleType to typeFilter array for unified processing.
	if singleType != "" {
		typeFilter = []string{singleType}
	}

	// Validate type names.
	validTypes := map[string]bool{
		"function_definition": true,
		"method":              true,
		"constructor":         true,
		"property":            true,
		"special_method":      true,
		"class_definition":    true,
		"interface":           true,
		"enum":                true,
		"dataclass":           true,
		"module_variable":     true,
		"constant":            true,
		"class_field":         true,
	}

	for _, t := range typeFilter {
		if !validTypes[t] {
			return fmt.Sprintf(`{"error": "Invalid symbol type: %s", "valid_types": ["function_definition","method","constructor","property","special_method","class_definition","interface","enum","dataclass","module_variable","constant","class_field"]}`, t), true
		}
	}

	// Build type filter map for O(1) lookup.
	typeFilterMap := make(map[string]bool)
	for _, t := range typeFilter {
		typeFilterMap[t] = true
	}

	// Extract pagination params.
	pageParams, err := ExtractPaginationParams(args)
	if err != nil {
		return NewToolError(err.Message, err.Code, err.Data), true
	}

	var allMatches []map[string]interface{}

	// Search functions, methods, constructors, properties, special methods, and classes.
	for fqn, node := range s.callGraph.Functions {
		// Apply type filter if specified.
		if len(typeFilterMap) > 0 && !typeFilterMap[node.Type] {
			continue
		}

		// Apply module filter if specified.
		if !matchesModuleFilter(fqn, moduleFilter) {
			continue
		}

		// Apply name filter if specified.
		nameMatches := name == ""
		if name != "" {
			shortName := getShortName(fqn)
			nameMatches = shortName == name || strings.HasSuffix(fqn, "."+name) || fqn == name || strings.Contains(fqn, name)
		}

		if nameMatches {
			// Get LSP symbol kind.
			symbolKind, symbolKindName := getSymbolKind(node.Type)

			match := map[string]interface{}{
				"fqn":              fqn,
				"file":             node.File,
				"line":             node.LineNumber,
				"type":             node.Type,
				"symbol_kind":      symbolKind,
				"symbol_kind_name": symbolKindName,
			}

			// Add optional fields if available.
			if node.ReturnType != "" {
				match["return_type"] = node.ReturnType
			}
			if len(node.MethodArgumentsType) > 0 {
				match["parameters"] = node.MethodArgumentsType
			}
			if node.Modifier != "" {
				match["modifier"] = node.Modifier
			}
			if len(node.Annotation) > 0 {
				match["decorators"] = node.Annotation
			}
			if node.SuperClass != "" {
				match["superclass"] = node.SuperClass
			}
			if len(node.Interface) > 0 {
				match["interfaces"] = node.Interface
			}

			allMatches = append(allMatches, match)
		}
	}

	// Search class attributes if AttributeRegistry is available.
	if s.callGraph.Attributes != nil {
		// Only search class fields if type filter allows it.
		searchClassFields := len(typeFilterMap) == 0 || typeFilterMap["class_field"]

		if searchClassFields {
			if attrRegistry, ok := s.callGraph.Attributes.(*registry.AttributeRegistry); ok {
				for classFQN, classAttrs := range attrRegistry.Classes {
					for attrName, attr := range classAttrs.Attributes {
						attributeFQN := classFQN + "." + attrName

						// Apply module filter if specified.
						if !matchesModuleFilter(attributeFQN, moduleFilter) {
							continue
						}

						// Apply name filter if specified.
						nameMatches := name == ""
						if name != "" {
							nameMatches = attrName == name || strings.Contains(attrName, name) ||
								strings.HasSuffix(attributeFQN, "."+name) ||
								strings.Contains(attributeFQN, name)
						}

						if nameMatches {
							// Get LSP symbol kind for class_field.
							symbolKind, symbolKindName := getSymbolKind("class_field")
							match := map[string]interface{}{
								"fqn":              attributeFQN,
								"type":             "class_field",
								"symbol_kind":      symbolKind,
								"symbol_kind_name": symbolKindName,
								"class":            classFQN,
								"name":             attrName,
							}

							// Add location if available.
							if attr.Location != nil {
								match["file"] = attr.Location.File
								// Note: SourceLocation uses byte offsets, not line numbers.
								// Could convert but would require reading file - skip for now.
							}

							// Add type information if available.
							if attr.Type != nil && attr.Type.TypeFQN != "" {
								match["inferred_type"] = attr.Type.TypeFQN
								match["confidence"] = attr.Confidence
							}

							// Add assignment location.
							if attr.AssignedIn != "" {
								match["assigned_in"] = attr.AssignedIn
							}

							allMatches = append(allMatches, match)
						}
					}
				}
			}
		}
	}

	// Search codeGraph.Nodes for class definitions and variables.
	// These types are stored in the raw AST graph, not in callGraph.Functions.
	missingTypes := map[string]bool{
		"class_definition": true,
		"interface":        true,
		"enum":             true,
		"dataclass":        true,
		"module_variable":  true,
		"constant":         true,
	}

	// Only search if we're looking for these types or no type filter specified.
	searchCodeGraph := len(typeFilterMap) == 0
	for t := range typeFilterMap {
		if missingTypes[t] {
			searchCodeGraph = true
			break
		}
	}

	if searchCodeGraph && s.codeGraph != nil {
		// Build class context for determining which class constants/fields belong to.
		// This ensures class-level constants get proper FQNs: module.ClassName.CONSTANT_NAME
		classContext := buildClassContext(s.codeGraph)

		for _, node := range s.codeGraph.Nodes {
			// Skip if not one of the missing types.
			if !missingTypes[node.Type] {
				continue
			}

			// Apply type filter if specified.
			if len(typeFilterMap) > 0 && !typeFilterMap[node.Type] {
				continue
			}

			// Build FQN for this node.
			// Classes: module.ClassName
			// Module-level variables/constants: module.VARIABLE_NAME
			// Class-level constants/fields: module.ClassName.CONSTANT_NAME
			modulePath, ok := s.moduleRegistry.FileToModule[node.File]
			if !ok {
				continue
			}

			// Use helper function to build class-qualified FQN for class-level symbols.
			fqn := buildNodeFQN(modulePath, node, classContext)

			// Apply module filter if specified.
			if !matchesModuleFilter(fqn, moduleFilter) {
				continue
			}

			// Apply name filter if specified.
			nameMatches := name == ""
			if name != "" {
				shortName := node.Name
				nameMatches = shortName == name || strings.HasSuffix(fqn, "."+name) || fqn == name || strings.Contains(fqn, name)
			}

			if nameMatches {
				// Get LSP symbol kind.
				symbolKind, symbolKindName := getSymbolKind(node.Type)

				match := map[string]interface{}{
					"fqn":              fqn,
					"file":             node.File,
					"line":             node.LineNumber,
					"type":             node.Type,
					"symbol_kind":      symbolKind,
					"symbol_kind_name": symbolKindName,
				}

				// Add optional fields if available.
				if len(node.Annotation) > 0 {
					match["decorators"] = node.Annotation
				}
				if node.SuperClass != "" {
					match["superclass"] = node.SuperClass
				}
				if len(node.Interface) > 0 {
					match["interfaces"] = node.Interface
				}
				if node.Modifier != "" {
					match["modifier"] = node.Modifier
				}

				allMatches = append(allMatches, match)
			}
		}
	}

	if len(allMatches) == 0 {
		// Build helpful error message.
		filters := []string{}
		if name != "" {
			filters = append(filters, fmt.Sprintf("name=%s", name))
		}
		if len(typeFilter) > 0 {
			filters = append(filters, fmt.Sprintf("types=%v", typeFilter))
		}
		if moduleFilter != "" {
			filters = append(filters, fmt.Sprintf("module=%s", moduleFilter))
		}
		filterStr := strings.Join(filters, ", ")
		return fmt.Sprintf(`{"error": "No symbols found", "filters": "%s", "suggestion": "Try different filters or partial name matching"}`, filterStr), true
	}

	// Apply pagination.
	matches, pageInfo := PaginateSlice(allMatches, pageParams)

	// Build filters_applied info for response.
	filtersApplied := map[string]interface{}{}
	if name != "" {
		filtersApplied["name"] = name
	}
	if len(typeFilter) > 0 {
		if len(typeFilter) == 1 {
			filtersApplied["type"] = typeFilter[0]
		} else {
			filtersApplied["types"] = typeFilter
		}
	}
	if moduleFilter != "" {
		filtersApplied["module"] = moduleFilter
	}

	result := map[string]interface{}{
		"filters_applied": filtersApplied,
		"matches":         matches,
		"pagination":      pageInfo,
	}
	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// toolFindModule searches for Python modules by name.
func (s *Server) toolFindModule(name string) (string, bool) {
	if name == "" {
		return `{"error": "name parameter is required"}`, true
	}

	// Try exact match first.
	if filePath, ok := s.moduleRegistry.Modules[name]; ok {
		// Count functions in this module.
		functionsCount := 0
		for fqn := range s.callGraph.Functions {
			if strings.HasPrefix(fqn, name+".") {
				functionsCount++
			}
		}

		result := map[string]interface{}{
			"module_fqn":      name,
			"file_path":       filePath,
			"match_type":      "exact",
			"functions_count": functionsCount,
		}
		bytes, _ := json.MarshalIndent(result, "", "  ")
		return string(bytes), false
	}

	// Try partial match.
	var matches []map[string]interface{}
	for moduleFQN, filePath := range s.moduleRegistry.Modules {
		if strings.Contains(moduleFQN, name) {
			// Count functions in this module.
			functionsCount := 0
			for fqn := range s.callGraph.Functions {
				if strings.HasPrefix(fqn, moduleFQN+".") {
					functionsCount++
				}
			}

			matches = append(matches, map[string]interface{}{
				"module_fqn":      moduleFQN,
				"file_path":       filePath,
				"match_type":      "partial",
				"functions_count": functionsCount,
			})
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf(`{"error": "Module not found: %s", "suggestion": "Check module name or try a partial match"}`, name), true
	}

	if len(matches) == 1 {
		// Single match.
		bytes, _ := json.MarshalIndent(matches[0], "", "  ")
		return string(bytes), false
	}

	// Multiple matches.
	result := map[string]interface{}{
		"query":         name,
		"matches":       matches,
		"matches_count": len(matches),
	}
	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// toolListModules lists all modules in the project.
func (s *Server) toolListModules() (string, bool) {
	modules := make([]map[string]interface{}, 0, len(s.moduleRegistry.Modules))

	for moduleFQN, filePath := range s.moduleRegistry.Modules {
		// Count functions in this module.
		functionsCount := 0
		for fqn := range s.callGraph.Functions {
			if strings.HasPrefix(fqn, moduleFQN+".") {
				functionsCount++
			}
		}

		modules = append(modules, map[string]interface{}{
			"module_fqn":      moduleFQN,
			"file_path":       filePath,
			"functions_count": functionsCount,
		})
	}

	result := map[string]interface{}{
		"modules":       modules,
		"total_modules": len(modules),
	}
	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// toolGetCallers finds all callers of a function with pagination support.
func (s *Server) toolGetCallers(args map[string]interface{}) (string, bool) {
	function, _ := args["function"].(string)
	if function == "" {
		return `{"error": "function parameter is required"}`, true
	}

	// Extract pagination params.
	pageParams, err := ExtractPaginationParams(args)
	if err != nil {
		return NewToolError(err.Message, err.Code, err.Data), true
	}

	fqns := s.findMatchingFQNs(function)
	if len(fqns) == 0 {
		return fmt.Sprintf(`{"error": "Function not found: %s"}`, function), true
	}

	// Use first match.
	targetFQN := fqns[0]
	targetNode := s.callGraph.Functions[targetFQN]

	// Get callers from reverse edges.
	callerFQNs := s.callGraph.ReverseEdges[targetFQN]

	allCallers := make([]map[string]interface{}, 0, len(callerFQNs))
	for _, callerFQN := range callerFQNs {
		callerNode := s.callGraph.Functions[callerFQN]
		if callerNode == nil {
			continue
		}

		caller := map[string]interface{}{
			"fqn":  callerFQN,
			"name": getShortName(callerFQN),
			"file": callerNode.File,
			"line": callerNode.LineNumber,
		}

		// Find the specific call site location.
		for _, cs := range s.callGraph.CallSites[callerFQN] {
			if cs.TargetFQN == targetFQN || cs.Target == getShortName(targetFQN) {
				caller["call_line"] = cs.Location.Line
				caller["call_column"] = cs.Location.Column
				break
			}
		}

		allCallers = append(allCallers, caller)
	}

	// Apply pagination.
	callers, pageInfo := PaginateSlice(allCallers, pageParams)

	result := map[string]interface{}{
		"target": map[string]interface{}{
			"fqn":  targetFQN,
			"name": getShortName(targetFQN),
			"file": targetNode.File,
			"line": targetNode.LineNumber,
		},
		"callers":    callers,
		"pagination": pageInfo,
	}

	if len(fqns) > 1 {
		result["note"] = fmt.Sprintf("Multiple matches found (%d). Showing callers for first match. Other matches: %v", len(fqns), fqns[1:])
	}

	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// toolGetCallees finds all functions called by a function.
func (s *Server) toolGetCallees(args map[string]interface{}) (string, bool) {
	function, _ := args["function"].(string)
	if function == "" {
		return `{"error": "function parameter is required"}`, true
	}

	// Extract pagination params.
	pageParams, rpcErr := ExtractPaginationParams(args)
	if rpcErr != nil {
		return fmt.Sprintf(`{"error": "%s"}`, rpcErr.Message), true
	}

	fqns := s.findMatchingFQNs(function)
	if len(fqns) == 0 {
		return fmt.Sprintf(`{"error": "Function not found: %s"}`, function), true
	}

	sourceFQN := fqns[0]
	sourceNode := s.callGraph.Functions[sourceFQN]

	// Get call sites for this function.
	callSites := s.callGraph.CallSites[sourceFQN]

	allCallees := make([]map[string]interface{}, 0, len(callSites))
	resolvedCount := 0
	unresolvedCount := 0

	for _, cs := range callSites {
		callee := map[string]interface{}{
			"target":    cs.Target,
			"call_line": cs.Location.Line,
			"resolved":  cs.Resolved,
		}

		if cs.Resolved {
			resolvedCount++
			callee["target_fqn"] = cs.TargetFQN

			// Try to get file info for resolved target.
			if targetNode := s.callGraph.Functions[cs.TargetFQN]; targetNode != nil {
				callee["target_file"] = targetNode.File
				callee["target_line"] = targetNode.LineNumber
			}
		} else {
			unresolvedCount++
			if cs.FailureReason != "" {
				callee["failure_reason"] = cs.FailureReason
			}
		}

		// Include type inference info if used.
		if cs.ResolvedViaTypeInference {
			callee["type_inference"] = map[string]interface{}{
				"inferred_type":   cs.InferredType,
				"type_confidence": cs.TypeConfidence,
			}
		}

		allCallees = append(allCallees, callee)
	}

	// Apply pagination.
	callees, pageInfo := PaginateSlice(allCallees, pageParams)

	result := map[string]interface{}{
		"source": map[string]interface{}{
			"fqn":  sourceFQN,
			"name": getShortName(sourceFQN),
			"file": sourceNode.File,
			"line": sourceNode.LineNumber,
		},
		"callees":          callees,
		"pagination":       pageInfo,
		"resolved_count":   resolvedCount,
		"unresolved_count": unresolvedCount,
	}

	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// toolGetCallDetails gets detailed info about a specific call site.
func (s *Server) toolGetCallDetails(callerName, calleeName string) (string, bool) {
	if callerName == "" || calleeName == "" {
		return `{"error": "caller and callee parameters are required"}`, true
	}

	callerFQNs := s.findMatchingFQNs(callerName)
	if len(callerFQNs) == 0 {
		return fmt.Sprintf(`{"error": "Caller function not found: %s"}`, callerName), true
	}

	callerFQN := callerFQNs[0]
	callSites := s.callGraph.CallSites[callerFQN]

	// Find matching call site.
	for _, cs := range callSites {
		if strings.Contains(cs.Target, calleeName) || strings.Contains(cs.TargetFQN, calleeName) {
			callSite := map[string]interface{}{
				"caller_fqn": callerFQN,
				"target":     cs.Target,
				"target_fqn": cs.TargetFQN,
				"location": map[string]interface{}{
					"file":   cs.Location.File,
					"line":   cs.Location.Line,
					"column": cs.Location.Column,
				},
				"resolved": cs.Resolved,
			}

			// Add arguments if available.
			if len(cs.Arguments) > 0 {
				args := make([]map[string]interface{}, len(cs.Arguments))
				for i, arg := range cs.Arguments {
					args[i] = map[string]interface{}{
						"position": arg.Position,
						"value":    arg.Value,
					}
				}
				callSite["arguments"] = args
			}

			// Add resolution info.
			resolution := map[string]interface{}{
				"resolved": cs.Resolved,
			}
			if !cs.Resolved && cs.FailureReason != "" {
				resolution["failure_reason"] = cs.FailureReason
			}
			if cs.ResolvedViaTypeInference {
				resolution["via_type_inference"] = true
				resolution["inferred_type"] = cs.InferredType
				resolution["type_confidence"] = cs.TypeConfidence
				resolution["type_source"] = cs.TypeSource
			}
			callSite["resolution"] = resolution

			result := map[string]interface{}{
				"call_site": callSite,
			}
			bytes, _ := json.MarshalIndent(result, "", "  ")
			return string(bytes), false
		}
	}

	return fmt.Sprintf(`{"error": "Call site not found: %s -> %s", "suggestion": "Check that the caller actually calls the callee"}`, callerName, calleeName), true
}

// toolResolveImport resolves an import path to file location.
func (s *Server) toolResolveImport(importPath string) (string, bool) {
	if importPath == "" {
		return `{"error": "import parameter is required"}`, true
	}

	// Try exact match first.
	if filePath, ok := s.moduleRegistry.Modules[importPath]; ok {
		result := map[string]interface{}{
			"import":       importPath,
			"resolved":     true,
			"file_path":    filePath,
			"module_fqn":   importPath,
			"match_type":   "exact",
			"alternatives": []interface{}{},
		}
		bytes, _ := json.MarshalIndent(result, "", "  ")
		return string(bytes), false
	}

	// Try short name lookup.
	shortName := getShortName(importPath)
	if files, ok := s.moduleRegistry.ShortNames[shortName]; ok && len(files) > 0 {
		if len(files) == 1 {
			// Unique match.
			filePath := files[0]
			moduleFQN := s.moduleRegistry.FileToModule[filePath]
			result := map[string]interface{}{
				"import":       importPath,
				"resolved":     true,
				"file_path":    filePath,
				"module_fqn":   moduleFQN,
				"match_type":   "short_name",
				"alternatives": []interface{}{},
			}
			bytes, _ := json.MarshalIndent(result, "", "  ")
			return string(bytes), false
		}

		// Multiple matches - return alternatives.
		alternatives := make([]map[string]string, len(files))
		for i, f := range files {
			alternatives[i] = map[string]string{
				"fqn":  s.moduleRegistry.FileToModule[f],
				"file": f,
			}
		}
		result := map[string]interface{}{
			"import":       importPath,
			"resolved":     false,
			"match_type":   "ambiguous",
			"alternatives": alternatives,
			"suggestion":   "Multiple modules match. Use fully qualified import path.",
		}
		bytes, _ := json.MarshalIndent(result, "", "  ")
		return string(bytes), false
	}

	// Try partial match.
	var partialMatches []map[string]string
	for moduleFQN, filePath := range s.moduleRegistry.Modules {
		if strings.Contains(moduleFQN, importPath) {
			partialMatches = append(partialMatches, map[string]string{
				"fqn":  moduleFQN,
				"file": filePath,
			})
		}
	}

	if len(partialMatches) > 0 {
		result := map[string]interface{}{
			"import":       importPath,
			"resolved":     false,
			"match_type":   "partial",
			"alternatives": partialMatches,
			"suggestion":   "No exact match. Did you mean one of these?",
		}
		bytes, _ := json.MarshalIndent(result, "", "  ")
		return string(bytes), false
	}

	return fmt.Sprintf(`{"error": "Import not found: %s", "suggestion": "Check if the module is in the indexed project path"}`, importPath), true
}

// ============================================================================
// Helper Functions
// ============================================================================

// findMatchingFQNs finds all FQNs matching a name.
func (s *Server) findMatchingFQNs(name string) []string {
	var matches []string
	for fqn := range s.callGraph.Functions {
		shortName := getShortName(fqn)
		if shortName == name || strings.HasSuffix(fqn, "."+name) || fqn == name {
			matches = append(matches, fqn)
		}
	}
	return matches
}

// getShortName extracts the last part of a FQN.
func getShortName(fqn string) string {
	parts := strings.Split(fqn, ".")
	if len(parts) == 0 {
		return fqn
	}
	return parts[len(parts)-1]
}

// buildClassContext creates a map of file locations to class names.
// This allows us to determine which class a constant/field belongs to based on its location.
// Returns a map with keys in format "file:startByte:endByte" → className.
func buildClassContext(codeGraph *graph.CodeGraph) map[string]string {
	classCtx := make(map[string]string)

	// Find all class definitions (including enums, interfaces, dataclasses).
	for _, node := range codeGraph.Nodes {
		if node.Type == "class_definition" || node.Type == "interface" ||
			node.Type == "enum" || node.Type == "dataclass" {
			// For each class, store its byte range.
			// Class-level constants/fields within this range belong to this class.
			if node.SourceLocation != nil {
				// Store class name by file + start/end bytes.
				key := fmt.Sprintf("%s:%d:%d", node.File, node.SourceLocation.StartByte, node.SourceLocation.EndByte)
				classCtx[key] = node.Name
			}
		}
	}

	return classCtx
}

// findContainingClass determines which class a node belongs to based on its byte location.
// Returns the class name if found, or empty string if the node is at module level.
func findContainingClass(node *graph.Node, classContext map[string]string) string {
	if node.SourceLocation == nil {
		return ""
	}

	// Find the smallest (most specific) class that contains this node.
	// This handles nested classes correctly (returns innermost class).
	var bestMatch string
	var bestRange uint32 = ^uint32(0) // Max uint32

	for key, className := range classContext {
		// Parse key format: "/path/to/file.py:startByte:endByte"
		// Use strings.LastIndex to find the last two colons (for byte ranges).
		lastColon := strings.LastIndex(key, ":")
		if lastColon == -1 {
			continue
		}
		secondLastColon := strings.LastIndex(key[:lastColon], ":")
		if secondLastColon == -1 {
			continue
		}

		// Extract components.
		file := key[:secondLastColon]
		classStartStr := key[secondLastColon+1 : lastColon]
		classEndStr := key[lastColon+1:]

		// Parse byte positions.
		var classStart, classEnd uint32
		if _, err := fmt.Sscanf(classStartStr, "%d", &classStart); err != nil {
			continue
		}
		if _, err := fmt.Sscanf(classEndStr, "%d", &classEnd); err != nil {
			continue
		}

		// Check if node is within this class's byte range.
		if file == node.File &&
			node.SourceLocation.StartByte >= classStart &&
			node.SourceLocation.EndByte <= classEnd {
			// Calculate class range size.
			classRange := classEnd - classStart

			// Keep the smallest containing class (most specific).
			if classRange < bestRange {
				bestMatch = className
				bestRange = classRange
			}
		}
	}

	return bestMatch
}

// matchesModuleFilter checks if an FQN matches the module filter.
// Returns true if no filter specified, or if FQN starts with the module path.
func matchesModuleFilter(fqn string, moduleFilter string) bool {
	if moduleFilter == "" {
		return true
	}

	// Exact match: module filter is the entire FQN.
	if fqn == moduleFilter {
		return true
	}

	// Prefix match: FQN starts with "moduleFilter."
	// This ensures "core.settings" matches "core.settings.base.DEBUG"
	// but NOT "core.settings_backup.X"
	return strings.HasPrefix(fqn, moduleFilter+".")
}

// buildNodeFQN constructs the fully qualified name for a node.
// For class-level symbols: module.ClassName.symbolName.
// For module-level symbols: module.symbolName.
func buildNodeFQN(modulePath string, node *graph.Node, classContext map[string]string) string {
	// For class-level constants and fields, find the containing class.
	if node.Scope == "class" {
		className := findContainingClass(node, classContext)
		if className != "" {
			return fmt.Sprintf("%s.%s.%s", modulePath, className, node.Name)
		}
	}

	// For module-level or if class not found, use simple FQN.
	return modulePath + "." + node.Name
}

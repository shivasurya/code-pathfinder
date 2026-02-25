package mcp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	dockerpkg "github.com/shivasurya/code-pathfinder/sast-engine/mcp/docker"
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
	case "parameter":
		return SymbolKindVariable, "Variable"

	// Java types (for compatibility)
	case "method_declaration":
		return SymbolKindMethod, "Method"
	case "class_declaration":
		return SymbolKindClass, "Class"
	case "variable_declaration":
		return SymbolKindVariable, "Variable"

	// Go types
	case "function_declaration":
		return SymbolKindFunction, "Function"
	case "init_function":
		return SymbolKindFunction, "Function"
	case "struct_definition":
		return SymbolKindStruct, "Struct"
	case "type_alias":
		return SymbolKindTypeParam, "TypeAlias"
	case "package_variable":
		return SymbolKindVariable, "Variable"
	case "variable_assignment":
		return SymbolKindVariable, "Variable"
	case "func_literal":
		return SymbolKindFunction, "Function"

	// Docker types
	case "dockerfile_instruction":
		return SymbolKindConstant, "DockerInstruction"
	case "compose_service":
		return SymbolKindModule, "ComposeService"

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
			Description: `Get comprehensive statistics about the indexed codebase (Python, Go, Java). Use this FIRST to understand the project scope before making other queries.

Returns:
- Project info: project_path, python_version, indexed_at, build_time_seconds
- Overall stats: total_symbols, call_edges, modules, files, taint_summaries, class_fields
- symbols_by_type: Breakdown by all 13 Python types (function_definition, method, constructor, property, special_method, class_definition, interface, enum, dataclass, module_variable, constant, class_field, parameter)
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
			Description: `Search and filter symbols by name and/or type. Supports Python, Go, and Java. Supports partial matching. Results are paginated.

Symbol Types Available:
- Python Functions: function_definition, method, constructor, property, special_method
- Python Classes: class_definition, interface (Protocol/ABC), enum, dataclass
- Python Variables: module_variable, constant (UPPERCASE), class_field, parameter
- Go Functions: function_declaration, init_function, method, func_literal
- Go Types: struct_definition, interface, type_alias
- Go Variables: package_variable, constant, variable_assignment

Returns: For ALL symbols: fqn, file, line, type, symbol_kind (LSP integer), symbol_kind_name (human-readable).
For functions/methods: return_type, parameters, decorators. For classes: superclass, interfaces. For fields: inferred_type, confidence, assigned_in.
For parameters: inferred_type (type annotation), parent_fqn (containing function).

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
- find_symbol(module="data_manager", type="method") - finds all methods in data_manager package
- find_symbol(type="parameter") - lists all typed parameters
- find_symbol(type="parameter", module="core.utils") - finds parameters in specific module`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name":   {Type: "string", Description: "Symbol name to find. Optional. Can be: short name ('login'), partial name ('auth'), or FQN ('myapp.auth.login')"},
					"type":   {Type: "string", Description: "Filter by single symbol type. Optional. One of: function_definition, method, constructor, property, special_method, class_definition, interface, enum, dataclass, module_variable, constant, class_field, parameter"},
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
		{
			Name: "find_dockerfile_instructions",
			Description: `Search Dockerfile instructions with semantic filtering. Supports instruction-specific filters (has_digest for FROM, user for USER, port for EXPOSE).

Returns:
- file, line, instruction, raw_content
- Parsed details (base_image, tag, digest for FROM; user, group for USER; port, protocol for EXPOSE)

Use when: Finding specific instructions, locating images, analyzing exposed ports, or understanding Dockerfile structure.

Examples:
- find_dockerfile_instructions(instruction_type="FROM", has_digest=false) - find unpinned base images
- find_dockerfile_instructions(instruction_type="USER", user="appuser") - find specific user
- find_dockerfile_instructions(instruction_type="EXPOSE", port=8080) - find services on port 8080
- find_dockerfile_instructions(instruction_type="FROM", base_image="python") - find Python-based images
- find_dockerfile_instructions(file_path="api/Dockerfile") - get all instructions from specific Dockerfile`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"instruction_type": {Type: "string", Description: "Filter by instruction type (FROM, RUN, USER, EXPOSE, COPY, ADD, WORKDIR, etc.)"},
					"file_path":        {Type: "string", Description: "Filter by Dockerfile path (supports partial matching)"},
					"base_image":       {Type: "string", Description: "Filter FROM instructions by base image name (e.g., 'python', 'alpine')"},
					"port":             {Type: "integer", Description: "Filter EXPOSE instructions by port number"},
					"has_digest":       {Type: "boolean", Description: "Filter FROM instructions by digest pinning (true=pinned, false=unpinned)"},
					"user":             {Type: "string", Description: "Filter USER instructions by username"},
					"limit":            {Type: "integer", Description: "Maximum results to return (default: 100)"},
				},
				Required: []string{},
			},
		},
		{
			Name: "find_compose_services",
			Description: `Search docker-compose services with filtering. Supports configuration filters (has_privileged, has_volume, exposes_port).

Returns:
- service_name, file, line
- Configuration (image, build, ports, volumes, environment, privileged, network_mode)

Use when: Finding specific services, analyzing service configurations, or understanding multi-container architecture.

Examples:
- find_compose_services(has_privileged=true) - find privileged containers
- find_compose_services(has_volume="/var/run/docker.sock") - find services with Docker socket
- find_compose_services(exposes_port=8080) - find services on port 8080
- find_compose_services(service_name="db") - find database services
- find_compose_services(file_path="docker-compose.yml") - get all services from specific file`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"service_name":   {Type: "string", Description: "Filter by service name (supports partial matching)"},
					"file_path":      {Type: "string", Description: "Filter by docker-compose.yml path"},
					"has_privileged": {Type: "boolean", Description: "Filter by privileged mode (true=privileged containers only)"},
					"exposes_port":   {Type: "integer", Description: "Filter services exposing specific port"},
					"has_volume":     {Type: "string", Description: "Filter services with specific volume path (e.g., '/var/run/docker.sock')"},
					"limit":          {Type: "integer", Description: "Maximum results to return (default: 100)"},
				},
				Required: []string{},
			},
		},
		{
			Name: "get_dockerfile_details",
			Description: `Get complete breakdown of a Dockerfile with all instructions and multi-stage analysis.

Returns:
- file, total_instructions
- instructions: Array of parsed instructions with details
- multi_stage: is_multi_stage, base_image, stages (array of stage aliases)
- summary: has_user_instruction, has_healthcheck, unpinned_images

Use when: Understanding complete Dockerfile structure, analyzing multi-stage builds, or reviewing all instructions.

Examples:
- get_dockerfile_details(file_path="/app/Dockerfile") - get complete Dockerfile breakdown
- get_dockerfile_details(file_path="Dockerfile") - analyze Dockerfile in current context`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path": {Type: "string", Description: "Path to Dockerfile (required)"},
				},
				Required: []string{"file_path"},
			},
		},
		{
			Name: "get_docker_dependencies",
			Description: `Retrieves dependency information for Docker entities (compose services or Dockerfile stages).

Use cases:
- Map docker-compose service dependencies (depends_on relationships)
- Analyze multi-stage Dockerfile build dependencies (COPY --from)
- Traverse upstream/downstream dependency chains

Examples:
- get_docker_dependencies(type="compose", name="web") - find all dependencies for "web" service
- get_docker_dependencies(type="dockerfile", name="builder", file_path="Dockerfile") - find stage dependencies
- get_docker_dependencies(type="compose", name="api", direction="upstream", max_depth=2) - upstream only, 2 levels

Parameters:
- type: Entity type - "compose" for docker-compose services or "dockerfile" for Dockerfile stages (required)
- name: Entity name - service name (for compose) or stage name/alias (for dockerfile) (required)
- file_path: Filter to specific file path (optional)
- direction: Traversal direction - "upstream" (dependencies), "downstream" (dependents), or "both" (default: "both")
- max_depth: Maximum traversal depth (default: 10)

Returns:
- target: Target entity name
- type: Entity type (compose or dockerfile)
- file: Source file path
- line: Line number
- direction: Traversal direction used
- max_depth: Maximum depth used
- upstream: Array of dependencies (entities this depends on)
- downstream: Array of dependents (entities that depend on this)
- dependency_chain: Simple chain string (e.g., "db → api → web")`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"type": {
						Type:        "string",
						Description: "Entity type: \"compose\" for docker-compose services or \"dockerfile\" for Dockerfile stages",
					},
					"name": {
						Type:        "string",
						Description: "Entity name: service name (for compose) or stage name/alias (for dockerfile)",
					},
					"file_path": {
						Type:        "string",
						Description: "Optional: Filter to specific file path",
					},
					"direction": {
						Type:        "string",
						Description: "Traversal direction: \"upstream\", \"downstream\", or \"both\" (default: \"both\")",
					},
					"max_depth": {
						Type:        "integer",
						Description: "Maximum traversal depth (default: 10)",
					},
				},
				Required: []string{"type", "name"},
			},
		},
	}
}

// executeTool runs a tool and returns the result.
func (s *Server) executeTool(name string, args map[string]any) (string, bool) {
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
	case "find_dockerfile_instructions":
		return s.toolFindDockerfileInstructions(args)
	case "find_compose_services":
		return s.toolFindComposeServices(args)
	case "get_dockerfile_details":
		return s.toolGetDockerfileDetails(args)
	case "get_docker_dependencies":
		return s.toolGetDockerDependencies(args)
	default:
		return fmt.Sprintf(`{"error": "Unknown tool: %s"}`, name), true
	}
}

// ============================================================================
// Tool Implementations
// ============================================================================

// toolGetIndexInfo returns comprehensive index statistics including symbol type breakdown.
func (s *Server) toolGetIndexInfo() (string, bool) {
	// Check if indexing is complete.
	status := s.statusTracker.GetStatus()
	if status.State != StateReady {
		// Return indexing status instead of full info.
		result := map[string]any{
			"status":   "indexing",
			"state":    status.State.String(),
			"phase":    status.Progress.Phase.String(),
			"message":  status.Progress.Message,
			"progress": status.Progress.OverallProgress,
		}

		if status.State == StateFailed {
			result["error"] = status.Error
		}

		jsonData, _ := json.Marshal(result)
		return string(jsonData), false
	}

	// Count symbols by type and LSP kind.
	symbolsByType := make(map[string]int)
	symbolsByLSPKind := make(map[string]int)

	for _, node := range s.callGraph.Functions {
		symbolsByType[node.Type]++

		// Get LSP kind for this symbol.
		_, kindName := getSymbolKind(node.Type)
		symbolsByLSPKind[kindName]++
	}

	// Count typed parameters.
	parameterCount := len(s.callGraph.Parameters)
	if parameterCount > 0 {
		symbolsByType["parameter"] = parameterCount
		_, paramKindName := getSymbolKind("parameter")
		symbolsByLSPKind[paramKindName] += parameterCount
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

	// Count Docker nodes from CodeGraph.
	dockerfileCount := 0
	composeServiceCount := 0
	if s.codeGraph != nil && s.codeGraph.Nodes != nil {
		for _, node := range s.codeGraph.Nodes {
			if node.Type == "dockerfile_instruction" {
				dockerfileCount++
			} else if node.Type == "compose_service" {
				composeServiceCount++
			}
		}
	}

	// Calculate module statistics.
	moduleStats := make([]map[string]any, 0, len(s.moduleRegistry.Modules))
	totalFunctionsInModules := 0

	for moduleFQN, filePath := range s.moduleRegistry.Modules {
		functionsCount := 0
		for fqn := range s.callGraph.Functions {
			if strings.HasPrefix(fqn, moduleFQN+".") {
				functionsCount++
			}
		}
		totalFunctionsInModules += functionsCount

		moduleStats = append(moduleStats, map[string]any{
			"module_fqn":      moduleFQN,
			"file_path":       filePath,
			"functions_count": functionsCount,
		})
	}

	// Build comprehensive result.
	result := map[string]any{
		"project_path":       s.projectPath,
		"python_version":     s.pythonVersion,
		"indexed_at":         s.indexedAt.Format("2006-01-02T15:04:05Z07:00"),
		"build_time_seconds": s.buildTime.Seconds(),

		// Overall statistics.
		"stats": map[string]any{
			"total_symbols":       len(s.callGraph.Functions),
			"call_edges":          len(s.callGraph.Edges),
			"modules":             len(s.moduleRegistry.Modules),
			"files":               len(s.moduleRegistry.FileToModule),
			"taint_summaries":     len(s.callGraph.Summaries),
			"class_fields":        classFieldsCount,
			"docker_instructions": dockerfileCount,
			"compose_services":    composeServiceCount,
		},

		// Symbol breakdown by Python type (12 types).
		"symbols_by_type": symbolsByType,

		// Symbol breakdown by LSP Symbol Kind (human-readable).
		"symbols_by_lsp_kind": symbolsByLSPKind,

		// Module statistics (top 10 by function count).
		"top_modules": getTopModules(moduleStats, 10),

		// Index health indicators.
		"health": map[string]any{
			"indexed_symbols":              len(s.callGraph.Functions),
			"symbols_with_call_edges":      len(s.callGraph.Edges),
			"modules_indexed":              len(s.moduleRegistry.Modules),
			"average_functions_per_module": float64(totalFunctionsInModules) / float64(maxInt(len(s.moduleRegistry.Modules), 1)),
		},
	}

	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// getTopModules returns the top N modules by function count.
func getTopModules(moduleStats []map[string]any, limit int) []map[string]any {
	// Sort by functions_count descending.
	type moduleStat struct {
		data           map[string]any
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
	result := make([]map[string]any, 0, limit)
	for i := 0; i < len(stats) && i < limit; i++ {
		result = append(result, stats[i].data)
	}

	return result
}

// returnIndexingStatus returns a consistent "indexing" response for all tools.
func (s *Server) returnIndexingStatus() string {
	status := s.statusTracker.GetStatus()
	result := map[string]any{
		"status":   "indexing",
		"message":  "Index is still building. Please wait.",
		"phase":    status.Progress.Phase.String(),
		"progress": status.Progress.OverallProgress * 100, // As percentage
	}
	jsonData, _ := json.Marshal(result)
	return string(jsonData)
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
func (s *Server) toolFindSymbol(args map[string]any) (string, bool) {
	// Check if ready.
	if !s.statusTracker.IsReady() {
		return s.returnIndexingStatus(), false
	}

	name, _ := args["name"].(string)
	singleType, _ := args["type"].(string)
	moduleFilter, _ := args["module"].(string)

	// Handle types parameter (array).
	var typeFilter []string
	if typesParam, ok := args["types"].([]any); ok {
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
		// Python types
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
		"parameter":           true,
		// Go types
		"function_declaration": true,
		"init_function":        true,
		"struct_definition":    true,
		"type_alias":           true,
		"package_variable":     true,
		"variable_assignment":  true,
		"func_literal":         true,
		// Docker types
		"dockerfile_instruction": true,
		"compose_service":        true,
	}

	for _, t := range typeFilter {
		if !validTypes[t] {
			return fmt.Sprintf(`{"error": "Invalid symbol type: %s", "valid_types": ["function_definition","method","constructor","property","special_method","class_definition","interface","enum","dataclass","module_variable","constant","class_field","parameter","function_declaration","init_function","struct_definition","type_alias","package_variable","variable_assignment","func_literal","dockerfile_instruction","compose_service"]}`, t), true
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

	var allMatches []map[string]any

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

			match := map[string]any{
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
							match := map[string]any{
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

	// Search typed parameters from callGraph.Parameters.
	if s.callGraph.Parameters != nil {
		searchParameters := len(typeFilterMap) == 0 || typeFilterMap["parameter"]
		if searchParameters {
			for fqn, param := range s.callGraph.Parameters {
				// Apply module filter if specified.
				if !matchesModuleFilter(fqn, moduleFilter) {
					continue
				}

				// Apply name filter if specified.
				nameMatches := name == ""
				if name != "" {
					nameMatches = param.Name == name || strings.HasSuffix(fqn, "."+name) || fqn == name || strings.Contains(fqn, name)
				}

				if nameMatches {
					symbolKind, symbolKindName := getSymbolKind("parameter")
					match := map[string]any{
						"fqn":              fqn,
						"file":             param.File,
						"line":             param.Line,
						"type":             "parameter",
						"symbol_kind":      symbolKind,
						"symbol_kind_name": symbolKindName,
						"inferred_type":    param.TypeAnnotation,
						"parent_fqn":       param.ParentFQN,
					}
					allMatches = append(allMatches, match)
				}
			}
		}
	}

	// Search codeGraph.Nodes for class definitions, variables, and Docker nodes.
	// These types are stored in the raw AST graph, not in callGraph.Functions.
	missingTypes := map[string]bool{
		"class_definition":       true,
		"interface":              true,
		"enum":                   true,
		"dataclass":              true,
		"module_variable":        true,
		"constant":               true,
		"dockerfile_instruction": true,
		"compose_service":        true,
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
			// Docker nodes: Dockerfile.INSTRUCTION or docker-compose.yml.SERVICE_NAME
			var fqn string
			if node.Type == "dockerfile_instruction" || node.Type == "compose_service" {
				// Docker nodes: use file name as prefix.
				fileName := node.File
				if idx := strings.LastIndex(fileName, "/"); idx != -1 {
					fileName = fileName[idx+1:]
				}
				fqn = fileName + "." + node.Name
			} else {
				modulePath, ok := s.moduleRegistry.FileToModule[node.File]
				if !ok {
					continue
				}
				// Use helper function to build class-qualified FQN for class-level symbols.
				fqn = buildNodeFQN(modulePath, node, classContext)
			}

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

				match := map[string]any{
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

				// Look up inferred type for module variables and constants.
				if (node.Type == "module_variable" || node.Type == "constant") && s.callGraph.TypeEngine != nil {
					if modulePath, ok := s.moduleRegistry.FileToModule[node.File]; ok {
						if varInfo := s.callGraph.TypeEngine.GetModuleVariableType(modulePath, node.Name, node.LineNumber); varInfo != nil {
							match["inferred_type"] = varInfo.TypeFQN
							match["confidence"] = varInfo.Confidence
						}
					}
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
	filtersApplied := map[string]any{}
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

	result := map[string]any{
		"filters_applied": filtersApplied,
		"matches":         matches,
		"pagination":      pageInfo,
	}
	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// toolFindModule searches for Python modules by name.
func (s *Server) toolFindModule(name string) (string, bool) {
	// Check if ready.
	if !s.statusTracker.IsReady() {
		return s.returnIndexingStatus(), false
	}

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

		result := map[string]any{
			"module_fqn":      name,
			"file_path":       filePath,
			"match_type":      "exact",
			"functions_count": functionsCount,
		}
		bytes, _ := json.MarshalIndent(result, "", "  ")
		return string(bytes), false
	}

	// Try partial match.
	var matches []map[string]any
	for moduleFQN, filePath := range s.moduleRegistry.Modules {
		if strings.Contains(moduleFQN, name) {
			// Count functions in this module.
			functionsCount := 0
			for fqn := range s.callGraph.Functions {
				if strings.HasPrefix(fqn, moduleFQN+".") {
					functionsCount++
				}
			}

			matches = append(matches, map[string]any{
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
	result := map[string]any{
		"query":         name,
		"matches":       matches,
		"matches_count": len(matches),
	}
	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// toolListModules lists all modules in the project.
func (s *Server) toolListModules() (string, bool) {
	// Check if ready.
	if !s.statusTracker.IsReady() {
		return s.returnIndexingStatus(), false
	}

	modules := make([]map[string]any, 0, len(s.moduleRegistry.Modules))

	for moduleFQN, filePath := range s.moduleRegistry.Modules {
		// Count functions in this module.
		functionsCount := 0
		for fqn := range s.callGraph.Functions {
			if strings.HasPrefix(fqn, moduleFQN+".") {
				functionsCount++
			}
		}

		modules = append(modules, map[string]any{
			"module_fqn":      moduleFQN,
			"file_path":       filePath,
			"functions_count": functionsCount,
		})
	}

	result := map[string]any{
		"modules":       modules,
		"total_modules": len(modules),
	}
	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// stdlibInfoForFQN returns Go stdlib metadata for a resolved FQN (e.g., "net/http.Get").
// Returns nil when stdlib context is unavailable or the FQN is not a stdlib function.
// Callers should guard on is_stdlib before calling this.
func (s *Server) stdlibInfoForFQN(fqn string) map[string]interface{} {
	if s.goModuleRegistry == nil || s.goModuleRegistry.StdlibLoader == nil {
		return nil
	}
	dotIdx := strings.LastIndex(fqn, ".")
	if dotIdx <= 0 {
		return nil
	}
	importPath := fqn[:dotIdx]
	funcName := fqn[dotIdx+1:]
	if !s.goModuleRegistry.StdlibLoader.ValidateStdlibImport(importPath) {
		return nil
	}
	info := map[string]interface{}{
		"package": importPath,
	}
	fn, err := s.goModuleRegistry.StdlibLoader.GetFunction(importPath, funcName)
	if err == nil && fn != nil {
		if fn.Signature != "" {
			info["signature"] = fn.Signature
		}
		if len(fn.Returns) > 0 {
			returnTypes := make([]string, 0, len(fn.Returns))
			for _, ret := range fn.Returns {
				if ret.Type != "" {
					returnTypes = append(returnTypes, ret.Type)
				}
			}
			if len(returnTypes) > 0 {
				info["return_types"] = returnTypes
			}
		}
	}
	return info
}

// toolGetCallers finds all callers of a function with pagination support.
func (s *Server) toolGetCallers(args map[string]any) (string, bool) {
	// Check if ready.
	if !s.statusTracker.IsReady() {
		return s.returnIndexingStatus(), false
	}

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

	allCallers := make([]map[string]any, 0, len(callerFQNs))
	for _, callerFQN := range callerFQNs {
		callerNode := s.callGraph.Functions[callerFQN]
		if callerNode == nil {
			continue
		}

		caller := map[string]any{
			"fqn":  callerFQN,
			"name": getShortName(callerFQN),
			"file": callerNode.File,
			"line": callerNode.LineNumber,
		}

		// Add return type if available (Python annotations or Go inferred types)
		if returnType := getReturnType(callerNode, callerFQN, s.callGraph); returnType != "" {
			caller["return_type"] = returnType
		}

		// Find the specific call site location.
		for _, cs := range s.callGraph.CallSites[callerFQN] {
			if cs.TargetFQN == targetFQN || cs.Target == getShortName(targetFQN) {
				caller["call_line"] = cs.Location.Line
				caller["call_column"] = cs.Location.Column
				if cs.IsStdlib {
					caller["is_stdlib"] = true
				}
				break
			}
		}

		allCallers = append(allCallers, caller)
	}

	// Apply pagination.
	callers, pageInfo := PaginateSlice(allCallers, pageParams)

	targetInfo := map[string]any{
		"fqn":  targetFQN,
		"name": getShortName(targetFQN),
		"file": targetNode.File,
		"line": targetNode.LineNumber,
	}

	// Add return type if available (Python annotations or Go inferred types)
	if returnType := getReturnType(targetNode, targetFQN, s.callGraph); returnType != "" {
		targetInfo["return_type"] = returnType
	}

	result := map[string]any{
		"target":     targetInfo,
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
func (s *Server) toolGetCallees(args map[string]any) (string, bool) {
	// Check if ready.
	if !s.statusTracker.IsReady() {
		return s.returnIndexingStatus(), false
	}

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

	allCallees := make([]map[string]any, 0, len(callSites))
	resolvedCount := 0
	unresolvedCount := 0

	for _, cs := range callSites {
		callee := map[string]any{
			"target":    cs.Target,
			"call_line": cs.Location.Line,
			"resolved":  cs.Resolved,
			"is_stdlib": cs.IsStdlib,
		}

		if cs.Resolved {
			resolvedCount++
			callee["target_fqn"] = cs.TargetFQN

			// Try to get file info for resolved target.
			if targetNode := s.callGraph.Functions[cs.TargetFQN]; targetNode != nil {
				callee["target_file"] = targetNode.File
				callee["target_line"] = targetNode.LineNumber
			}

			// Add stdlib metadata when available.
			if cs.IsStdlib {
				if info := s.stdlibInfoForFQN(cs.TargetFQN); info != nil {
					callee["stdlib_info"] = info
				}
			}
		} else {
			unresolvedCount++
			if cs.FailureReason != "" {
				callee["failure_reason"] = cs.FailureReason
			}
		}

		// Include type inference info if used.
		if cs.ResolvedViaTypeInference {
			callee["type_inference"] = map[string]any{
				"inferred_type":   cs.InferredType,
				"type_confidence": cs.TypeConfidence,
			}
		}

		allCallees = append(allCallees, callee)
	}

	// Apply pagination.
	callees, pageInfo := PaginateSlice(allCallees, pageParams)

	sourceInfo := map[string]any{
		"fqn":  sourceFQN,
		"name": getShortName(sourceFQN),
		"file": sourceNode.File,
		"line": sourceNode.LineNumber,
	}

	// Add return type if available (Python annotations or Go inferred types)
	if returnType := getReturnType(sourceNode, sourceFQN, s.callGraph); returnType != "" {
		sourceInfo["return_type"] = returnType
	}

	result := map[string]any{
		"source":           sourceInfo,
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
	// Check if ready.
	if !s.statusTracker.IsReady() {
		return s.returnIndexingStatus(), false
	}

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
			callSite := map[string]any{
				"caller_fqn": callerFQN,
				"target":     cs.Target,
				"target_fqn": cs.TargetFQN,
				"location": map[string]any{
					"file":   cs.Location.File,
					"line":   cs.Location.Line,
					"column": cs.Location.Column,
				},
				"resolved": cs.Resolved,
			}

			// Add arguments if available.
			if len(cs.Arguments) > 0 {
				args := make([]map[string]any, len(cs.Arguments))
				for i, arg := range cs.Arguments {
					args[i] = map[string]any{
						"position": arg.Position,
						"value":    arg.Value,
					}
				}
				callSite["arguments"] = args
			}

			// Add resolution info.
			resolution := map[string]interface{}{
				"resolved":  cs.Resolved,
				"is_stdlib": cs.IsStdlib,
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
			if cs.IsStdlib {
				if info := s.stdlibInfoForFQN(cs.TargetFQN); info != nil {
					resolution["stdlib_info"] = info
				}
			}
			callSite["resolution"] = resolution

			result := map[string]any{
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
	// Check if ready.
	if !s.statusTracker.IsReady() {
		return s.returnIndexingStatus(), false
	}

	if importPath == "" {
		return `{"error": "import parameter is required"}`, true
	}

	// Try exact match first.
	if filePath, ok := s.moduleRegistry.Modules[importPath]; ok {
		result := map[string]any{
			"import":       importPath,
			"resolved":     true,
			"file_path":    filePath,
			"module_fqn":   importPath,
			"match_type":   "exact",
			"alternatives": []any{},
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
			result := map[string]any{
				"import":       importPath,
				"resolved":     true,
				"file_path":    filePath,
				"module_fqn":   moduleFQN,
				"match_type":   "short_name",
				"alternatives": []any{},
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
		result := map[string]any{
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
		result := map[string]any{
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

// toolFindDockerfileInstructions searches Dockerfile instructions with semantic filtering.
func (s *Server) toolFindDockerfileInstructions(args map[string]any) (string, bool) {
	// Extract parameters
	instructionType, _ := args["instruction_type"].(string)
	filePath, _ := args["file_path"].(string)
	baseImage, _ := args["base_image"].(string)
	port, _ := args["port"].(float64) // JSON numbers are float64
	user, _ := args["user"].(string)
	limit, _ := args["limit"].(float64)

	// Handle has_digest parameter (optional bool pointer)
	var hasDigest *bool
	if val, ok := args["has_digest"]; ok && val != nil {
		if boolVal, ok := val.(bool); ok {
			hasDigest = &boolVal
		}
	}

	// Default limit
	if limit == 0 {
		limit = 100
	}

	matches := []map[string]any{}

	// Check if codeGraph exists
	if s.codeGraph == nil || s.codeGraph.Nodes == nil {
		result := map[string]any{
			"matches": matches,
			"total":   0,
			"filters_applied": map[string]any{
				"instruction_type": instructionType,
				"file_path":        filePath,
				"base_image":       baseImage,
				"port":             int(port),
				"has_digest":       hasDigest,
				"user":             user,
			},
		}
		bytes, _ := json.Marshal(result)
		return string(bytes), false
	}

	// Iterate through Docker instruction nodes
	for _, node := range s.codeGraph.Nodes {
		if node.Type != "dockerfile_instruction" {
			continue
		}

		// Filter by instruction type
		if instructionType != "" && node.Name != instructionType {
			continue
		}

		// Filter by file path
		if filePath != "" && !strings.Contains(node.File, filePath) {
			continue
		}

		// Get raw content for instruction-specific filtering
		rawContent := ""
		if len(node.MethodArgumentsValue) > 0 {
			rawContent = node.MethodArgumentsValue[0]
		}

		// Apply instruction-specific filters
		if node.Name == "FROM" {
			// Filter by base image
			if baseImage != "" && !strings.Contains(rawContent, baseImage) {
				continue
			}

			// Filter by digest pinning
			if hasDigest != nil {
				nodeHasDigest := strings.Contains(rawContent, "@sha256:")
				if *hasDigest != nodeHasDigest {
					continue
				}
			}
		}

		if node.Name == "USER" && user != "" {
			if !strings.Contains(rawContent, user) {
				continue
			}
		}

		if node.Name == "EXPOSE" && port > 0 {
			portStr := fmt.Sprintf("%d", int(port))
			if !strings.Contains(rawContent, portStr) {
				continue
			}
		}

		// Build rich result
		match := buildDockerInstructionMatch(node, rawContent)
		matches = append(matches, match)

		if len(matches) >= int(limit) {
			break
		}
	}

	result := map[string]any{
		"matches": matches,
		"total":   len(matches),
		"filters_applied": map[string]any{
			"instruction_type": instructionType,
			"file_path":        filePath,
			"base_image":       baseImage,
			"port":             int(port),
			"has_digest":       hasDigest,
			"user":             user,
		},
	}

	bytes, _ := json.Marshal(result)
	return string(bytes), false
}

// buildDockerInstructionMatch builds a rich match result for a Dockerfile instruction.
func buildDockerInstructionMatch(node *graph.Node, rawContent string) map[string]any {
	match := map[string]any{
		"file":        node.File,
		"line":        node.LineNumber,
		"instruction": node.Name,
		"raw_content": rawContent,
	}

	// Extract args (skip first element which is raw_content)
	args := []string{}
	if len(node.MethodArgumentsValue) > 1 {
		args = node.MethodArgumentsValue[1:]
	}
	match["args"] = args

	// Instruction-specific parsing
	switch node.Name {
	case "FROM":
		details := parseFromInstruction(rawContent)
		match["base_image"] = details.BaseImage
		match["tag"] = details.Tag
		match["digest"] = details.Digest
		match["stage_alias"] = details.StageAlias

	case "USER":
		details := parseUserInstruction(rawContent)
		match["user"] = details.User
		match["group"] = details.Group

	case "EXPOSE":
		details := parseExposeInstruction(rawContent)
		match["port"] = details.Port
		match["protocol"] = details.Protocol

	case "WORKDIR":
		match["path"] = extractWorkdirPath(rawContent)

	case "COPY", "ADD":
		details := parseCopyInstruction(rawContent)
		match["source"] = details.Source
		match["destination"] = details.Destination
		match["from_stage"] = details.FromStage
		match["chown"] = details.Chown
	}

	return match
}

// toolFindComposeServices searches docker-compose services with filtering.
func (s *Server) toolFindComposeServices(args map[string]any) (string, bool) {
	// Extract parameters
	serviceName, _ := args["service_name"].(string)
	filePath, _ := args["file_path"].(string)
	exposesPort, _ := args["exposes_port"].(float64)
	hasVolume, _ := args["has_volume"].(string)
	limit, _ := args["limit"].(float64)

	// Handle has_privileged parameter (optional bool pointer)
	var hasPrivileged *bool
	if val, ok := args["has_privileged"]; ok && val != nil {
		if boolVal, ok := val.(bool); ok {
			hasPrivileged = &boolVal
		}
	}

	// Default limit
	if limit == 0 {
		limit = 100
	}

	matches := []map[string]any{}

	// Check if codeGraph exists
	if s.codeGraph == nil || s.codeGraph.Nodes == nil {
		result := map[string]any{
			"matches": matches,
			"total":   0,
			"filters_applied": map[string]any{
				"service_name":   serviceName,
				"file_path":      filePath,
				"has_privileged": hasPrivileged,
				"exposes_port":   int(exposesPort),
				"has_volume":     hasVolume,
			},
		}
		bytes, _ := json.Marshal(result)
		return string(bytes), false
	}

	// Iterate through compose service nodes
	for _, node := range s.codeGraph.Nodes {
		if node.Type != "compose_service" {
			continue
		}

		// Filter by service name
		if serviceName != "" && !strings.Contains(node.Name, serviceName) {
			continue
		}

		// Filter by file path
		if filePath != "" && !strings.Contains(node.File, filePath) {
			continue
		}

		// Parse service properties
		serviceProps := parseComposeServiceProperties(node)

		// Apply filters
		if hasPrivileged != nil && serviceProps.Privileged != *hasPrivileged {
			continue
		}

		if exposesPort > 0 && !serviceProps.exposesPort(int(exposesPort)) {
			continue
		}

		if hasVolume != "" && !serviceProps.hasVolumePath(hasVolume) {
			continue
		}

		// Build rich result
		match := buildComposeServiceMatch(node, serviceProps)
		matches = append(matches, match)

		if len(matches) >= int(limit) {
			break
		}
	}

	result := map[string]any{
		"matches": matches,
		"total":   len(matches),
		"filters_applied": map[string]any{
			"service_name":   serviceName,
			"file_path":      filePath,
			"has_privileged": hasPrivileged,
			"exposes_port":   int(exposesPort),
			"has_volume":     hasVolume,
		},
	}

	bytes, _ := json.Marshal(result)
	return string(bytes), false
}

// parseComposeServiceProperties parses service properties from node.
func parseComposeServiceProperties(node *graph.Node) ComposeServiceProperties {
	props := ComposeServiceProperties{}

	// Parse from MethodArgumentsValue (format: "key=value")
	for _, arg := range node.MethodArgumentsValue {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "image":
			props.Image = value
		case "build":
			props.Build = value
		case "port":
			props.Ports = append(props.Ports, value)
		case "volume":
			props.Volumes = append(props.Volumes, value)
		case "env":
			props.Environment = append(props.Environment, value)
		case "privileged":
			props.Privileged = (value == "true")
		case "network_mode":
			props.NetworkMode = value
		case "cap_add":
			props.CapAdd = append(props.CapAdd, value)
		case "cap_drop":
			props.CapDrop = append(props.CapDrop, value)
		case "security_opt":
			props.SecurityOpt = append(props.SecurityOpt, value)
		}
	}

	return props
}

// buildComposeServiceMatch builds a rich match result for a compose service.
func buildComposeServiceMatch(node *graph.Node, props ComposeServiceProperties) map[string]any {
	match := map[string]any{
		"service_name": node.Name,
		"file":         node.File,
		"line":         node.LineNumber,
		"image":        props.Image,
		"build":        props.Build,
		"ports":        props.Ports,
		"volumes":      props.Volumes,
		"environment":  props.Environment,
		"privileged":   props.Privileged,
		"network_mode": props.NetworkMode,
	}

	return match
}

// toolGetDockerfileDetails returns complete breakdown of a Dockerfile.
func (s *Server) toolGetDockerfileDetails(args map[string]any) (string, bool) {
	filePath, _ := args["file_path"].(string)

	if filePath == "" {
		return `{"error": "file_path parameter is required"}`, true
	}

	// Check if codeGraph exists
	if s.codeGraph == nil || s.codeGraph.Nodes == nil {
		return `{"error": "No code graph available"}`, true
	}

	instructions := []map[string]any{}
	var baseImage string
	stages := []string{}
	isMultiStage := false
	hasUserInstruction := false
	hasHealthcheck := false
	unpinnedImages := 0

	// Collect all instructions for the file
	for _, node := range s.codeGraph.Nodes {
		if node.Type != "dockerfile_instruction" {
			continue
		}

		// Match file path (exact or suffix match)
		if node.File != filePath && !strings.HasSuffix(node.File, filePath) {
			continue
		}

		rawContent := ""
		if len(node.MethodArgumentsValue) > 0 {
			rawContent = node.MethodArgumentsValue[0]
		}

		instruction := buildDockerInstructionMatch(node, rawContent)
		instructions = append(instructions, instruction)

		// Track metadata
		if node.Name == "FROM" {
			fromDetails := parseFromInstruction(rawContent)
			if baseImage == "" {
				baseImage = fromDetails.BaseImage + ":" + fromDetails.Tag
			}
			if fromDetails.StageAlias != "" {
				stages = append(stages, fromDetails.StageAlias)
				isMultiStage = true
			}
			if fromDetails.Digest == "" {
				unpinnedImages++
			}
		}

		if node.Name == "USER" {
			hasUserInstruction = true
		}

		if node.Name == "HEALTHCHECK" {
			hasHealthcheck = true
		}
	}

	if len(instructions) == 0 {
		return fmt.Sprintf(`{"error": "No Dockerfile found at path: %s"}`, filePath), true
	}

	// Build result
	result := map[string]any{
		"file":               filePath,
		"total_instructions": len(instructions),
		"instructions":       instructions,
		"multi_stage": map[string]any{
			"is_multi_stage": isMultiStage,
			"base_image":     baseImage,
			"stages":         stages,
		},
	}

	// Summary
	summary := map[string]any{
		"has_user_instruction": hasUserInstruction,
		"has_healthcheck":      hasHealthcheck,
		"unpinned_images":      unpinnedImages,
	}

	result["summary"] = summary

	bytes, _ := json.Marshal(result)
	return string(bytes), false
}

// toolGetDockerDependencies retrieves dependency information for Docker entities.
func (s *Server) toolGetDockerDependencies(args map[string]any) (string, bool) {
	// Extract parameters
	entityType, _ := args["type"].(string)
	if entityType == "" {
		return `{"error": "type parameter is required"}`, false
	}

	name, _ := args["name"].(string)
	if name == "" {
		return `{"error": "name parameter is required"}`, false
	}

	filePath, _ := args["file_path"].(string)

	direction, _ := args["direction"].(string)
	if direction == "" {
		direction = "both"
	}

	maxDepth := 10
	if depth, ok := args["max_depth"].(float64); ok {
		maxDepth = int(depth)
	}

	// Build dependency graph based on entity type
	var depGraph *dockerpkg.DependencyGraph
	switch entityType {
	case "compose":
		depGraph = dockerpkg.BuildComposeGraph(s.codeGraph)
	case "dockerfile":
		depGraph = dockerpkg.BuildDockerfileGraph(s.codeGraph, filePath)
	default:
		return `{"error": "type must be 'compose' or 'dockerfile'"}`, false
	}

	// Parse direction
	var traversalDirection dockerpkg.TraversalDirection
	switch direction {
	case "upstream":
		traversalDirection = dockerpkg.DirectionUpstream
	case "downstream":
		traversalDirection = dockerpkg.DirectionDownstream
	case "both":
		traversalDirection = dockerpkg.DirectionBoth
	default:
		return `{"error": "direction must be 'upstream', 'downstream', or 'both'"}`, false
	}

	// Perform traversal
	result := dockerpkg.Traverse(depGraph, name, traversalDirection, maxDepth)

	// Add filters applied
	result.FiltersApplied = map[string]any{
		"type":      entityType,
		"name":      name,
		"file_path": filePath,
		"direction": direction,
		"max_depth": maxDepth,
	}

	// Marshal to JSON
	bytes, _ := json.Marshal(result)
	return string(bytes), false
}

// ============================================================================
// Docker Parsing Helpers
// ============================================================================

// FromDetails contains parsed FROM instruction data.
type FromDetails struct {
	BaseImage  string // e.g., "python"
	Tag        string // e.g., "3.11" or "latest" (default if omitted)
	Digest     string // e.g., "sha256:abc123..." (empty if not pinned)
	StageAlias string // e.g., "builder" (from AS clause, empty if single-stage)
}

// UserDetails contains parsed USER instruction data.
type UserDetails struct {
	User  string // Username or UID
	Group string // Group name or GID (empty if not specified)
}

// ExposeDetails contains parsed EXPOSE instruction data.
type ExposeDetails struct {
	Port     int    // Port number
	Protocol string // "tcp" or "udp" (default: "tcp")
}

// CopyDetails contains parsed COPY/ADD instruction data.
type CopyDetails struct {
	Source      string // Source path
	Destination string // Destination path
	FromStage   string // --from flag value (empty if not multi-stage copy)
	Chown       string // --chown flag value (empty if not specified)
}

// ComposeServiceProperties contains parsed compose service data.
type ComposeServiceProperties struct {
	Image       string
	Build       string
	Ports       []string
	Volumes     []string
	Environment []string
	Privileged  bool
	NetworkMode string
	CapAdd      []string
	CapDrop     []string
	SecurityOpt []string
}

// exposesPort checks if the service exposes a specific port.
func (c *ComposeServiceProperties) exposesPort(port int) bool {
	portStr := fmt.Sprintf("%d", port)
	for _, p := range c.Ports {
		if strings.Contains(p, portStr) {
			return true
		}
	}
	return false
}

// hasVolumePath checks if the service has a volume with specific path.
func (c *ComposeServiceProperties) hasVolumePath(path string) bool {
	for _, v := range c.Volumes {
		if strings.Contains(v, path) {
			return true
		}
	}
	return false
}

// parseFromInstruction parses: FROM python:3.11@sha256:abc AS builder.
func parseFromInstruction(rawContent string) FromDetails {
	details := FromDetails{Tag: "latest"} // Default tag

	parts := strings.Fields(rawContent)
	if len(parts) < 2 {
		return details
	}

	image := parts[1]

	// Extract digest (@sha256:...)
	if idx := strings.Index(image, "@"); idx != -1 {
		details.Digest = image[idx+1:]
		image = image[:idx]
	}

	// Extract tag (:3.11)
	if before, after, ok := strings.Cut(image, ":"); ok {
		details.BaseImage = before
		details.Tag = after
	} else {
		details.BaseImage = image
	}

	// Extract stage alias (AS builder)
	for i, part := range parts {
		if strings.ToUpper(part) == "AS" && i+1 < len(parts) {
			details.StageAlias = parts[i+1]
			break
		}
	}

	return details
}

// parseUserInstruction parses: USER appuser:appgroup.
func parseUserInstruction(rawContent string) UserDetails {
	details := UserDetails{}

	parts := strings.Fields(rawContent)
	if len(parts) < 2 {
		return details
	}

	userSpec := parts[1]
	if before, after, ok := strings.Cut(userSpec, ":"); ok {
		details.User = before
		details.Group = after
	} else {
		details.User = userSpec
	}

	return details
}

// parseExposeInstruction parses: EXPOSE 8080/tcp.
func parseExposeInstruction(rawContent string) ExposeDetails {
	details := ExposeDetails{Protocol: "tcp"} // Default protocol

	parts := strings.Fields(rawContent)
	if len(parts) < 2 {
		return details
	}

	portSpec := parts[1]
	if before, after, ok := strings.Cut(portSpec, "/"); ok {
		details.Port, _ = strconv.Atoi(before)
		details.Protocol = after
	} else {
		details.Port, _ = strconv.Atoi(portSpec)
	}

	return details
}

// parseCopyInstruction parses: COPY --from=builder --chown=user:group /src /dst.
func parseCopyInstruction(rawContent string) CopyDetails {
	details := CopyDetails{}

	parts := strings.Fields(rawContent)

	// Parse flags and find source/destination indices
	sourceIdx := 1
	for i := 1; i < len(parts); i++ {
		switch {
		case strings.HasPrefix(parts[i], "--from="):
			details.FromStage = strings.TrimPrefix(parts[i], "--from=")
			sourceIdx = i + 1
		case strings.HasPrefix(parts[i], "--chown="):
			details.Chown = strings.TrimPrefix(parts[i], "--chown=")
			sourceIdx = i + 1
		case !strings.HasPrefix(parts[i], "--"):
			break
		}
	}

	// Extract source and destination
	if sourceIdx < len(parts) {
		details.Source = parts[sourceIdx]
	}
	if sourceIdx+1 < len(parts) {
		details.Destination = parts[sourceIdx+1]
	}

	return details
}

// extractWorkdirPath parses: WORKDIR /app.
func extractWorkdirPath(rawContent string) string {
	parts := strings.Fields(rawContent)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
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

// getReturnType retrieves the return type for a function from either:
//  1. node.ReturnType (annotation-based or inferred for Python/Go)
//  2. callGraph.GoTypeEngine (for Go Phase 2 type tracking)
//
// Returns empty string if no return type is available.
func getReturnType(node *graph.Node, fqn string, callGraph *core.CallGraph) string {
	// Check node.ReturnType first (covers Python and Go with annotations)
	if node.ReturnType != "" {
		return node.ReturnType
	}

	// For Go functions: check GoTypeEngine for inferred types
	if callGraph.GoTypeEngine != nil && strings.HasSuffix(node.File, ".go") {
		if typeInfo, ok := callGraph.GoTypeEngine.GetReturnType(fqn); ok {
			if typeInfo != nil && typeInfo.TypeFQN != "" {
				return typeInfo.TypeFQN
			}
		}
	}

	return ""
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

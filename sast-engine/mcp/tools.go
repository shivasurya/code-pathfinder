package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// getToolDefinitions returns the complete tool schemas.
func (s *Server) getToolDefinitions() []Tool {
	return []Tool{
		{
			Name: "get_index_info",
			Description: `Get statistics about the indexed Python codebase. Use this FIRST to understand the project scope before making other queries.

Returns: project_path, python_version, indexed_at timestamp, build_time, and stats (functions count, call_edges count, modules count, files count, taint_summaries count).

Use when: Starting analysis, understanding project size, or verifying the index is built correctly.`,
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name: "find_symbol",
			Description: `Search for functions, classes, or methods by name. Supports partial matching. Results are paginated.

Returns: List of matches with FQN (fully qualified name like 'myapp.auth.login'), file path, line number, type, and metadata (return_type, parameters, decorators, superclass if available). Includes pagination info.

Use when: Looking for a specific function, exploring what functions exist, or finding where something is defined.

Examples:
- find_symbol("login") - finds all functions containing 'login'
- find_symbol("authenticate_user") - finds exact function
- find_symbol("myapp.auth") - finds all symbols in auth module`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name":   {Type: "string", Description: "Symbol name to find. Can be: short name ('login'), partial name ('auth'), or FQN ('myapp.auth.login')"},
					"limit":  {Type: "integer", Description: "Max results to return (default: 50, max: 500)"},
					"cursor": {Type: "string", Description: "Pagination cursor from previous response"},
				},
				Required: []string{"name"},
			},
		},
		{
			Name: "get_callers",
			Description: `Find all functions that CALL a given function (reverse call graph / incoming edges). Answer: "Who uses this function?" Results are paginated.

Returns: Target function info and list of callers with their FQN, file, line number, and the specific call site location. Includes pagination info.

Use when: Understanding function usage, impact analysis before refactoring, finding entry points, or tracing how data flows INTO a function.

Examples:
- get_callers("sanitize_input") - who calls the sanitize function?
- get_callers("database.execute") - what code runs database queries?`,
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

Use when: Understanding function dependencies, analyzing what a function does, tracing data flow FROM a function, or finding unresolved external calls.

Examples:
- get_callees("process_payment") - what functions does payment processing call?
- get_callees("handle_request") - what are the dependencies of the request handler?`,
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

// toolGetIndexInfo returns index statistics.
func (s *Server) toolGetIndexInfo() (string, bool) {
	result := map[string]interface{}{
		"project_path":       s.projectPath,
		"python_version":     s.pythonVersion,
		"indexed_at":         s.indexedAt.Format("2006-01-02T15:04:05Z07:00"),
		"build_time_seconds": s.buildTime.Seconds(),
		"stats": map[string]int{
			"functions":       len(s.callGraph.Functions),
			"call_edges":      len(s.callGraph.Edges),
			"modules":         len(s.moduleRegistry.Modules),
			"files":           len(s.moduleRegistry.FileToModule),
			"taint_summaries": len(s.callGraph.Summaries),
		},
	}
	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), false
}

// toolFindSymbol finds symbols by name with pagination support.
func (s *Server) toolFindSymbol(args map[string]interface{}) (string, bool) {
	name, _ := args["name"].(string)
	if name == "" {
		return `{"error": "name parameter is required"}`, true
	}

	// Extract pagination params.
	pageParams, err := ExtractPaginationParams(args)
	if err != nil {
		return NewToolError(err.Message, err.Code, err.Data), true
	}

	var allMatches []map[string]interface{}

	for fqn, node := range s.callGraph.Functions {
		shortName := getShortName(fqn)
		if shortName == name || strings.HasSuffix(fqn, "."+name) || fqn == name || strings.Contains(fqn, name) {
			match := map[string]interface{}{
				"fqn":  fqn,
				"file": node.File,
				"line": node.LineNumber,
				"type": node.Type,
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

			allMatches = append(allMatches, match)
		}
	}

	if len(allMatches) == 0 {
		return fmt.Sprintf(`{"error": "Symbol not found: %s", "suggestion": "Try a partial name or check spelling"}`, name), true
	}

	// Apply pagination.
	matches, pageInfo := PaginateSlice(allMatches, pageParams)

	result := map[string]interface{}{
		"query":      name,
		"matches":    matches,
		"pagination": pageInfo,
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

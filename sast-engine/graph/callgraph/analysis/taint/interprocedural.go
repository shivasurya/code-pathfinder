package taint

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/cfg"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// TaintTransferSummary describes how taint flows through a function.
// For each parameter, it records whether that parameter:
//   - leads to a tainted return value
//   - leads to a sink (direct detection within the function)
//
// It also records whether the function itself is a source (returns tainted data
// without needing tainted parameters).
type TaintTransferSummary struct {
	FunctionFQN string

	// ParamNames lists the function's parameter names in order.
	ParamNames []string

	// ParamToReturn maps parameter index to whether taint on that param
	// propagates to the return value.
	ParamToReturn map[int]bool

	// ParamToSink maps parameter index to whether taint on that param
	// reaches a sink within the function.
	ParamToSink map[int]bool

	// IsSource is true if the function returns tainted data (calls a source
	// internally) regardless of parameters.
	IsSource bool

	// IsSanitizer is true if the function sanitizes its input (all params
	// go through a sanitizer before return).
	IsSanitizer bool

	// ReturnTaintedBySource is true if the return value is tainted by an
	// internal source call (e.g., return os.getenv("X")).
	ReturnTaintedBySource bool
}

// BuildTaintTransferSummary constructs a TaintTransferSummary for a function.
// It adds synthetic parameter definitions to the VDG, then checks:
//   - which parameters can reach a return statement (param→return)
//   - which parameters can reach a sink (param→sink)
//   - whether the function itself contains a source that reaches return
func BuildTaintTransferSummary(
	functionFQN string,
	statements []*core.Statement,
	paramNames []string,
	sources []string,
	sinks []string,
	sanitizers []string,
	callGraph *core.CallGraph,
	calleeSummaries map[string]*TaintTransferSummary,
) *TaintTransferSummary {
	summary := &TaintTransferSummary{
		FunctionFQN:   functionFQN,
		ParamNames:    paramNames,
		ParamToReturn: make(map[int]bool),
		ParamToSink:   make(map[int]bool),
	}

	if len(statements) == 0 {
		return summary
	}

	// Build VDG with synthetic parameter defs at line 0
	vdg := NewVarDepGraph()

	// Add synthetic parameter definitions before processing statements.
	// Parameters are defined at line 0 (before any real statement).
	for _, paramName := range paramNames {
		key := nodeKey(paramName, 0)
		vdg.Nodes[key] = &VarDefSite{
			VarName: paramName,
			Line:    0,
		}
		vdg.LatestDef[paramName] = key
	}

	// Build VDG from statements (this will create edges from params to other vars)
	vdg.Build(statements, sources, sinks, sanitizers)

	// Enhance VDG with callee summaries for transitive propagation
	if callGraph != nil && len(calleeSummaries) > 0 {
		EnhanceVDGWithCalleeSummaries(vdg, statements, functionFQN, callGraph, calleeSummaries)
	}

	// Find return statements and source nodes
	var returnStmts []*core.Statement
	for _, stmt := range statements {
		if stmt.Type == core.StatementTypeReturn {
			returnStmts = append(returnStmts, stmt)
		}
	}

	// Collect source and sanitizer node keys
	var sourceKeys []string
	for key, node := range vdg.Nodes {
		if node.IsTaintSrc {
			sourceKeys = append(sourceKeys, key)
		}
	}

	// Check each parameter: can it reach a return statement?
	for i, paramName := range paramNames {
		paramKey := nodeKey(paramName, 0)
		if _, exists := vdg.Nodes[paramKey]; !exists {
			continue
		}

		// Check if param reaches any return statement's used variable
		for _, retStmt := range returnStmts {
			for _, usedVar := range retStmt.Uses {
				defKey, found := vdg.LatestDefAt(usedVar, retStmt.LineNumber)
				if !found {
					continue
				}
				path := vdg.findPath(paramKey, defKey)
				if path != nil && !vdg.pathContainsSanitizer(path) {
					summary.ParamToReturn[i] = true
					break
				}
			}
			if summary.ParamToReturn[i] {
				break
			}
		}

		// Check if param reaches any sink
		for _, stmt := range statements {
			if stmt.Type != core.StatementTypeCall || stmt.Def != "" {
				continue
			}
			if !matchesAnyPattern(stmt.CallTarget, sinks) {
				continue
			}
			for _, usedVar := range stmt.Uses {
				defKey, found := vdg.LatestDefAt(usedVar, stmt.LineNumber)
				if !found {
					continue
				}
				path := vdg.findPath(paramKey, defKey)
				if path != nil && !vdg.pathContainsSanitizer(path) {
					summary.ParamToSink[i] = true
					break
				}
			}
			if summary.ParamToSink[i] {
				break
			}
		}
	}

	// Check transitive ParamToSink: if param reaches a callee arg
	// and callee has ParamToSink for that arg position, propagate.
	if callGraph != nil && len(calleeSummaries) > 0 {
		for i, paramName := range paramNames {
			if summary.ParamToSink[i] {
				continue // already found direct sink
			}
			paramKey := nodeKey(paramName, 0)
			if _, exists := vdg.Nodes[paramKey]; !exists {
				continue
			}

			for _, stmt := range statements {
				if stmt.CallTarget == "" {
					continue
				}
				calleeFQN := resolveCallTarget(stmt.CallTarget, functionFQN, callGraph)
				if calleeFQN == "" {
					continue
				}
				ts, ok := calleeSummaries[calleeFQN]
				if !ok {
					continue
				}

				callSiteArgs := findCallSiteArgs(stmt, functionFQN, callGraph)
				for argIdx, arg := range callSiteArgs {
					if !ts.ParamToSink[argIdx] || !arg.IsVariable {
						continue
					}
					argDefKey, found := vdg.LatestDefAt(arg.Value, stmt.LineNumber)
					if !found {
						continue
					}
					path := vdg.findPath(paramKey, argDefKey)
					if path != nil && !vdg.pathContainsSanitizer(path) {
						summary.ParamToSink[i] = true
						break
					}
				}
				if summary.ParamToSink[i] {
					break
				}
			}
		}
	}

	// Check if function is a sanitizer: all params go through sanitizer before return
	if len(paramNames) > 0 && len(returnStmts) > 0 {
		// Approach 1: Check VDG paths (works when intermediate variables exist)
		allSanitized := true
		hasSanitizerNode := false
		for i, paramName := range paramNames {
			_ = i
			paramKey := nodeKey(paramName, 0)
			if _, exists := vdg.Nodes[paramKey]; !exists {
				continue
			}
			for _, retStmt := range returnStmts {
				for _, usedVar := range retStmt.Uses {
					defKey, found := vdg.LatestDefAt(usedVar, retStmt.LineNumber)
					if !found {
						continue
					}
					path := vdg.findPath(paramKey, defKey)
					if path != nil && !vdg.pathContainsSanitizer(path) {
						allSanitized = false
					}
				}
			}
		}
		if allSanitized {
			for _, node := range vdg.Nodes {
				if node.IsSanitized {
					hasSanitizerNode = true
					break
				}
			}
			summary.IsSanitizer = hasSanitizerNode
		}

		// Approach 2: Direct return of sanitizer call (e.g., return html.escape(data))
		// All return statements must call a sanitizer for this to apply.
		if !summary.IsSanitizer && len(returnStmts) > 0 {
			allReturnsSanitized := true
			for _, retStmt := range returnStmts {
				if retStmt.CallTarget == "" || !matchesAnyPattern(retStmt.CallTarget, sanitizers) {
					allReturnsSanitized = false
					break
				}
			}
			if allReturnsSanitized {
				summary.IsSanitizer = true
			}
		}
	}

	// Check if function returns source-tainted data (internal source → return)
	for _, srcKey := range sourceKeys {
		for _, retStmt := range returnStmts {
			for _, usedVar := range retStmt.Uses {
				defKey, found := vdg.LatestDefAt(usedVar, retStmt.LineNumber)
				if !found {
					continue
				}
				path := vdg.findPath(srcKey, defKey)
				if path != nil && !vdg.pathContainsSanitizer(path) {
					summary.IsSource = true
					summary.ReturnTaintedBySource = true
					break
				}
			}
			if summary.IsSource {
				break
			}
		}
		if summary.IsSource {
			break
		}
	}

	// Handle direct return of source call: `return os.getenv("X")`
	// These have type=return and CallTarget matching a source pattern,
	// but no Def so the VDG has no node for them.
	if !summary.ReturnTaintedBySource {
		for _, retStmt := range returnStmts {
			if retStmt.CallTarget != "" && matchesAnyPattern(retStmt.CallTarget, sources) {
				summary.IsSource = true
				summary.ReturnTaintedBySource = true
				break
			}
		}
	}

	// Handle direct return of callee whose summary says ReturnTaintedBySource:
	// `return get_user_input()` where get_user_input is not a source pattern itself,
	// but its transfer summary says it returns tainted data.
	// Same pattern: type=return, def="", so VDG has no node.
	if !summary.ReturnTaintedBySource && callGraph != nil && len(calleeSummaries) > 0 {
		for _, retStmt := range returnStmts {
			if retStmt.CallTarget == "" {
				continue
			}
			calleeFQN := resolveCallTarget(retStmt.CallTarget, functionFQN, callGraph)
			if calleeFQN == "" {
				continue
			}
			if ts, ok := calleeSummaries[calleeFQN]; ok && ts.ReturnTaintedBySource {
				summary.IsSource = true
				summary.ReturnTaintedBySource = true
				break
			}
		}
	}

	// Handle direct return of callee whose summary says IsSanitizer:
	// `return sanitize(data)` where sanitize's summary says IsSanitizer=true.
	if !summary.IsSanitizer && callGraph != nil && len(calleeSummaries) > 0 {
		if len(returnStmts) > 0 {
			allReturnsSanitized := true
			for _, retStmt := range returnStmts {
				if retStmt.CallTarget == "" {
					allReturnsSanitized = false
					break
				}
				calleeFQN := resolveCallTarget(retStmt.CallTarget, functionFQN, callGraph)
				if calleeFQN == "" {
					// Check if it matches a direct sanitizer pattern (already handled above)
					if !matchesAnyPattern(retStmt.CallTarget, sanitizers) {
						allReturnsSanitized = false
						break
					}
					continue
				}
				if ts, ok := calleeSummaries[calleeFQN]; ok && ts.IsSanitizer {
					continue
				}
				allReturnsSanitized = false
				break
			}
			if allReturnsSanitized {
				summary.IsSanitizer = true
			}
		}
	}

	return summary
}

// BuildTaintTransferSummaryWithCFG builds a transfer summary using CFG-flattened statements.
func BuildTaintTransferSummaryWithCFG(
	functionFQN string,
	cfGraph *cfg.ControlFlowGraph,
	blockStmts cfg.BlockStatements,
	paramNames []string,
	sources []string,
	sinks []string,
	sanitizers []string,
	callGraph *core.CallGraph,
	calleeSummaries map[string]*TaintTransferSummary,
) *TaintTransferSummary {
	allStatements := FlattenBlockStatements(cfGraph, blockStmts)
	return BuildTaintTransferSummary(functionFQN, allStatements, paramNames, sources, sinks, sanitizers, callGraph, calleeSummaries)
}

// AnalyzeInterProcedural performs inter-procedural taint analysis using
// taint transfer summaries. For each function in the call graph, it:
//  1. Builds a TaintTransferSummary
//  2. At call sites like `y = callee(x)`, checks if callee's summary says
//     the return is tainted when the argument is tainted
//  3. Propagates taint across function boundaries
//
// Parameters:
//   - callerFQN: the function being analyzed
//   - statements: the caller's statements
//   - sources/sinks/sanitizers: pattern lists
//   - callGraph: for looking up callee info
//   - transferSummaries: pre-computed summaries for all functions
func AnalyzeInterProcedural(
	callerFQN string,
	statements []*core.Statement,
	sources []string,
	sinks []string,
	sanitizers []string,
	callGraph *core.CallGraph,
	transferSummaries map[string]*TaintTransferSummary,
) *core.TaintSummary {
	result := core.NewTaintSummary(callerFQN)

	// Build VDG for the caller
	vdg := NewVarDepGraph()
	vdg.Build(statements, sources, sinks, sanitizers)

	// Enhance VDG with inter-procedural taint propagation
	EnhanceVDGWithCalleeSummaries(vdg, statements, callerFQN, callGraph, transferSummaries)

	// Now find taint flows with the enhanced VDG (direct sinks)
	detections := vdg.FindTaintFlows(statements, sinks)

	// Also find inter-procedural sinks: calls to functions whose transfer
	// summary says ParamToSink (e.g., dangerous_eval wraps eval internally)
	for _, stmt := range statements {
		if stmt.CallTarget == "" {
			continue
		}

		calleeFQN := resolveCallTarget(stmt.CallTarget, callerFQN, callGraph)
		if calleeFQN == "" {
			continue
		}
		ts, ok := transferSummaries[calleeFQN]
		if !ok {
			continue
		}

		// Check if any argument to this indirect sink is tainted
		callSiteArgs := findCallSiteArgs(stmt, callerFQN, callGraph)
		for paramIdx, arg := range callSiteArgs {
			if !ts.ParamToSink[paramIdx] || !arg.IsVariable {
				continue
			}

			argDefKey, found := vdg.LatestDefAt(arg.Value, stmt.LineNumber)
			if !found {
				continue
			}

			// Check if this argument is tainted (reachable from a source)
			for srcKey, srcNode := range vdg.Nodes {
				if !srcNode.IsTaintSrc {
					continue
				}
				path := vdg.findPath(srcKey, argDefKey)
				if path != nil && !vdg.pathContainsSanitizer(path) {
					detections = append(detections, TaintDetection{
						SourceLine:      srcNode.Line,
						SourceVar:       srcNode.VarName,
						SinkLine:        stmt.LineNumber,
						SinkCall:        calleeFQN,
						PropagationPath: vdg.pathToVarNames(path),
						Confidence:      0.9,
					})
					break
				}
			}
		}
	}

	for _, det := range detections {
		taintInfo := &core.TaintInfo{
			SourceLine:      det.SourceLine,
			SourceVar:       det.SourceVar,
			SinkLine:        det.SinkLine,
			SinkCall:        det.SinkCall,
			PropagationPath: det.PropagationPath,
			Confidence:      det.Confidence,
		}
		result.AddDetection(taintInfo)
		result.AddTaintedVar(det.SourceVar, &core.TaintInfo{
			SourceLine: det.SourceLine,
			SourceVar:  det.SourceVar,
			Confidence: det.Confidence,
		})
	}

	return result
}

// EnhanceVDGWithCalleeSummaries enhances a VDG with inter-procedural taint info.
// For each assignment `y = callee(x)`, checks callee's transfer summary to determine:
//   - If callee returns tainted data (ReturnTaintedBySource) → mark y as source
//   - If callee is a sanitizer → mark y as sanitized
//   - If tainted arg flows to callee return (ParamToReturn) → mark y as source
func EnhanceVDGWithCalleeSummaries(
	vdg *VarDepGraph,
	statements []*core.Statement,
	callerFQN string,
	callGraph *core.CallGraph,
	transferSummaries map[string]*TaintTransferSummary,
) {
	for _, stmt := range statements {
		if stmt.Def == "" || stmt.CallTarget == "" {
			continue
		}

		calleeFQN := resolveCallTarget(stmt.CallTarget, callerFQN, callGraph)
		if calleeFQN == "" {
			continue
		}

		transferSummary, ok := transferSummaries[calleeFQN]
		if !ok {
			continue
		}

		defKey := nodeKey(stmt.Def, stmt.LineNumber)
		node, exists := vdg.Nodes[defKey]
		if !exists {
			continue
		}

		// Case 1: Callee is itself a source (returns tainted data)
		if transferSummary.ReturnTaintedBySource {
			node.IsTaintSrc = true
		}

		// Case 2: Callee is a sanitizer
		if transferSummary.IsSanitizer {
			node.IsSanitized = true
		}

		// Case 3: Callee propagates taint from param to return
		callSiteArgs := findCallSiteArgs(stmt, callerFQN, callGraph)
		for paramIdx, arg := range callSiteArgs {
			if !arg.IsVariable {
				continue
			}
			argDefKey, found := vdg.LatestDefAt(arg.Value, stmt.LineNumber)
			if !found {
				continue
			}

			argTainted := false
			for srcKey, srcNode := range vdg.Nodes {
				if srcNode.IsTaintSrc {
					path := vdg.findPath(srcKey, argDefKey)
					if path != nil && !vdg.pathContainsSanitizer(path) {
						argTainted = true
						break
					}
				}
			}

			if argTainted && transferSummary.ParamToReturn[paramIdx] {
				node.IsTaintSrc = true
			}
		}
	}
}

// findCallSiteArgs finds the matching call site for a statement and returns its arguments.
// This provides proper argument-to-parameter mapping (stmt.Uses includes function name parts).
func findCallSiteArgs(stmt *core.Statement, callerFQN string, callGraph *core.CallGraph) []core.Argument {
	callSites, ok := callGraph.CallSites[callerFQN]
	if !ok {
		return nil
	}

	// Match by line number and target name
	for _, cs := range callSites {
		if cs.Location.Line == int(stmt.LineNumber) {
			return cs.Arguments
		}
	}
	return nil
}

// resolveCallTarget resolves a call target string to a function FQN.
// It checks the call graph's call sites for the caller function to find
// matching resolved targets.
func resolveCallTarget(callTarget string, callerFQN string, callGraph *core.CallGraph) string {
	// Check call sites for this caller
	callSites, ok := callGraph.CallSites[callerFQN]
	if !ok {
		return ""
	}

	for _, cs := range callSites {
		// Match by target name
		if cs.Target == callTarget || cs.TargetFQN == callTarget {
			if cs.TargetFQN != "" {
				return cs.TargetFQN
			}
			return cs.Target
		}
		// Also try matching the stripped call target (remove parens)
		stripped := callTarget
		if idx := len(stripped) - 1; idx >= 0 {
			for i := 0; i < len(stripped); i++ {
				if stripped[i] == '(' {
					stripped = stripped[:i]
					break
				}
			}
		}
		if cs.Target == stripped || cs.TargetFQN == stripped {
			if cs.TargetFQN != "" {
				return cs.TargetFQN
			}
			return cs.Target
		}
	}

	return ""
}

// Package mcp provides instance resolution tools for the MCP server.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution/strategies"
	sitter "github.com/smacker/go-tree-sitter"
	python "github.com/smacker/go-tree-sitter/python"
)

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// ResolveInstanceCallRequest represents the input for resolve_instance_call.
type ResolveInstanceCallRequest struct {
	Expression string               `json:"expression"`
	FilePath   string               `json:"file_path"` //nolint:tagliatelle // MCP protocol uses snake_case
	Line       int                  `json:"line"`
	Column     int                  `json:"column"`
	Context    *InstanceCallContext `json:"context,omitempty"`
}

// InstanceCallContext provides additional context for resolution.
type InstanceCallContext struct {
	SelfType  string            `json:"self_type,omitempty"` //nolint:tagliatelle // MCP protocol uses snake_case
	Variables map[string]string `json:"variables,omitempty"`
}

// ResolveInstanceCallResponse represents the output for resolve_instance_call.
type ResolveInstanceCallResponse struct {
	Success      bool                `json:"success"`
	ResolvedType string              `json:"resolved_type,omitempty"` //nolint:tagliatelle // MCP protocol uses snake_case
	Method       string              `json:"method,omitempty"`
	CanonicalFQN string              `json:"canonical_fqn,omitempty"` //nolint:tagliatelle // MCP protocol uses snake_case
	Definition   *DefinitionLocation `json:"definition,omitempty"`
	Confidence   float64             `json:"confidence"`
	Error        string              `json:"error,omitempty"`
}

// DefinitionLocation represents where a method is defined.
type DefinitionLocation struct {
	FilePath  string `json:"file_path"` //nolint:tagliatelle // MCP protocol uses snake_case
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Signature string `json:"signature,omitempty"`
	Docstring string `json:"docstring,omitempty"`
}

// GetInstanceTypeRequest represents the input for get_instance_type.
type GetInstanceTypeRequest struct {
	Variable string `json:"variable"`
	FilePath string `json:"file_path"` //nolint:tagliatelle // MCP protocol uses snake_case
	Line     int    `json:"line"`
}

// GetInstanceTypeResponse represents the output for get_instance_type.
type GetInstanceTypeResponse struct {
	Success    bool    `json:"success"`
	TypeFQN    string  `json:"type_fqn,omitempty"` //nolint:tagliatelle // MCP protocol uses snake_case
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source,omitempty"`
	Error      string  `json:"error,omitempty"`
}

// =============================================================================
// TOOL HANDLER
// =============================================================================

// InstanceToolHandler handles instance resolution MCP tools.
type InstanceToolHandler struct {
	inferencer   *resolution.BidirectionalInferencer
	attrRegistry strategies.AttributeRegistryInterface
	callGraph    *core.CallGraph
}

// NewInstanceToolHandler creates a new InstanceToolHandler.
func NewInstanceToolHandler(
	inferencer *resolution.BidirectionalInferencer,
	attrRegistry strategies.AttributeRegistryInterface,
	callGraph *core.CallGraph,
) *InstanceToolHandler {
	return &InstanceToolHandler{
		inferencer:   inferencer,
		attrRegistry: attrRegistry,
		callGraph:    callGraph,
	}
}

// HandleResolveInstanceCall handles the resolve_instance_call tool.
func (h *InstanceToolHandler) HandleResolveInstanceCall(args json.RawMessage) (*ResolveInstanceCallResponse, error) {
	var req ResolveInstanceCallRequest
	if err := json.Unmarshal(args, &req); err != nil {
		return &ResolveInstanceCallResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid request: %v", err),
		}, nil
	}

	// Parse the expression
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(req.Expression))
	if err != nil {
		return &ResolveInstanceCallResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to parse expression: %v", err),
		}, nil
	}

	rootNode := tree.RootNode()
	if rootNode.ChildCount() == 0 {
		return &ResolveInstanceCallResponse{
			Success: false,
			Error:   "empty expression",
		}, nil
	}

	// Get the expression node (skip expression_statement wrapper)
	exprNode := rootNode.Child(0)
	if exprNode.Type() == "expression_statement" && exprNode.ChildCount() > 0 {
		exprNode = exprNode.Child(0)
	}

	// Set up resolution context
	store := resolution.NewTypeStore()

	// Add context variables
	if req.Context != nil {
		for varName, typeFQN := range req.Context.Variables {
			store.Set(varName, core.NewConcreteType(typeFQN, 0.95),
				core.ConfidenceAssignment, req.FilePath, req.Line, req.Column)
		}
	}

	// Set up self type
	var selfType core.Type
	var classFQN string
	if req.Context != nil && req.Context.SelfType != "" {
		selfType = core.NewConcreteType(req.Context.SelfType, 0.95)
		classFQN = req.Context.SelfType
	}

	// Resolve the expression
	resolvedType, confidence := h.inferencer.InferType(
		exprNode,
		store,
		[]byte(req.Expression),
		req.FilePath,
		selfType,
		classFQN,
		"",
	)

	if core.IsAnyType(resolvedType) {
		return &ResolveInstanceCallResponse{
			Success:    false,
			Confidence: confidence,
			Error:      "could not resolve expression type",
		}, nil
	}

	// Extract method name from expression
	methodName := h.extractMethodName(exprNode, []byte(req.Expression))

	// Build canonical FQN
	canonicalFQN := resolvedType.FQN()
	if methodName != "" {
		canonicalFQN = canonicalFQN + "." + methodName
	}

	// Look up definition location
	definition := h.lookupDefinition(resolvedType.FQN(), methodName)

	return &ResolveInstanceCallResponse{
		Success:      true,
		ResolvedType: resolvedType.FQN(),
		Method:       methodName,
		CanonicalFQN: canonicalFQN,
		Definition:   definition,
		Confidence:   confidence,
	}, nil
}

// extractMethodName extracts the method name from a call expression.
func (h *InstanceToolHandler) extractMethodName(node *sitter.Node, source []byte) string {
	if node == nil || node.Type() != "call" {
		return ""
	}

	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return ""
	}

	if funcNode.Type() == "attribute" {
		attrNode := funcNode.ChildByFieldName("attribute")
		if attrNode != nil {
			return string(source[attrNode.StartByte():attrNode.EndByte()])
		}
	}

	return ""
}

// lookupDefinition finds where a method is defined.
func (h *InstanceToolHandler) lookupDefinition(classFQN, methodName string) *DefinitionLocation {
	if h.attrRegistry == nil {
		return nil
	}

	classAttrs := h.attrRegistry.GetClassAttributes(classFQN)
	if classAttrs == nil {
		return nil
	}

	// Check if method exists
	methodFQN := classFQN + "." + methodName
	if slices.Contains(classAttrs.Methods, methodFQN) {
		// Return basic location from class (detailed location would require method registry)
		return &DefinitionLocation{
			FilePath: classAttrs.FilePath,
			Line:     0, // Would need method-level tracking
			Column:   0,
		}
	}

	return nil
}

// HandleGetInstanceType handles the get_instance_type tool.
func (h *InstanceToolHandler) HandleGetInstanceType(args json.RawMessage) (*GetInstanceTypeResponse, error) {
	var req GetInstanceTypeRequest
	if err := json.Unmarshal(args, &req); err != nil {
		return &GetInstanceTypeResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid request: %v", err),
		}, nil
	}

	// Create a simple expression from the variable name
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(req.Variable))
	if err != nil {
		return &GetInstanceTypeResponse{ //nolint:nilerr // MCP pattern: errors embedded in response
			Success: false,
			Error:   "failed to parse variable",
		}, nil
	}

	store := resolution.NewTypeStore()

	resolvedType, confidence := h.inferencer.InferType(
		tree.RootNode().Child(0).Child(0),
		store,
		[]byte(req.Variable),
		req.FilePath,
		nil,
		"",
		"",
	)

	if core.IsAnyType(resolvedType) {
		return &GetInstanceTypeResponse{
			Success: false,
			Error:   "variable type unknown",
		}, nil
	}

	return &GetInstanceTypeResponse{
		Success:    true,
		TypeFQN:    resolvedType.FQN(),
		Confidence: confidence,
		Source:     "type_inference",
	}, nil
}

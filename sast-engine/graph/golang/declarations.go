package golang

import sitter "github.com/smacker/go-tree-sitter"

// FunctionInfo holds extracted information from a Go function declaration.
// Used by the dispatcher in parser_golang.go to create graph.Node instances.
type FunctionInfo struct {
	Name       string
	Params     GoParams
	ReturnType string
	Visibility string // "public" or "private"
	IsInit     bool
	LineNumber uint32
}

// MethodInfo holds extracted information from a Go method declaration.
type MethodInfo struct {
	FunctionInfo
	ReceiverType string // bare type name, e.g., "Server" (pointer stripped)
}

// ParseFunctionDeclaration extracts function information from a Go function_declaration node.
//
// Example:
//
//	func Foo(a, b int) string {} → Name="Foo", Params, ReturnType="string", Visibility="public"
//	func init() {}              → Name="init", IsInit=true
func ParseFunctionDeclaration(node *sitter.Node, sourceCode []byte) *FunctionInfo {
	name := ""
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		name = nameNode.Content(sourceCode)
	}

	paramList := node.ChildByFieldName("parameters")
	params := ExtractParameters(paramList, sourceCode)

	resultNode := node.ChildByFieldName("result")
	returnType := ExtractReturnType(resultNode, sourceCode)

	return &FunctionInfo{
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Visibility: DetermineVisibility(name),
		IsInit:     IsInitFunction(name),
		LineNumber: node.StartPoint().Row + 1,
	}
}

// ParseMethodDeclaration extracts method information from a Go method_declaration node.
//
// Example:
//
//	func (s *Server) Start() error {} → Name="Start", ReceiverType="Server"
//	func (s Server) String() string {} → Name="String", ReceiverType="Server"
func ParseMethodDeclaration(node *sitter.Node, sourceCode []byte) *MethodInfo {
	name := ""
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		name = nameNode.Content(sourceCode)
	}

	paramList := node.ChildByFieldName("parameters")
	params := ExtractParameters(paramList, sourceCode)

	resultNode := node.ChildByFieldName("result")
	returnType := ExtractReturnType(resultNode, sourceCode)

	receiverNode := node.ChildByFieldName("receiver")
	receiverType := ExtractReceiverType(receiverNode, sourceCode)

	return &MethodInfo{
		FunctionInfo: FunctionInfo{
			Name:       name,
			Params:     params,
			ReturnType: returnType,
			Visibility: DetermineVisibility(name),
			IsInit:     false, // methods are never init functions
			LineNumber: node.StartPoint().Row + 1,
		},
		ReceiverType: receiverType,
	}
}

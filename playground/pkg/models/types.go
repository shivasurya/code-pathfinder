package models

// ParseRequest represents the input for AST parsing
type ParseRequest struct {
	JavaSource string `json:"code"`
}

// ParseResponse represents the output from AST parsing
type ParseResponse struct {
	AST *ASTNode `json:"ast"`
}

// ASTNode represents a node in the Abstract Syntax Tree
type ASTNode struct {
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Line        int       `json:"line"`
	Modifier    string    `json:"modifier,omitempty"`
	Value       string    `json:"value,omitempty"`
	Package     string    `json:"package,omitempty"`
	Imports     []string  `json:"imports,omitempty"`
	SuperClass  string    `json:"superClass,omitempty"`
	DataType    string    `json:"dataType,omitempty"`
	ReturnType  string    `json:"returnType,omitempty"`
	Arguments   []string  `json:"arguments,omitempty"`
	Children    []ASTNode `json:"children"`
}

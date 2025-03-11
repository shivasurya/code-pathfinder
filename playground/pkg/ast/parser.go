package ast

import (
	"fmt"

	"github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/shivasurya/code-pathfinder/playground/pkg/models"
)

// ParseJavaSource parses Java source code into an AST using tree-sitter
func ParseJavaSource(sourceCode string) (*models.ASTNode, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	sourceBytes := []byte(sourceCode)
	tree := parser.Parse(nil, sourceBytes)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse source code")
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("invalid AST: no root node")
	}

	ast := buildASTFromTreeSitter(root, sourceBytes)
	if ast == nil {
		return nil, fmt.Errorf("failed to build AST")
	}

	return ast, nil
}

// buildASTFromTreeSitter converts a tree-sitter AST into our AST structure
func buildASTFromTreeSitter(node *sitter.Node, sourceBytes []byte) *models.ASTNode {
	if node == nil {
		return nil
	}

	// Create AST node
	astNode := &models.ASTNode{
		Type:     node.Type(),
		Name:     node.Type(),
		Code:     getNodeText(node, sourceBytes),
		Line:     int(node.StartPoint().Row + 1),
		Children: make([]models.ASTNode, 0),
	}

	// Process specific node types
	switch node.Type() {
	case "package_declaration":
		astNode.Type = "PackageDeclaration"
		astNode.Package = getNodeText(node, sourceBytes)

	case "import_declaration":
		astNode.Type = "ImportDeclaration"
		astNode.Imports = []string{getNodeText(node, sourceBytes)}

	case "class_declaration":
		astNode.Type = "ClassDeclaration"
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			astNode.Name = getNodeText(nameNode, sourceBytes)
		}
		if superNode := node.ChildByFieldName("superclass"); superNode != nil {
			astNode.SuperClass = getNodeText(superNode, sourceBytes)
		}
		astNode.Modifier = getModifiers(node, sourceBytes)

	case "method_declaration":
		astNode.Type = "MethodDeclaration"
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			astNode.Name = getNodeText(nameNode, sourceBytes)
		}
		if typeNode := node.ChildByFieldName("type"); typeNode != nil {
			astNode.ReturnType = getNodeText(typeNode, sourceBytes)
		}
		astNode.Arguments = getMethodParameters(node, sourceBytes)
		astNode.Modifier = getModifiers(node, sourceBytes)

	case "field_declaration":
		astNode.Type = "FieldDeclaration"
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			astNode.Name = getNodeText(nameNode, sourceBytes)
		}
		if typeNode := node.ChildByFieldName("type"); typeNode != nil {
			astNode.DataType = getNodeText(typeNode, sourceBytes)
		}
		astNode.Modifier = getModifiers(node, sourceBytes)
		astNode.Value = getInitializer(node, sourceBytes)

	case "variable_declaration":
		astNode.Type = "VariableDeclaration"
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			astNode.Name = getNodeText(nameNode, sourceBytes)
		}
		if typeNode := node.ChildByFieldName("type"); typeNode != nil {
			astNode.DataType = getNodeText(typeNode, sourceBytes)
		}
		astNode.Value = getInitializer(node, sourceBytes)
	}

	// Process child nodes
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if child := node.NamedChild(i); child != nil {
			if childNode := buildASTFromTreeSitter(child, sourceBytes); childNode != nil {
				astNode.Children = append(astNode.Children, *childNode)
			}
		}
	}

	return astNode
}

// Helper functions for tree-sitter AST processing
func getNodeText(node *sitter.Node, sourceBytes []byte) string {
	if node == nil {
		return ""
	}
	return string(node.Content(sourceBytes))
}

func getModifiers(node *sitter.Node, sourceBytes []byte) string {
	if modNode := node.ChildByFieldName("modifiers"); modNode != nil {
		return getNodeText(modNode, sourceBytes)
	}
	return ""
}

func getMethodParameters(node *sitter.Node, sourceBytes []byte) []string {
	var params []string
	if paramsList := node.ChildByFieldName("parameters"); paramsList != nil {
		for i := 0; i < int(paramsList.NamedChildCount()); i++ {
			if param := paramsList.NamedChild(i); param != nil {
				params = append(params, getNodeText(param, sourceBytes))
			}
		}
	}
	return params
}

func getInitializer(node *sitter.Node, sourceBytes []byte) string {
	if initNode := node.ChildByFieldName("initializer"); initNode != nil {
		return getNodeText(initNode, sourceBytes)
	}
	return ""
}

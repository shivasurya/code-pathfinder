package java

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseClass(node *sitter.Node, sourceCode []byte, file string) *model.Class {
	var classDeclaration *model.Class
	className := node.ChildByFieldName("name").Content(sourceCode)
	packageName := ""
	accessModifier := ""
	superClass := ""
	annotationMarkers := []string{}
	implementedInterface := []string{}
	classDeclaration.ClassOrInterface.QualifiedName = className
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "modifiers" {
			accessModifier = child.Content(sourceCode)
			for j := 0; j < int(child.ChildCount()); j++ {
				if child.Child(j).Type() == "marker_annotation" {
					annotationMarkers = append(annotationMarkers, child.Child(j).Content(sourceCode))
				}
			}
		}
		if child.Type() == "superclass" {
			for j := 0; j < int(child.ChildCount()); j++ {
				if child.Child(j).Type() == "type_identifier" {
					superClass = child.Child(j).Content(sourceCode)
				}
			}
		}
		if child.Type() == "super_interfaces" {
			for j := 0; j < int(child.ChildCount()); j++ {
				// typelist node and then iterate through type_identifier node
				typeList := child.Child(j)
				for k := 0; k < int(typeList.ChildCount()); k++ {
					implementedInterface = append(implementedInterface, typeList.Child(k).Content(sourceCode))
				}
			}
		}
	}
	classDeclaration.Annotations = annotationMarkers
	classDeclaration.ClassOrInterface.Package = packageName
	classDeclaration.SourceFile = file
	classDeclaration.Modifiers = []string{ExtractVisibilityModifier(accessModifier)}
	classDeclaration.SuperTypes = []string{superClass}

	return classDeclaration
}

func ParseObjectCreationExpr(node *sitter.Node, sourceCode []byte, file string) *model.ClassInstanceExpr {
	className := ""
	classInstanceExpression := &model.ClassInstanceExpr{
		ClassName: "",
		Args:      []*model.Expr{},
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_identifier" || child.Type() == "scoped_type_identifier" {
			className = child.Content(sourceCode)
			classInstanceExpression.ClassName = className
		}
		if child.Type() == "argument_list" {
			classInstanceExpression.Args = []*model.Expr{}
			for j := 0; j < int(child.ChildCount()); j++ {
				argType := child.Child(j).Type()
				argumentStopWords := map[string]bool{
					"(": true,
					")": true,
					"{": true,
					"}": true,
					"[": true,
					"]": true,
					",": true,
				}
				if !argumentStopWords[argType] {
					argument := &model.Expr{}
					argument.Type = child.Child(j).Type()
					argument.NodeString = child.Child(j).Content(sourceCode)
					classInstanceExpression.Args = append(classInstanceExpression.Args, argument)
				}
			}
		}
	}

	return classInstanceExpression
}

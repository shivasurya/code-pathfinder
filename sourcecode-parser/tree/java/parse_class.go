package java

import (
	"strconv"
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	util "github.com/shivasurya/code-pathfinder/sourcecode-parser/util"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseClass(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	var javadoc *model.Node
	var classDeclaration *model.Class
	if node.PrevSibling() != nil && node.PrevSibling().Type() == "block_comment" {
		commentContent := node.PrevSibling().Content(sourceCode)
		if strings.HasPrefix(commentContent, "/*") {
			javadoc = ParseJavadocTags(node, sourceCode, file)
		}
	}
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

	classNode := &model.Node{
		ID:               util.GenerateMethodID(className, []string{}, file),
		Type:             "class_declaration",
		Name:             className,
		CodeSnippet:      node.Content(sourceCode),
		LineNumber:       node.StartPoint().Row + 1,
		PackageName:      packageName,
		Modifier:         ExtractVisibilityModifier(accessModifier),
		SuperClass:       superClass,
		Interface:        implementedInterface,
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		JavaDoc:          javadoc.JavaDoc,
		Annotation:       annotationMarkers,
	}

	return classNode
}

func ParseObjectCreationExpr(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	className := ""
	classInstanceExpression := model.ClassInstanceExpr{
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

	objectNode := &model.Node{
		ID:                util.GenerateMethodID(className, []string{strconv.Itoa(int(node.StartPoint().Row + 1))}, file),
		Type:              "ClassInstanceExpr",
		Name:              className,
		CodeSnippet:       node.Content(sourceCode),
		LineNumber:        node.StartPoint().Row + 1,
		File:              file,
		IsJavaSourceFile:  IsJavaSourceFile(file),
		ClassInstanceExpr: &classInstanceExpression,
	}
	return objectNode
}

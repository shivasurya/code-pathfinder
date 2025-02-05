package java

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	util "github.com/shivasurya/code-pathfinder/sourcecode-parser/util"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseClass(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	var javadoc *model.Javadoc
	if node.PrevSibling() != nil && node.PrevSibling().Type() == "block_comment" {
		commentContent := node.PrevSibling().Content(sourceCode)
		if strings.HasPrefix(commentContent, "/*") {
			javadoc = ParseJavadocTags(commentContent)
		}
	}
	className := node.ChildByFieldName("name").Content(sourceCode)
	packageName := ""
	accessModifier := ""
	superClass := ""
	annotationMarkers := []string{}
	implementedInterface := []string{}
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
		JavaDoc:          javadoc,
		Annotation:       annotationMarkers,
	}

	return classNode
}

package java

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseImportDeclaration(node *sitter.Node, sourceCode []byte, file string) *model.ImportType {
	importType := &model.ImportType{}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "scoped_identifier" || child.Type() == "identifier" {
			importType.ImportedType = child.Content(sourceCode)
		}
	}
	return importType
}

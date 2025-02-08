package db

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
)

type StorageNode struct {
	Package    []*model.Package
	ImportDecl []*model.ImportType
	Annotation []*model.Annotation
	ClassDecl  []*model.ClassOrInterface
	MethodDecl []*model.Method
	MethodCall []*model.MethodCall
	FieldDecl  []*model.FieldDeclaration
	Variable   []*model.LocalVariableDecl
	BinaryExpr []*model.BinaryExpr
	JavaDoc    []*model.Javadoc
}

func (s *StorageNode) AddPackage(node *model.Package) {
	s.Package = append(s.Package, node)
}

func (s *StorageNode) AddImportDecl(node *model.ImportType) {
	s.ImportDecl = append(s.ImportDecl, node)
}

func (s *StorageNode) AddClassDecl(node *model.ClassOrInterface) {
	s.ClassDecl = append(s.ClassDecl, node)
}

func (s *StorageNode) AddMethodDecl(node *model.Method) {
	s.MethodDecl = append(s.MethodDecl, node)
}

func (s *StorageNode) AddFieldDecl(node *model.FieldDeclaration) {
	s.FieldDecl = append(s.FieldDecl, node)
}

func (s *StorageNode) AddVariable(node *model.LocalVariableDecl) {
	s.Variable = append(s.Variable, node)
}

func (s *StorageNode) AddBinaryExpr(node *model.BinaryExpr) {
	s.BinaryExpr = append(s.BinaryExpr, node)
}

func (s *StorageNode) AddAnnotation(node *model.Annotation) {
	s.Annotation = append(s.Annotation, node)
}

func (s *StorageNode) AddJavaDoc(node *model.Javadoc) {
	s.JavaDoc = append(s.JavaDoc, node)
}

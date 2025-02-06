package db

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
)

type StorageNode struct {
	Package    []*model.TreeNode
	ImportDecl []*model.TreeNode
	ClassDecl  []*model.TreeNode
	MethodDecl []*model.TreeNode
	FieldDecl  []*model.TreeNode
	Variable   []*model.TreeNode
	BinaryExpr []*model.TreeNode
	Annotation []*model.TreeNode
	JavaDoc    []*model.TreeNode
	Comment    []*model.TreeNode
}

func (s *StorageNode) AddPackage(node *model.TreeNode) {
	s.Package = append(s.Package, node)
}

func (s *StorageNode) AddImportDecl(node *model.TreeNode) {
	s.ImportDecl = append(s.ImportDecl, node)
}

func (s *StorageNode) AddClassDecl(node *model.TreeNode) {
	s.ClassDecl = append(s.ClassDecl, node)
}

func (s *StorageNode) AddMethodDecl(node *model.TreeNode) {
	s.MethodDecl = append(s.MethodDecl, node)
}

func (s *StorageNode) AddFieldDecl(node *model.TreeNode) {
	s.FieldDecl = append(s.FieldDecl, node)
}

func (s *StorageNode) AddVariable(node *model.TreeNode) {
	s.Variable = append(s.Variable, node)
}

func (s *StorageNode) AddBinaryExpr(node *model.TreeNode) {
	s.BinaryExpr = append(s.BinaryExpr, node)
}

func (s *StorageNode) AddAnnotation(node *model.TreeNode) {
	s.Annotation = append(s.Annotation, node)
}

func (s *StorageNode) AddJavaDoc(node *model.TreeNode) {
	s.JavaDoc = append(s.JavaDoc, node)
}

func (s *StorageNode) AddComment(node *model.TreeNode) {
	s.Comment = append(s.Comment, node)
}

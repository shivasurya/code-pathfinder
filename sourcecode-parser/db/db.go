package db

import "github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"

type StorageNode struct {
	Package    []*graph.TreeNode
	ImportDecl []*graph.TreeNode
	ClassDecl  []*graph.TreeNode
	MethodDecl []*graph.TreeNode
	FieldDecl  []*graph.TreeNode
	Variable   []*graph.TreeNode
	BinaryExpr []*graph.TreeNode
	Annotation []*graph.TreeNode
	JavaDoc    []*graph.TreeNode
	Comment    []*graph.TreeNode
}

func (s *StorageNode) AddPackage(node *graph.TreeNode) {
	s.Package = append(s.Package, node)
}

func (s *StorageNode) AddImportDecl(node *graph.TreeNode) {
	s.ImportDecl = append(s.ImportDecl, node)
}

func (s *StorageNode) AddClassDecl(node *graph.TreeNode) {
	s.ClassDecl = append(s.ClassDecl, node)
}

func (s *StorageNode) AddMethodDecl(node *graph.TreeNode) {
	s.MethodDecl = append(s.MethodDecl, node)
}

func (s *StorageNode) AddFieldDecl(node *graph.TreeNode) {
	s.FieldDecl = append(s.FieldDecl, node)
}

func (s *StorageNode) AddVariable(node *graph.TreeNode) {
	s.Variable = append(s.Variable, node)
}

func (s *StorageNode) AddBinaryExpr(node *graph.TreeNode) {
	s.BinaryExpr = append(s.BinaryExpr, node)
}

func (s *StorageNode) AddAnnotation(node *graph.TreeNode) {
	s.Annotation = append(s.Annotation, node)
}

func (s *StorageNode) AddJavaDoc(node *graph.TreeNode) {
	s.JavaDoc = append(s.JavaDoc, node)
}

func (s *StorageNode) AddComment(node *graph.TreeNode) {
	s.Comment = append(s.Comment, node)
}

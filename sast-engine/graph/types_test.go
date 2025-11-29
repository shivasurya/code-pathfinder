package graph

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/model"
)

func TestNodeCreation(t *testing.T) {
	node := &Node{
		ID:          "test_id",
		Type:        "method_declaration",
		Name:        "testMethod",
		LineNumber:  10,
		IsExternal:  false,
		Modifier:    "public",
		ReturnType:  "void",
		PackageName: "com.test",
		File:        "Test.java",
	}

	if node.ID != "test_id" {
		t.Errorf("Expected ID 'test_id', got %s", node.ID)
	}
	if node.Type != "method_declaration" {
		t.Errorf("Expected Type 'method_declaration', got %s", node.Type)
	}
	if node.Name != "testMethod" {
		t.Errorf("Expected Name 'testMethod', got %s", node.Name)
	}
	if node.LineNumber != 10 {
		t.Errorf("Expected LineNumber 10, got %d", node.LineNumber)
	}
}

func TestNodeWithJavaDoc(t *testing.T) {
	javadoc := &model.Javadoc{
		Author:                "Test Author",
		Version:               "1.0",
		NumberOfCommentLines:  5,
		CommentedCodeElements: "/** Test javadoc */",
	}

	node := &Node{
		ID:      "test_id",
		Name:    "testMethod",
		JavaDoc: javadoc,
	}

	if node.JavaDoc == nil {
		t.Fatal("JavaDoc should not be nil")
	}
	if node.JavaDoc.Author != "Test Author" {
		t.Errorf("Expected Author 'Test Author', got %s", node.JavaDoc.Author)
	}
}

func TestNodeWithStatements(t *testing.T) {
	ifStmt := &model.IfStmt{
		ConditionalStmt: model.ConditionalStmt{
			Condition: &model.Expr{NodeString: "x > 0"},
		},
	}

	node := &Node{
		ID:     "test_id",
		Name:   "ifNode",
		Type:   "IfStmt",
		IfStmt: ifStmt,
	}

	if node.IfStmt == nil {
		t.Fatal("IfStmt should not be nil")
	}
	if node.IfStmt.Condition.NodeString != "x > 0" {
		t.Errorf("Expected condition 'x > 0', got %s", node.IfStmt.Condition.NodeString)
	}
}

func TestEdgeCreation(t *testing.T) {
	fromNode := &Node{ID: "from", Name: "FromNode"}
	toNode := &Node{ID: "to", Name: "ToNode"}

	edge := &Edge{
		From: fromNode,
		To:   toNode,
	}

	if edge.From.ID != "from" {
		t.Errorf("Expected From.ID 'from', got %s", edge.From.ID)
	}
	if edge.To.ID != "to" {
		t.Errorf("Expected To.ID 'to', got %s", edge.To.ID)
	}
}

func TestCodeGraphCreation(t *testing.T) {
	graph := &CodeGraph{
		Nodes: make(map[string]*Node),
		Edges: make([]*Edge, 0),
	}

	if graph.Nodes == nil {
		t.Error("Nodes map should not be nil")
	}
	if graph.Edges == nil {
		t.Error("Edges slice should not be nil")
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Errorf("Expected 0 edges, got %d", len(graph.Edges))
	}
}

func TestNodeLanguageFlags(t *testing.T) {
	javaNode := &Node{
		ID:               "java_node",
		isJavaSourceFile: true,
	}

	pythonNode := &Node{
		ID:                 "python_node",
		isPythonSourceFile: true,
	}

	if !javaNode.isJavaSourceFile {
		t.Error("Java node should have isJavaSourceFile=true")
	}
	if pythonNode.isJavaSourceFile {
		t.Error("Python node should have isJavaSourceFile=false")
	}
	if !pythonNode.isPythonSourceFile {
		t.Error("Python node should have isPythonSourceFile=true")
	}
}

func TestNodeMethodArguments(t *testing.T) {
	node := &Node{
		ID:                   "method_id",
		MethodArgumentsType:  []string{"int", "String"},
		MethodArgumentsValue: []string{"count", "name"},
	}

	if len(node.MethodArgumentsType) != 2 {
		t.Errorf("Expected 2 argument types, got %d", len(node.MethodArgumentsType))
	}
	if len(node.MethodArgumentsValue) != 2 {
		t.Errorf("Expected 2 argument values, got %d", len(node.MethodArgumentsValue))
	}
	if node.MethodArgumentsType[0] != "int" {
		t.Errorf("Expected first type 'int', got %s", node.MethodArgumentsType[0])
	}
	if node.MethodArgumentsValue[1] != "name" {
		t.Errorf("Expected second value 'name', got %s", node.MethodArgumentsValue[1])
	}
}

func TestNodeAnnotations(t *testing.T) {
	node := &Node{
		ID:         "annotated_method",
		Annotation: []string{"@Override", "@Deprecated"},
	}

	if len(node.Annotation) != 2 {
		t.Errorf("Expected 2 annotations, got %d", len(node.Annotation))
	}
	if node.Annotation[0] != "@Override" {
		t.Errorf("Expected first annotation '@Override', got %s", node.Annotation[0])
	}
}

func TestNodeExceptions(t *testing.T) {
	node := &Node{
		ID:               "throwing_method",
		ThrowsExceptions: []string{"IOException", "SQLException"},
	}

	if len(node.ThrowsExceptions) != 2 {
		t.Errorf("Expected 2 exceptions, got %d", len(node.ThrowsExceptions))
	}
	if node.ThrowsExceptions[0] != "IOException" {
		t.Errorf("Expected first exception 'IOException', got %s", node.ThrowsExceptions[0])
	}
}

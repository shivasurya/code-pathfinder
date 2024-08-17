package main

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"

	"github.com/expr-lang/expr"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
)

type Env struct {
	Node *GraphNode
}

func (env *Env) GetVisibility() string {
	return env.Node.Modifier
}

func (env *Env) GetAnnotations() []string {
	return env.Node.Annotation
}

func (env *Env) GetReturnType() string {
	return env.Node.ReturnType
}

func (env *Env) GetName() string {
	return env.Node.Name
}

func (env *Env) GetArgumentTypes() []string {
	return env.Node.MethodArgumentsType
}

func (env *Env) GetArgumentNames() []string {
	return env.Node.MethodArgumentsValue
}

func (env *Env) GetSuperClass() string {
	return env.Node.SuperClass
}

func (env *Env) GetInterfaces() []string {
	return env.Node.Interface
}

func (env *Env) GetScope() string {
	return env.Node.Scope
}

func (env *Env) GetVariableValue() string {
	return env.Node.VariableValue
}

func (env *Env) GetVariableDataType() string {
	return env.Node.DataType
}

func (env *Env) GetThrowsTypes() []string {
	return env.Node.ThrowsExceptions
}

func (env *Env) HasAccess() bool {
	return env.Node.hasAccess
}

func (env *Env) IsJavaSourceFile() bool {
	return env.Node.isJavaSourceFile
}

func (env *Env) GetDoc() *model.Javadoc {
	if env.Node.JavaDoc == nil {
		env.Node.JavaDoc = &model.Javadoc{}
	}
	return env.Node.JavaDoc
}

func QueryEntities(graph *CodeGraph, query parser.Query) []*GraphNode {
	result := make([]*GraphNode, 0)

	for _, node := range graph.Nodes {
		for _, entity := range query.SelectList {
			if entity.Entity == node.Type && FilterEntities(node, query) {
				result = append(result, node)
			}
		}
	}
	return result
}

func generateProxyEnv(node *GraphNode, query parser.Query) map[string]interface{} {
	proxyenv := Env{Node: node}
	methodDeclaration := "method_declaration"
	classDeclaration := "class_declaration"
	methodInvocation := "method_invocation"
	variableDeclaration := "variable_declaration"
	binaryExpression := "binary_expression"
	// print query select list
	for _, entity := range query.SelectList {
		switch entity.Entity {
		case "method_declaration":
			methodDeclaration = entity.Alias
		case "class_declaration":
			classDeclaration = entity.Alias
		case "method_invocation":
			methodInvocation = entity.Alias
		case "variable_declaration":
			variableDeclaration = entity.Alias
		case "binary_expression":
			binaryExpression = entity.Alias
		}
	}
	env := map[string]interface{}{
		"isJavaSourceFile": proxyenv.IsJavaSourceFile(),
		methodDeclaration: map[string]interface{}{
			"getVisibility":   proxyenv.GetVisibility,
			"getAnnotation":   proxyenv.GetAnnotations,
			"getReturnType":   proxyenv.GetReturnType,
			"getName":         proxyenv.GetName,
			"getArgumentType": proxyenv.GetArgumentTypes,
			"getArgumentName": proxyenv.GetArgumentNames,
			"getThrowsType":   proxyenv.GetThrowsTypes,
			"getDoc":          proxyenv.GetDoc,
		},
		classDeclaration: map[string]interface{}{
			"getSuperClass": proxyenv.GetSuperClass,
			"getName":       proxyenv.GetName,
			"getAnnotation": proxyenv.GetAnnotations,
			"getVisibility": proxyenv.GetVisibility,
			"getInterface":  proxyenv.GetInterfaces,
			"getDoc":        proxyenv.GetDoc,
		},
		methodInvocation: map[string]interface{}{
			"getArgumentName": proxyenv.GetArgumentNames,
			"getName":         proxyenv.GetName,
			"getDoc":          proxyenv.GetDoc,
		},
		variableDeclaration: map[string]interface{}{
			"getName":             proxyenv.GetName,
			"getVisibility":       proxyenv.GetVisibility,
			"getVariableValue":    proxyenv.GetVariableValue,
			"getVariableDataType": proxyenv.GetVariableDataType,
			"getScope":            proxyenv.GetScope,
			"getDoc":              proxyenv.GetDoc,
		},
		binaryExpression: map[string]interface{}{
			"getLeftOperand":  proxyenv.GetVariableValue,
			"getRightOperand": proxyenv.GetVariableValue,
			"getOperator":     proxyenv.GetVariableDataType,
			"getDoc":          proxyenv.GetDoc,
			"isExample":       true,
		},
	}
	return env
}

func FilterEntities(node *GraphNode, query parser.Query) bool {
	expression := query.Expression
	if expression == "" {
		return true
	}

	env := generateProxyEnv(node, query)

	program, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		fmt.Println("Error compiling expression: ", err)
		return false
	}
	output, err := expr.Run(program, env)
	if err != nil {
		fmt.Println("Error evaluating expression: ", err)
		return false
	}
	if output.(bool) {
		return true
	}
	return false
}

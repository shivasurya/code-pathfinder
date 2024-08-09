package main

import (
	"fmt"

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

func (env *Env) GetCommentAuthor() string {
	if env.Node.JavaDoc != nil {
		if env.Node.JavaDoc.Author != "" {
			return env.Node.JavaDoc.Author
		}
	}
	return ""
}

func (env *Env) GetCommentSee() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "see" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func (env *Env) GetCommentVersion() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "version" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func (env *Env) GetCommentSince() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "since" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func (env *Env) GetCommentParam() []string {
	result := []string{}
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "param" && docTag.Text != "" {
				result = append(result, docTag.Text)
			}
		}
	}
	return result
}

func (env *Env) GetCommentThrows() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "throws" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func (env *Env) GetCommentReturn() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "return" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
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
		}
	}
	env := map[string]interface{}{
		"isJavaSourceFile": proxyenv.IsJavaSourceFile(),
		methodDeclaration: map[string]interface{}{
			"getVisibility":     proxyenv.GetVisibility,
			"getAnnotation":     proxyenv.GetAnnotations,
			"getReturnType":     proxyenv.GetReturnType,
			"getName":           proxyenv.GetName,
			"getArgumentType":   proxyenv.GetArgumentTypes,
			"getArgumentName":   proxyenv.GetArgumentNames,
			"getThrowsType":     proxyenv.GetThrowsTypes,
			"getCommentAuthor":  proxyenv.GetCommentAuthor,
			"getCommentSee":     proxyenv.GetCommentSee,
			"getCommentVersion": proxyenv.GetCommentVersion,
			"getCommentSince":   proxyenv.GetCommentSince,
			"getCommentParams":  proxyenv.GetCommentParam,
			"getCommentThrows":  proxyenv.GetCommentThrows,
			"getCommentReturn":  proxyenv.GetCommentReturn,
		},
		classDeclaration: map[string]interface{}{
			"getSuperClass":     proxyenv.GetSuperClass,
			"getName":           proxyenv.GetName,
			"getAnnotation":     proxyenv.GetAnnotations,
			"getVisibility":     proxyenv.GetVisibility,
			"getInterface":      proxyenv.GetInterfaces,
			"getCommentAuthor":  proxyenv.GetCommentAuthor,
			"getCommentSee":     proxyenv.GetCommentSee,
			"getCommentVersion": proxyenv.GetCommentVersion,
			"getCommentSince":   proxyenv.GetCommentSince,
			"getCommentParams":  proxyenv.GetCommentParam,
			"getCommentThrows":  proxyenv.GetCommentThrows,
			"getCommentReturn":  proxyenv.GetCommentReturn,
		},
		methodInvocation: map[string]interface{}{
			"getArgumentName":   proxyenv.GetArgumentNames,
			"getName":           proxyenv.GetName,
			"getCommentAuthor":  proxyenv.GetCommentAuthor,
			"getCommentSee":     proxyenv.GetCommentSee,
			"getCommentVersion": proxyenv.GetCommentVersion,
			"getCommentSince":   proxyenv.GetCommentSince,
			"getCommentParams":  proxyenv.GetCommentParam,
			"getCommentThrows":  proxyenv.GetCommentThrows,
			"getCommentReturn":  proxyenv.GetCommentReturn,
		},
		variableDeclaration: map[string]interface{}{
			"getName":             proxyenv.GetName,
			"getVisibility":       proxyenv.GetVisibility,
			"getVariableValue":    proxyenv.GetVariableValue,
			"getVariableDataType": proxyenv.GetVariableDataType,
			"getScope":            proxyenv.GetScope,
			"getCommentAuthor":    proxyenv.GetCommentAuthor,
			"getCommentSee":       proxyenv.GetCommentSee,
			"getCommentVersion":   proxyenv.GetCommentVersion,
			"getCommentSince":     proxyenv.GetCommentSince,
			"getCommentParam":     proxyenv.GetCommentParam,
			"getCommentThrows":    proxyenv.GetCommentThrows,
			"getCommentReturn":    proxyenv.GetCommentReturn,
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

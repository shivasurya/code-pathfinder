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

func (env *Env) GetCommentParam() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "param" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
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
	proxyenv := &Env{Node: node}
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
		"isJavaSourceFile": func() bool {
			return proxyenv.IsJavaSourceFile()
		},
		"getCommentAuthor": func() string {
			return proxyenv.GetCommentAuthor()
		},
		"getCommentSee": func() string {
			return proxyenv.GetCommentSee()
		},
		"getCommentVersion": func() string {
			return proxyenv.GetCommentVersion()
		},
		"getCommentSince": func() string {
			return proxyenv.GetCommentSince()
		},
		"getCommentParam": func() string {
			return proxyenv.GetCommentParam()
		},
		"getCommentThrows": func() string {
			return proxyenv.GetCommentThrows()
		},
		"getCommentReturn": func() string {
			return proxyenv.GetCommentReturn()
		},
		methodDeclaration: map[string]interface{}{
			"getVisibility": func() string {
				return proxyenv.GetVisibility()
			},
			"getAnnotation": func() []string {
				return proxyenv.GetAnnotations()
			},
			"getReturnType": func() string {
				return proxyenv.GetReturnType()
			},
			"getName": func() string {
				return proxyenv.GetName()
			},
			"getArgumentType": func() []string {
				return proxyenv.GetArgumentTypes()
			},
			"getArgumentName": func() []string {
				return proxyenv.GetArgumentNames()
			},
			"getInterface": func() []string {
				return proxyenv.GetInterfaces()
			},
			"getThrowsType": func() []string {
				return proxyenv.GetThrowsTypes()
			},
		},
		classDeclaration: map[string]interface{}{
			"getSuperClass": func() string {
				return proxyenv.GetSuperClass()
			},
			"getName": func() string {
				return proxyenv.GetName()
			},
			"getAnnotation": func() []string {
				return proxyenv.GetAnnotations()
			},
			"getVisibility": func() string {
				return proxyenv.GetVisibility()
			},
			"getInterface": func() []string {
				return proxyenv.GetInterfaces()
			},
		},
		methodInvocation: map[string]interface{}{
			"getArgumentName": func() []string {
				return proxyenv.GetArgumentNames()
			},
			"getName": func() string {
				return proxyenv.GetName()
			},
		},
		variableDeclaration: map[string]interface{}{
			"getName": func() string {
				return proxyenv.GetName()
			},
			"getVisibility": func() string {
				return proxyenv.GetVisibility()
			},
			"getVariableValue": func() string {
				return proxyenv.GetVariableValue()
			},
			"getVariableDataType": func() string {
				return proxyenv.GetVariableDataType()
			},
			"getScope": func() string {
				return proxyenv.GetScope()
			},
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

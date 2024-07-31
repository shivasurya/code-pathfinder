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

func (env *Env) GetAnnotation(annotationVal string) string {
	for _, annotation := range env.Node.Annotation {
		if annotation == annotationVal {
			return annotation
		}
	}
	return ""
}

func (env *Env) GetReturnType() string {
	return env.Node.ReturnType
}

func (env *Env) GetName() string {
	return env.Node.Name
}

func (env *Env) GetArgumentType(argVal string) string {
	for i, arg := range env.Node.MethodArgumentsType {
		if arg == argVal {
			return env.Node.MethodArgumentsType[i]
		}
	}
	return ""
}

func (env *Env) GetArgumentName(argVal string) string {
	for i, arg := range env.Node.MethodArgumentsValue {
		if arg == argVal {
			return env.Node.MethodArgumentsValue[i]
		}
	}
	return ""
}

func (env *Env) GetSuperClass() string {
	return env.Node.SuperClass
}

func (env *Env) GetInterface(interfaceVal string) string {
	for i, iface := range env.Node.Interface {
		if iface == interfaceVal {
			return env.Node.Interface[i]
		}
	}
	return ""
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

func (env *Env) GetThrowsType(throwsVal string) string {
	for i, arg := range env.Node.ThrowsExceptions {
		if arg == throwsVal {
			return env.Node.ThrowsExceptions[i]
		}
	}
	return ""
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
		methodDeclaration: map[string]interface{}{
			"getVisibility": func() string {
				return proxyenv.GetVisibility()
			},
			"getAnnotation": func(annotationVal string) string {
				return proxyenv.GetAnnotation(annotationVal)
			},
			"getReturnType": func() string {
				return proxyenv.GetReturnType()
			},
			"getName": func() string {
				return proxyenv.GetName()
			},
			"getArgumentType": func(argVal string) string {
				return proxyenv.GetArgumentType(argVal)
			},
			"getArgumentName": func(argVal string) string {
				return proxyenv.GetArgumentName(argVal)
			},
			"getInterface": func(interfaceVal string) string {
				return proxyenv.GetInterface(interfaceVal)
			},
			"getScope": func() string {
				return proxyenv.GetScope()
			},
		},
		classDeclaration: map[string]interface{}{
			"getSuperClass": func() string {
				return proxyenv.GetSuperClass()
			},
			"getName": func() string {
				return proxyenv.GetName()
			},
		},
		methodInvocation: map[string]interface{}{
			"getThrowsType": func(throwsVal string) string {
				return proxyenv.GetThrowsType(throwsVal)
			},
			"getName": func() string {
				return proxyenv.GetName()
			},
		},
		variableDeclaration: map[string]interface{}{
			"getVariableValue": func() string {
				return proxyenv.GetVariableValue()
			},
			"getVariableDataType": func() string {
				return proxyenv.GetVariableDataType()
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

package graph

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/analytics"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"

	"github.com/expr-lang/expr"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
)

type Env struct {
	Node *Node
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

func (env *Env) GetBinaryExpr() *model.BinaryExpr {
	return env.Node.BinaryExpr
}

func (env *Env) GetLeftOperand() string {
	return env.Node.BinaryExpr.LeftOperand.NodeString
}

func (env *Env) GetRightOperand() string {
	return env.Node.BinaryExpr.RightOperand.NodeString
}

func QueryEntities(graph *CodeGraph, query parser.Query) []*Node {
	result := make([]*Node, 0)

	// log query select list alone
	for _, entity := range query.SelectList {
		analytics.ReportEvent(entity.Entity)
	}

	for _, node := range graph.Nodes {
		for _, entity := range query.SelectList {
			if entity.Entity == node.Type && FilterEntities(node, query) {
				result = append(result, node)
			}
		}
	}
	return result
}

func generateProxyEnv(node *Node, query parser.Query) map[string]interface{} {
	proxyenv := Env{Node: node}
	methodDeclaration := "method_declaration"
	classDeclaration := "class_declaration"
	methodInvocation := "method_invocation"
	variableDeclaration := "variable_declaration"
	binaryExpression := "binary_expression"
	addExpression := "add_expression"
	subExpression := "sub_expression"
	mulExpression := "mul_expression"
	divExpression := "div_expression"
	comparisionExpression := "comparison_expression"
	remainderExpression := "remainder_expression"
	rightShiftExpression := "right_shift_expression"
	leftShiftExpression := "left_shift_expression"
	notEqualExpression := "not_equal_expression"
	equalExpression := "equal_expression"
	andBitwiseExpression := "and_bitwise_expression"
	andLogicalExpression := "and_logical_expression"
	orLogicalExpression := "or_logical_expression"
	orBitwiseExpression := "or_bitwise_expression"
	unsignedRightShiftExpression := "unsigned_right_shift_expression"
	xorBitwsieExpression := "xor_bitwise_expression"

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
		case "add_expression":
			addExpression = entity.Alias
		case "sub_expression":
			subExpression = entity.Alias
		case "mul_expression":
			mulExpression = entity.Alias
		case "div_expression":
			divExpression = entity.Alias
		case "comparison_expression":
			comparisionExpression = entity.Alias
		case "remainder_expression":
			remainderExpression = entity.Alias
		case "right_shift_expression":
			rightShiftExpression = entity.Alias
		case "left_shift_expression":
			leftShiftExpression = entity.Alias
		case "not_equal_expression":
			notEqualExpression = entity.Alias
		case "equal_expression":
			equalExpression = entity.Alias
		case "and_bitwise_expression":
			andBitwiseExpression = entity.Alias
		case "and_logical_expression":
			andLogicalExpression = entity.Alias
		case "or_logical_expression":
			orLogicalExpression = entity.Alias
		case "or_bitwise_expression":
			orBitwiseExpression = entity.Alias
		case "unsigned_right_shift_expression":
			unsignedRightShiftExpression = entity.Alias
		case "xor_bitwise_expression":
			xorBitwsieExpression = entity.Alias
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
			"getLeftOperand":  proxyenv.GetLeftOperand,
			"getRightOperand": proxyenv.GetRightOperand,
		},
		addExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "+",
		},
		subExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "-",
		},
		mulExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "*",
		},
		divExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "/",
		},
		comparisionExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "==",
		},
		remainderExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "%",
		},
		rightShiftExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   ">>",
		},
		leftShiftExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "<<",
		},
		notEqualExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "!=",
		},
		equalExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "==",
		},
		andBitwiseExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "&",
		},
		andLogicalExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "&&",
		},
		orLogicalExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "||",
		},
		orBitwiseExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "|",
		},
		unsignedRightShiftExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   ">>>",
		},
		xorBitwsieExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "^",
		},
	}
	return env
}

func FilterEntities(node *Node, query parser.Query) bool {
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

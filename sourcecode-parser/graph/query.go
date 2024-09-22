package graph

import (
	"fmt"
	"log"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/analytics"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
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

	cartesianProduct := generateCartesianProduct(graph, query.SelectList, query.Condition)

	for _, nodeSet := range cartesianProduct {
		if FilterEntities(nodeSet, query) {
			result = append(result, nodeSet...)
		}
	}
	return result
}

func generateCartesianProduct(graph *CodeGraph, selectList []parser.SelectList, conditions []string) [][]*Node {
	typeIndex := make(map[string][]*Node)

	// value and reference based reducing search space
	for _, condition := range conditions {
		// this code helps to reduce search space
		// if there is single entity in select list, the condition is easy to reduce the search space
		// if there are multiple entities in select list, the condition is hard to reduce the search space,
		// but I have tried my best using O(n^2) time complexity to reduce the search space
		if len(selectList) > 1 {
			lhsNodes := graph.FindNodesByType(selectList[0].Entity)
			rhsNodes := graph.FindNodesByType(selectList[1].Entity)
			for _, lhsNode := range lhsNodes {
				for _, rhsNode := range rhsNodes {
					if FilterEntities([]*Node{lhsNode, rhsNode}, parser.Query{Expression: condition, SelectList: selectList}) {
						typeIndex[lhsNode.Type] = append(typeIndex[lhsNode.Type], lhsNode)
						typeIndex[rhsNode.Type] = append(typeIndex[rhsNode.Type], rhsNode)
					}
				}
			}
		} else {
			for _, node := range graph.Nodes {
				query := parser.Query{Expression: condition, SelectList: selectList}
				if FilterEntities([]*Node{node}, query) {
					typeIndex[node.Type] = append(typeIndex[node.Type], node)
				}
			}
		}
	}

	if len(conditions) == 0 {
		for _, node := range graph.Nodes {
			typeIndex[node.Type] = append(typeIndex[node.Type], node)
		}
	}

	sets := make([][]interface{}, 0, len(selectList))

	for _, entity := range selectList {
		set := make([]interface{}, 0)
		if nodes, ok := typeIndex[entity.Entity]; ok {
			for _, node := range nodes {
				set = append(set, node)
			}
		}
		sets = append(sets, set)
	}

	product := cartesianProduct(sets)

	result := make([][]*Node, len(product))
	for i, p := range product {
		result[i] = make([]*Node, len(p))
		for j, node := range p {
			if n, ok := node.(*Node); ok {
				result[i][j] = n
			} else {
				// Handle the error case, e.g., skip this node or log an error
				// You might want to customize this part based on your error handling strategy
				log.Printf("Warning: Expected *Node type, got %T", node)
			}
		}
	}

	return result
}

func cartesianProduct(sets [][]interface{}) [][]interface{} {
	result := [][]interface{}{{}}
	for _, set := range sets {
		var newResult [][]interface{}
		for _, item := range set {
			for _, subResult := range result {
				newSubResult := make([]interface{}, len(subResult), len(subResult)+1)
				copy(newSubResult, subResult)
				newSubResult = append(newSubResult, item)
				newResult = append(newResult, newSubResult)
			}
		}
		result = newResult
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

func ReplacePredicateVariables(query parser.Query) string {
	expression := query.Expression
	if expression == "" {
		return query.Expression
	}

	for _, invokedPredicate := range query.PredicateInvocation {
		predicateExpression := invokedPredicate.PredicateName + "("
		for i, param := range invokedPredicate.Parameter {
			predicateExpression += param.Name + ","
			for _, entity := range query.SelectList {
				if entity.Alias == param.Name {
					matchedPredicate := invokedPredicate.Predicate
					invokedPredicate.Predicate.Body = strings.ReplaceAll(invokedPredicate.Predicate.Body, matchedPredicate.Parameter[i].Name, entity.Alias)
				}
			}
		}
		// remove the last comma
		predicateExpression = predicateExpression[:len(predicateExpression)-1]
		predicateExpression += ")"
		invokedPredicate.Predicate.Body = "(" + invokedPredicate.Predicate.Body + ")"
		expression = strings.ReplaceAll(expression, predicateExpression, invokedPredicate.Predicate.Body)
	}
	return expression
}

func FilterEntities(node []*Node, query parser.Query) bool {
	expression := query.Expression
	if expression == "" {
		return true
	}

	env := generateProxyEnvForSet(node, query)

	expression = ReplacePredicateVariables(query)

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

func generateProxyEnvForSet(nodeSet []*Node, query parser.Query) map[string]interface{} {
	env := make(map[string]interface{})

	for i, entity := range query.SelectList {
		proxyEnv := generateProxyEnv(nodeSet[i], query)
		env[entity.Alias] = proxyEnv[entity.Alias]
	}

	return env
}

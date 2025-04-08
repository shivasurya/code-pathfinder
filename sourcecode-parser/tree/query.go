package graph

import (
	"fmt"
	"log"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/analytics"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/db"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
)

type Env struct {
	Node *model.Node
}

// func (env *Env) GetVisibility() string {
// 	return env.Node.Modifier
// }

// func (env *Env) GetAnnotations() []string {
// 	return env.Node.Annotation
// }

// func (env *Env) GetReturnType() string {
// 	return env.Node.ReturnType
// }

// func (env *Env) GetName() string {
// 	return env.Node.Name
// }

// func (env *Env) GetArgumentTypes() []string {
// 	return env.Node.MethodArgumentsType
// }

// func (env *Env) GetArgumentNames() []string {
// 	return env.Node.MethodArgumentsValue
// }

// func (env *Env) GetSuperClass() string {
// 	return env.Node.SuperClass
// }

// func (env *Env) GetInterfaces() []string {
// 	return env.Node.Interface
// }

// func (env *Env) GetScope() string {
// 	return env.Node.Scope
// }

// func (env *Env) GetVariableValue() string {
// 	return env.Node.VariableValue
// }

// func (env *Env) GetVariableDataType() string {
// 	return env.Node.DataType
// }

// func (env *Env) GetThrowsTypes() []string {
// 	return env.Node.ThrowsExceptions
// }

func (env *Env) IsJavaSourceFile() bool {
	return true
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

func (env *Env) ToString() string {
	node := env.Node
	if node == nil {
		return ""
	}

	if node.AddExpr != nil {
		return node.AddExpr.LeftOperand.NodeString + " + " + node.AddExpr.RightOperand.NodeString
	} else if node.SubExpr != nil {
		return node.SubExpr.LeftOperand.NodeString + " - " + node.SubExpr.RightOperand.NodeString
	} else if node.MulExpr != nil {
		return node.MulExpr.LeftOperand.NodeString + " * " + node.MulExpr.RightOperand.NodeString
	} else if node.DivExpr != nil {
		return node.DivExpr.LeftOperand.NodeString + " / " + node.DivExpr.RightOperand.NodeString
	} else if node.MethodDecl != nil {
		return node.MethodDecl.Name
	} else if node.MethodCall != nil {
		return node.MethodCall.MethodName
	} else if node.ClassInstanceExpr != nil {
		return node.ClassInstanceExpr.ClassName
	} else if node.IfStmt != nil {
		return node.IfStmt.Condition.NodeString
	} else if node.WhileStmt != nil {
		return "while"
	} else if node.DoStmt != nil {
		return "do"
	} else if node.ForStmt != nil {
		return "for"
	} else if node.BreakStmt != nil {
		return "break"
	} else if node.ContinueStmt != nil {
		return "continue"
	} else if node.ReturnStmt != nil {
		return "return"
	}

	return ""
}

func (env *Env) GetRightOperand() string {
	return env.Node.BinaryExpr.RightOperand.NodeString
}

func (env *Env) GetClassInstanceExpr() *model.ClassInstanceExpr {
	return env.Node.ClassInstanceExpr
}

func (env *Env) GetClassInstanceExprName() string {
	return env.Node.ClassInstanceExpr.ClassName
}

func (env *Env) GetIfStmt() *model.IfStmt {
	return env.Node.IfStmt
}

func (env *Env) GetWhileStmt() *model.WhileStmt {
	return env.Node.WhileStmt
}

func (env *Env) GetDoStmt() *model.DoStmt {
	return env.Node.DoStmt
}

func (env *Env) GetForStmt() *model.ForStmt {
	return env.Node.ForStmt
}

func (env *Env) GetBreakStmt() *model.BreakStmt {
	return env.Node.BreakStmt
}

func (env *Env) GetContinueStmt() *model.ContinueStmt {
	return env.Node.ContinueStmt
}

func (env *Env) GetYieldStmt() *model.YieldStmt {
	return env.Node.YieldStmt
}

func (env *Env) GetAssertStmt() *model.AssertStmt {
	return env.Node.AssertStmt
}

func (env *Env) GetReturnStmt() *model.ReturnStmt {
	return env.Node.ReturnStmt
}

func (env *Env) GetBlockStmt() *model.BlockStmt {
	return env.Node.BlockStmt
}

func QueryEntities(db *db.StorageNode, treeHolder []*model.TreeNode, query parser.Query) (nodes [][]*model.Node, output [][]interface{}) {
	// Create evaluation context
	ctx := &parser.EvaluationContext{
		RelationshipMap: buildRelationshipMap(),
		EntityData:      make(map[string][]map[string]interface{}),
	}

	// Log query select list
	for _, entity := range query.SelectList {
		analytics.ReportEvent(entity.Entity)
	}

	// Prepare entity data
	for _, entity := range query.SelectList {
		entityData := make([]map[string]interface{}, 0)
		// Get nodes for this entity type
		nodes := getNodesForEntity(db, treeHolder, entity)
		// Convert nodes to map[string]interface{}
		for _, node := range nodes {
			entityData = append(entityData, nodeToMap(node))
		}
		ctx.EntityData[entity.Entity] = entityData
	}

	// Use the expression tree from the query
	if query.ExpressionTree == nil {
		return nil, nil
	}

	// Evaluate the condition
	result, err := parser.EvaluateExpressionTree(query.ExpressionTree, ctx)
	if err != nil {
		// Handle error appropriately
		return nil, nil
	}

	// Convert result data back to nodes
	resultNodes := make([][]*model.Node, 0)
	for _, data := range result.Data {
		nodeSet := make([]*model.Node, 0)
		for _, entity := range query.SelectList {
			if node := findNodeByData(db, treeHolder, entity, data); node != nil {
				nodeSet = append(nodeSet, node)
			}
		}
		if len(nodeSet) == len(query.SelectList) {
			resultNodes = append(resultNodes, nodeSet)
		}
	}

	output = generateOutput(resultNodes, query)
	return resultNodes, output
}

// buildRelationshipMap creates a relationship map for the entities
func buildRelationshipMap() *parser.RelationshipMap {
	rm := parser.NewRelationshipMap()
	// Add relationships between entities
	// For example:
	rm.AddRelationship("class", "methods", []string{"method"})
	rm.AddRelationship("method", "class", []string{"class"})
	return rm
}

// getNodesForEntity returns all nodes for a given entity type
func getNodesForEntity(db *db.StorageNode, treeHolder []*model.TreeNode, entity parser.SelectList) []*model.Node {
	// Get nodes for this entity type
	nodes := make([]*model.Node, 0)
	for _, tree := range treeHolder {
		if tree.Node.NodeType == entity.Entity {
			nodes = append(nodes, tree.Node)
		}
	}
	return nodes
}

// nodeToMap converts a node to a map[string]interface{}
func nodeToMap(node *model.Node) map[string]interface{} {
	result := make(map[string]interface{})
	// Add basic node properties
	result["id"] = node.NodeID
	result["type"] = node.NodeType
	// Add other properties based on node type
	switch node.NodeType {
	case "class":
		if node.ClassDecl != nil {
			result["name"] = node.ClassDecl.QualifiedName
		}
	case "method":
		if node.MethodDecl != nil {
			result["name"] = node.MethodDecl.QualifiedName
		}
	}
	return result
}

// findNodeByData finds a node that matches the given data
func findNodeByData(db *db.StorageNode, treeHolder []*model.TreeNode, entity parser.SelectList, data map[string]interface{}) *model.Node {
	// Get all nodes for this entity type
	nodes := getNodesForEntity(db, treeHolder, entity)
	
	// Find the node that matches the data
	for _, node := range nodes {
		if matchesData(node, data) {
			return node
		}
	}
	return nil
}

// matchesData checks if a node matches the given data
func matchesData(node *model.Node, data map[string]interface{}) bool {
	// Check if basic properties match
	if id, ok := data["id"]; ok {
		if id != node.NodeID {
			return false
		}
	}
	if nodeType, ok := data["type"]; ok {
		if nodeType != node.NodeType {
			return false
		}
	}
	if name, ok := data["name"]; ok {
		switch node.NodeType {
		case "class":
			if node.ClassDecl == nil || name != node.ClassDecl.QualifiedName {
				return false
			}
		case "method":
			if node.MethodDecl == nil || name != node.MethodDecl.QualifiedName {
				return false
			}
		}
	}
	return true
}

func generateOutput(nodeSet [][]*model.Node, query parser.Query) [][]interface{} {
	results := make([][]interface{}, 0, len(nodeSet))
	for _, nodeSet := range nodeSet {
		var result []interface{}
		for _, outputFormat := range query.SelectOutput {
			switch outputFormat.Type {
			case "string":
				outputFormat.SelectEntity = strings.ReplaceAll(outputFormat.SelectEntity, "\"", "")
				result = append(result, outputFormat.SelectEntity)
			case "method_chain", "variable":
				if outputFormat.Type == "variable" {
					outputFormat.SelectEntity += ".toString()"
				} else if outputFormat.Type == "method_chain" {
					if !strings.Contains(outputFormat.SelectEntity, ".") {
						continue
					}
				}
				response, err := evaluateExpression(nodeSet, outputFormat.SelectEntity, query)
				if err != nil {
					log.Print(err)
				}
				result = append(result, response)
			}
		}
		results = append(results, result)
	}
	return results
}

func evaluateExpression(node []*model.Node, expression string, query parser.Query) (interface{}, error) {
	env := generateProxyEnvForSet(node, query)

	program, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		fmt.Println("Error compiling expression: ", err)
		return "", err
	}
	output, err := expr.Run(program, env)
	if err != nil {
		fmt.Println("Error evaluating expression: ", err)
		return "", err
	}
	return output, nil
}

func generateCartesianProduct(db *db.StorageNode, treeHolder []*model.TreeNode, selectList []parser.SelectList, conditions []string) [][]*model.Node {
	// select list may contain multiple entities, create holder for each entity
	ts := make([][]interface{}, 0, len(selectList))
	// for each entity, get all nodes
	for _, entity := range selectList {
		ts = append(ts, db.GetTypedSlice(entity.Entity))
	}
	// figure out way to join entity nodes together using relationships
	// at worst case is cartesian product

	sets := make([][]interface{}, 0, len(selectList))
	sets = append(sets, ts...)

	product := cartesianProduct(sets)

	// Convert [][]interface{} to [][]*model.Node
	result := make([][]*model.Node, len(product))
	for i, row := range product {
		result[i] = make([]*model.Node, len(row))
		for j, item := range row {
			if node, ok := item.(*model.Node); ok {
				result[i][j] = node
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

func generateProxyEnv(node *model.Node, query parser.Query) map[string]interface{} {
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
	classInstanceExpression := "ClassInstanceExpr"
	ifStmt := "IfStmt"
	whileStmt := "WhileStmt"
	doStmt := "DoStmt"
	forStmt := "ForStmt"
	breakStmt := "BreakStmt"
	continueStmt := "ContinueStmt"
	yieldStmt := "YieldStmt"
	assertStmt := "AssertStmt"
	returnStmt := "ReturnStmt"
	blockStmt := "BlockStmt"

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
		case "ClassInstanceExpr":
			classInstanceExpression = entity.Alias
		case "IfStmt":
			ifStmt = entity.Alias
		case "WhileStmt":
			whileStmt = entity.Alias
		case "DoStmt":
			doStmt = entity.Alias
		case "ForStmt":
			forStmt = entity.Alias
		case "BreakStmt":
			breakStmt = entity.Alias
		case "ContinueStmt":
			continueStmt = entity.Alias
		case "YieldStmt":
			yieldStmt = entity.Alias
		case "AssertStmt":
			assertStmt = entity.Alias
		case "ReturnStmt":
			returnStmt = entity.Alias
		case "BlockStmt":
			blockStmt = entity.Alias
		}
	}
	env := map[string]interface{}{
		"isJavaSourceFile": proxyenv.IsJavaSourceFile(),
		methodDeclaration: map[string]interface{}{
			// "getVisibility":   proxyenv.GetVisibility,
			// "getAnnotation":   proxyenv.GetAnnotations,
			// "getReturnType":   proxyenv.GetReturnType,
			// "getName":         proxyenv.GetName,
			// "getArgumentType": proxyenv.GetArgumentTypes,
			// "getArgumentName": proxyenv.GetArgumentNames,
			// "getThrowsType":   proxyenv.GetThrowsTypes,
			"getDoc":   proxyenv.GetDoc,
			"toString": proxyenv.ToString,
		},
		classDeclaration: map[string]interface{}{
			// "getSuperClass": proxyenv.GetSuperClass,
			// "getName":       proxyenv.GetName,
			// "getAnnotation": proxyenv.GetAnnotations,
			// "getVisibility": proxyenv.GetVisibility,
			// "getInterface":  proxyenv.GetInterfaces,
			"getDoc":   proxyenv.GetDoc,
			"toString": proxyenv.ToString,
		},
		methodInvocation: map[string]interface{}{
			// "getArgumentName": proxyenv.GetArgumentNames,
			// "getName":         proxyenv.GetName,
			"getDoc":   proxyenv.GetDoc,
			"toString": proxyenv.ToString,
		},
		variableDeclaration: map[string]interface{}{
			// "getName":             proxyenv.GetName,
			// "getVisibility":       proxyenv.GetVisibility,
			// "getVariableValue":    proxyenv.GetVariableValue,
			// "getVariableDataType": proxyenv.GetVariableDataType,
			// "getScope":            proxyenv.GetScope,
			"getDoc":   proxyenv.GetDoc,
			"toString": proxyenv.ToString,
		},
		binaryExpression: map[string]interface{}{
			"getLeftOperand":  proxyenv.GetLeftOperand,
			"getRightOperand": proxyenv.GetRightOperand,
			"toString":        proxyenv.ToString,
		},
		addExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "+",
			"toString":      proxyenv.ToString,
		},
		subExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "-",
			"toString":      proxyenv.ToString,
		},
		mulExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "*",
			"toString":      proxyenv.ToString,
		},
		divExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "/",
			"toString":      proxyenv.ToString,
		},
		comparisionExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "==",
			"toString":      proxyenv.ToString,
		},
		remainderExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "%",
			"toString":      proxyenv.ToString,
		},
		rightShiftExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   ">>",
			"toString":      proxyenv.ToString,
		},
		leftShiftExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "<<",
			"toString":      proxyenv.ToString,
		},
		notEqualExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "!=",
			"toString":      proxyenv.ToString,
		},
		equalExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "==",
			"toString":      proxyenv.ToString,
		},
		andBitwiseExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "&",
			"toString":      proxyenv.ToString,
		},
		andLogicalExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "&&",
			"toString":      proxyenv.ToString,
		},
		orLogicalExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "||",
			"toString":      proxyenv.ToString,
		},
		orBitwiseExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "|",
			"toString":      proxyenv.ToString,
		},
		unsignedRightShiftExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   ">>>",
			"toString":      proxyenv.ToString,
		},
		xorBitwsieExpression: map[string]interface{}{
			"getBinaryExpr": proxyenv.GetBinaryExpr,
			"getOperator":   "^",
			"toString":      proxyenv.ToString,
		},
		classInstanceExpression: map[string]interface{}{
			// "getName":              proxyenv.GetName,
			"getDoc":               proxyenv.GetDoc,
			"toString":             proxyenv.ToString,
			"getClassInstanceExpr": proxyenv.GetClassInstanceExpr,
		},
		ifStmt: map[string]interface{}{
			"getIfStmt": proxyenv.GetIfStmt,
			"toString":  proxyenv.ToString,
		},
		whileStmt: map[string]interface{}{
			"getWhileStmt": proxyenv.GetWhileStmt,
			"toString":     proxyenv.ToString,
		},
		doStmt: map[string]interface{}{
			"getDoStmt": proxyenv.GetDoStmt,
			"toString":  proxyenv.ToString,
		},
		forStmt: map[string]interface{}{
			"getForStmt": proxyenv.GetForStmt,
			"toString":   proxyenv.ToString,
		},
		breakStmt: map[string]interface{}{
			"toString":     proxyenv.ToString,
			"getBreakStmt": proxyenv.GetBreakStmt,
		},
		continueStmt: map[string]interface{}{
			"toString":        proxyenv.ToString,
			"getContinueStmt": proxyenv.GetContinueStmt,
		},
		yieldStmt: map[string]interface{}{
			"toString":     proxyenv.ToString,
			"getYieldStmt": proxyenv.GetYieldStmt,
		},
		assertStmt: map[string]interface{}{
			"toString":      proxyenv.ToString,
			"getAssertStmt": proxyenv.GetAssertStmt,
		},
		returnStmt: map[string]interface{}{
			"toString":      proxyenv.ToString,
			"getReturnStmt": proxyenv.GetReturnStmt,
		},
		blockStmt: map[string]interface{}{
			"toString":     proxyenv.ToString,
			"getBlockStmt": proxyenv.GetBlockStmt,
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

func FilterEntities(node []*model.Node, query parser.Query) bool {
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
	if output.(bool) { //nolint:all
		return true
	}
	return false
}

type classInstance struct {
	Class   *parser.ClassDeclaration
	Methods map[string]string // method name -> result
}

func generateProxyEnvForSet(nodeSet []*model.Node, query parser.Query) map[string]interface{} {
	env := make(map[string]interface{})

	for i, entity := range query.SelectList {
		// Check if entity is a class type
		classDecl := findClassDeclaration(entity.Entity, query.Classes)
		if classDecl != nil {
			env[entity.Alias] = createClassInstance(classDecl)
		} else {
			// Handle existing node types
			proxyEnv := generateProxyEnv(nodeSet[i], query)
			env[entity.Alias] = proxyEnv[entity.Alias]
		}
	}
	return env
}

func findClassDeclaration(className string, classes []parser.ClassDeclaration) *parser.ClassDeclaration {
	for _, class := range classes {
		if class.Name == className {
			return &class
		}
	}
	return nil
}

func createClassInstance(class *parser.ClassDeclaration) *classInstance {
	instance := &classInstance{
		Class:   class,
		Methods: make(map[string]string),
	}

	// Initialize method results
	for _, method := range class.Methods {
		instance.Methods[method.Name] = method.Body
	}

	return instance
}

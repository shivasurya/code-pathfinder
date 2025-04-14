package eval

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
)

// IntermediateResult represents intermediate evaluation state at each node
type IntermediateResult struct {
	NodeType    string
	Operator    string
	Data        []interface{}
	Entities    []string
	LeftResult  *IntermediateResult
	RightResult *IntermediateResult
	Value       interface{}
	Err         error
}

// EvaluationResult represents the final result of evaluating an expression
type EvaluationResult struct {
	Data          []interface{}         // The filtered data after evaluation
	Entities      []string              // The entities involved in this evaluation
	Err           error                 // Any error that occurred during evaluation
	Intermediates []*IntermediateResult // Intermediate results for debugging
}

// EvaluationContext holds the context for expression evaluation
type EvaluationContext struct {
	RelationshipMap *RelationshipMap
	ProxyEnv        map[string][]map[string]interface{}
	EntityModel     map[string][]interface{}
}

// ComparisonType represents the type of comparison in an expression
type ComparisonType string

const (
	// SINGLE_ENTITY represents comparison between one entity and a static value
	SINGLE_ENTITY ComparisonType = "SINGLE_ENTITY"
	// DUAL_ENTITY represents comparison between two different entities
	DUAL_ENTITY ComparisonType = "DUAL_ENTITY"
)

func EvaluateExpressionTree(tree *parser.ExpressionNode, ctx *EvaluationContext) (*EvaluationResult, error) {
	if tree == nil {
		return &EvaluationResult{}, nil
	}

	// Evaluate the tree bottom-up
	intermediate, err := evaluateTreeNode(tree, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate tree: %w", err)
	}

	// Convert intermediate result to final result
	result := &EvaluationResult{
		Data:          intermediate.Data,
		Entities:      intermediate.Entities,
		Err:           intermediate.Err,
		Intermediates: collectIntermediates(intermediate),
	}

	return result, nil
}

func evaluateTreeNode(node *parser.ExpressionNode, ctx *EvaluationContext) (*IntermediateResult, error) {
	result := &IntermediateResult{}

	// Handle nil node
	if node == nil {
		return result, nil
	}

	// Handle different node types
	switch node.Type {
	case "binary":
		// For binary nodes, evaluate both sides first
		var leftResult, rightResult *IntermediateResult
		var err error

		// Evaluate left side
		if node.Left != nil {
			leftResult, err = evaluateTreeNode(node.Left, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate left subtree: %w", err)
			}
			result.LeftResult = leftResult
		}

		// Evaluate right side
		if node.Right != nil {
			rightResult, err = evaluateTreeNode(node.Right, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate right subtree: %w", err)
			}
			result.RightResult = rightResult
		}

		// Handle logical operators
		if node.Operator == "&&" || node.Operator == "||" {
			// Get the filtered data from both sides
			var leftData, rightData []interface{}

			if leftResult != nil && len(leftResult.Data) > 0 {
				leftData = leftResult.Data
			}

			if rightResult != nil && len(rightResult.Data) > 0 {
				rightData = rightResult.Data
			}

			// For AND, find intersection
			if node.Operator == "&&" {
				result.Data = findIntersection(leftData, rightData)
			} else {
				result.Data = findUnion(leftData, rightData)
			}

			result.Entities = []string{"method_declaration"}
			return result, nil
		}

		// For other binary operations, use standard evaluation
		return evaluateBinaryNode(node, leftResult, rightResult, ctx)

	case "variable":
		// All variables are assumed to be method_declaration fields
		result.Entities = []string{"method_declaration"}

	case "value":
		// Values don't have associated entities
		result.Value = node.Value
	}

	return result, nil
}

// evaluateBinaryNode evaluates a binary operation node
func evaluateBinaryNode(node *parser.ExpressionNode, left, right *IntermediateResult, ctx *EvaluationContext) (*IntermediateResult, error) {
	// Determine the type of comparison
	compType, err := DetectComparisonType(node)
	if err != nil {
		return nil, fmt.Errorf("failed to detect comparison type: %w", err)
	}

	// Get entities involved
	leftEntity, rightEntity, err := getInvolvedEntities(node)
	if err != nil {
		return nil, fmt.Errorf("failed to get involved entities: %w", err)
	}

	// Create result structure
	result := &IntermediateResult{
		NodeType:    node.Type,
		Operator:    node.Operator,
		LeftResult:  left,
		RightResult: right,
		Entities:    []string{},
	}

	// Add entities to the result
	if leftEntity != "" {
		result.Entities = append(result.Entities, leftEntity)
	}
	if rightEntity != "" && rightEntity != leftEntity {
		result.Entities = append(result.Entities, rightEntity)
	}

	// Handle different comparison types
	switch compType {
	case SINGLE_ENTITY:
		if node.Entity == "" {
			if node.Left != nil && node.Left.Entity != "" {
				node.Entity = node.Left.Entity
				node.Alias = node.Left.Alias
			} else if node.Right != nil && node.Right.Entity != "" {
				node.Entity = node.Right.Entity
				node.Alias = node.Right.Alias
			}
		}
		// Filter data based on the expression
		var filteredData []interface{}
		for _, item := range ctx.ProxyEnv[node.Entity] {
			proxyEnv := make(map[string]interface{})
			proxyEnv[node.Alias] = item
			match, err := evaluateNode(node, proxyEnv)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression: %w", err)
			}

			// If it matches, add to filtered data
			if matchBool, ok := match.(bool); ok && matchBool {
				filteredData = append(filteredData, item)
			}
		}

		result.Data = filteredData

	case DUAL_ENTITY:
		// For dual entity comparisons, check if they're related
		hasRelation := ctx.RelationshipMap.HasRelationship(node.Left.Entity, node.Right.Entity)

		// Get data for both entities
		leftData, leftOk := ctx.ProxyEnv[node.Left.Entity]
		rightData, rightOk := ctx.ProxyEnv[node.Right.Entity]

		if !leftOk || !rightOk {
			return nil, fmt.Errorf("missing data for entities: %s, %s", node.Left.Entity, node.Right.Entity)
		}

		// Handle related and unrelated entities
		if hasRelation {
			// For related entities, find matching pairs with optimized approach
			var matchedData []interface{}

			// Build an index of right items by their relationship key to avoid O(n²) complexity
			rightItemIndex := make(map[string][]interface{})
			for _, rightItem := range rightData {
				if relatedID, ok := rightItem["class_id"].(string); ok {
					rightItemIndex[relatedID] = append(rightItemIndex[relatedID], rightItem)
				}
			}
			fmt.Println("Right item index:", rightItemIndex)

			// For each left item, directly access related right items using the index
			for _, leftItem := range leftData {
				if id, ok := leftItem["id"].(string); ok {
					// Get only the related items instead of scanning all
					for _, rightItem := range rightItemIndex[id] {

						// Create proxy environment for evaluation
						proxyEnv := make(map[string]interface{})
						proxyEnv[node.Left.Alias] = leftItem
						proxyEnv[node.Right.Alias] = rightItem

						// Evaluate the expression
						match, err := evaluateNode(node, proxyEnv)
						if err != nil {
							return nil, fmt.Errorf("failed to evaluate expression: %w", err)
						}

						// If it matches, add to matched data
						if matchBool, ok := match.(bool); ok && matchBool {
							matchedData = append(matchedData, leftItem)
							matchedData = append(matchedData, rightItem)
						}
					}
				}
			}

			result.Data = matchedData
		} else {
			// For unrelated entities, use cross product
			var matchedData []interface{}

			// For each left item, check against each right item
			for _, leftItem := range leftData {
				for _, rightItem := range rightData {
					// Create proxy environment for evaluation
					proxyEnv := make(map[string]interface{})
					proxyEnv[node.Left.Alias] = leftItem
					proxyEnv[node.Right.Alias] = rightItem

					// Evaluate the expression
					match, err := evaluateNode(node, proxyEnv)
					if err != nil {
						return nil, fmt.Errorf("failed to evaluate expression: %w", err)
					}

					// If it matches, add to matched data
					if matchBool, ok := match.(bool); ok && matchBool {
						matchedData = append(matchedData, rightItem)
						matchedData = append(matchedData, leftItem)
					}
				}
			}

			result.Data = matchedData
		}

	default:
		return nil, fmt.Errorf("unknown comparison type: %s", compType)
	}

	return result, nil
}

// collectIntermediates collects all intermediate results into a flat list
func collectIntermediates(result *IntermediateResult) []*IntermediateResult {
	if result == nil {
		return nil
	}

	results := []*IntermediateResult{result}

	if result.LeftResult != nil {
		results = append(results, collectIntermediates(result.LeftResult)...)
	}
	if result.RightResult != nil {
		results = append(results, collectIntermediates(result.RightResult)...)
	}

	return results
}

// getInvolvedEntities returns the entity types involved in an expression
func getInvolvedEntities(node *parser.ExpressionNode) (leftEntity, rightEntity string, err error) {
	if node == nil {
		return "", "", fmt.Errorf("nil node")
	}

	switch node.Type {
	case "binary":
		leftEntity, err = getEntityName(node.Left)
		if err != nil {
			return "", "", fmt.Errorf("failed to get left entity: %w", err)
		}

		rightEntity, err = getEntityName(node.Right)
		if err != nil {
			return "", "", fmt.Errorf("failed to get right entity: %w", err)
		}

		return leftEntity, rightEntity, nil

	default:
		return "", "", fmt.Errorf("unsupported node type for getting entities: %s", node.Type)
	}
}

func findUnion(a, b []interface{}) []interface{} {
	seen := make(map[string]bool)
	var result []interface{}

	// Add items from the first slice
	for _, item := range a {
		if val, ok := item.(model.Identifiable); ok {
			id := val.GetID()
			if !seen[id] {
				result = append(result, item)
				seen[id] = true
			}
		}
	}

	// Add items from the second slice if not already present
	for _, item := range b {
		if val, ok := item.(model.Identifiable); ok {
			id := val.GetID()
			if !seen[id] {
				result = append(result, item)
				seen[id] = true
			}
		}
	}

	return result
}

func findIntersection(a []interface{}, b []interface{}) []interface{} {
	idSet := make(map[string]bool)
	var result []interface{}

	// Collect IDs from first slice
	for _, item := range a {
		if val, ok := item.(model.Identifiable); ok {
			idSet[val.GetID()] = true
		}
	}

	// Check intersection with second slice
	for _, item := range b {
		if val, ok := item.(model.Identifiable); ok {
			if idSet[val.GetID()] {
				result = append(result, item)
			}
		}
	}
	return result
}

// evaluateNode recursively evaluates a single node in the expression tree
// returns interface{} to support different types (bool, string, number)
func evaluateNode(node *parser.ExpressionNode, proxyEnv map[string]interface{}) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("nil node")
	}
	var expression string

	leftExpr := node.Left.Value
	rightExpr := node.Right.Value

	if node.Left.Alias != "" {
		leftExpr = fmt.Sprintf("%s.%s", node.Left.Alias, node.Left.Value)
	}

	if node.Right.Alias != "" {
		rightExpr = fmt.Sprintf("%s.%s", node.Right.Alias, node.Right.Value)
	}

	expression = fmt.Sprintf("%s %s %s", leftExpr, node.Operator, rightExpr)

	fmt.Println("Expression:", expression)

	result, err := expr.Compile(expression, expr.Env(proxyEnv))
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", err)
	}
	return expr.Run(result, proxyEnv)
}

func DetectComparisonType(node *parser.ExpressionNode) (ComparisonType, error) {
	if node == nil {
		return "", fmt.Errorf("nil node")
	}

	// Only analyze binary nodes
	if node.Type != "binary" {
		return "", fmt.Errorf("not a binary node")
	}

	// For logical operators (&&, ||), check both sides recursively
	if node.Operator == "&&" || node.Operator == "||" {
		leftType, err := DetectComparisonType(node.Left)
		if err != nil {
			return "", fmt.Errorf("failed to detect left comparison type: %w", err)
		}

		rightType, err := DetectComparisonType(node.Right)
		if err != nil {
			return "", fmt.Errorf("failed to detect right comparison type: %w", err)
		}

		// If either side is DUAL_ENTITY, the whole expression is DUAL_ENTITY
		if leftType == DUAL_ENTITY || rightType == DUAL_ENTITY {
			return DUAL_ENTITY, nil
		}
		return SINGLE_ENTITY, nil
	}

	// For comparison operators, check entity names
	leftEntity, err := getEntityName(node.Left)
	if err != nil {
		return "", fmt.Errorf("failed to get left entity: %w", err)
	}

	rightEntity, err := getEntityName(node.Right)
	if err != nil {
		return "", fmt.Errorf("failed to get right entity: %w", err)
	}

	// If either side is empty (literal/static value) or they're the same entity,
	// it's a SINGLE_ENTITY comparison
	if leftEntity == "" || rightEntity == "" || leftEntity == rightEntity {
		return SINGLE_ENTITY, nil
	}

	// Different entities are being compared
	return DUAL_ENTITY, nil
}

func getEntityName(node *parser.ExpressionNode) (string, error) {
	if node == nil {
		return "", fmt.Errorf("nil node")
	}

	switch node.Type {
	case "variable":
		// Split on dot and take the first part
		parts := strings.Split(node.Value, ".")
		return parts[0], nil
	case "literal":
		return "", nil
	case "method_call":
		return node.Entity, nil
	default:
		return "", fmt.Errorf("unsupported node type: %s", node.Type)
	}
}

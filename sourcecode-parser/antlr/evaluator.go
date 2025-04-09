package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// IntermediateResult represents intermediate evaluation state at each node
type IntermediateResult struct {
	NodeType    string                   // Type of the node (binary, unary, literal, etc)
	Operator    string                   // Operator if binary/unary node
	Data        []map[string]interface{} // Filtered data at this node
	Entities    []string                 // Entities involved at this node
	LeftResult  *IntermediateResult      // Result from left subtree
	RightResult *IntermediateResult      // Result from right subtree
	Err         error                    // Any error at this node
}

// EvaluationResult represents the final result of evaluating an expression
type EvaluationResult struct {
	Data          []map[string]interface{} // The filtered data after evaluation
	Entities      []string                 // The entities involved in this evaluation
	Err           error                    // Any error that occurred during evaluation
	Intermediates []*IntermediateResult    // Intermediate results for debugging
}

// EvaluationContext holds the context for expression evaluation
type EvaluationContext struct {
	RelationshipMap *RelationshipMap
	EntityData      map[string][]map[string]interface{} // map[EntityType][]EntityData
}

// RelationshipMap represents relationships between entities and their attributes
type RelationshipMap struct {
	// map[EntityName]map[AttributeName][]RelatedEntity
	// Example: {"class": {"methods": ["method", "function"]}}
	Relationships map[string]map[string][]string
}

// NewRelationshipMap creates a new RelationshipMap
func NewRelationshipMap() *RelationshipMap {
	return &RelationshipMap{
		Relationships: make(map[string]map[string][]string),
	}
}

// AddRelationship adds a relationship between an entity and its related entities through an attribute
func (rm *RelationshipMap) AddRelationship(entity, attribute string, relatedEntities []string) {
	if rm.Relationships[entity] == nil {
		rm.Relationships[entity] = make(map[string][]string)
	}
	rm.Relationships[entity][attribute] = relatedEntities
}

// HasRelationship checks if two entities are related through any attribute
func (rm *RelationshipMap) HasRelationship(entity1, entity2 string) bool {
	// Check direct relationships from entity1 to entity2
	if attrs, ok := rm.Relationships[entity1]; ok {
		for _, relatedEntities := range attrs {
			for _, related := range relatedEntities {
				if related == entity2 {
					return true
				}
			}
		}
	}

	// Check direct relationships from entity2 to entity1
	if attrs, ok := rm.Relationships[entity2]; ok {
		for _, relatedEntities := range attrs {
			for _, related := range relatedEntities {
				if related == entity1 {
					return true
				}
			}
		}
	}

	return false
}

// CheckExpressionRelationship checks if a binary expression involves related entities
func CheckExpressionRelationship(node *ExpressionNode, relationshipMap *RelationshipMap) (bool, error) {
	// First check if it's a dual entity comparison
	compType, err := DetectComparisonType(node)
	if err != nil {
		return false, fmt.Errorf("failed to detect comparison type: %w", err)
	}

	if compType != DUAL_ENTITY {
		return false, nil // Not a dual entity comparison
	}

	// Get entity names from both sides
	leftEntity, err := getEntityName(node.Left)
	if err != nil {
		return false, fmt.Errorf("failed to get left entity: %w", err)
	}

	rightEntity, err := getEntityName(node.Right)
	if err != nil {
		return false, fmt.Errorf("failed to get right entity: %w", err)
	}

	// Check if entities are related
	return relationshipMap.HasRelationship(leftEntity, rightEntity), nil
}

// ComparisonType represents the type of comparison in an expression
type ComparisonType string

const (
	// SINGLE_ENTITY represents comparison between one entity and a static value
	SINGLE_ENTITY ComparisonType = "SINGLE_ENTITY"
	// DUAL_ENTITY represents comparison between two different entities
	DUAL_ENTITY ComparisonType = "DUAL_ENTITY"
)

// EvaluateExpressionTree evaluates the expression tree against input data
// and returns filtered data based on the expression conditions
func EvaluateExpressionTree(tree *ExpressionNode, ctx *EvaluationContext) (*EvaluationResult, error) {
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

// evaluateTreeNode evaluates a single node in the expression tree
// and returns an intermediate result
func evaluateTreeNode(node *ExpressionNode, ctx *EvaluationContext) (*IntermediateResult, error) {
	if node == nil {
		return nil, fmt.Errorf("nil node")
	}

	result := &IntermediateResult{
		NodeType: node.Type,
		Operator: node.Operator,
	}

	switch node.Type {
	case "binary":
		// For binary nodes, evaluate both sides first
		var leftResult, rightResult *IntermediateResult
		var err error

		if node.Left != nil {
			leftResult, err = evaluateTreeNode(node.Left, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate left subtree: %w", err)
			}
			result.LeftResult = leftResult
		}

		if node.Right != nil {
			rightResult, err = evaluateTreeNode(node.Right, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate right subtree: %w", err)
			}
			result.RightResult = rightResult
		}

		// Now evaluate the binary operation
		if node.Operator == "&&" || node.Operator == "||" {
			// For logical operators, evaluate both sides and combine results
			if leftResult != nil {
				result.Data = append(result.Data, leftResult.Data...)
				result.Entities = append(result.Entities, leftResult.Entities...)
			}
			if rightResult != nil {
				result.Data = append(result.Data, rightResult.Data...)
				result.Entities = append(result.Entities, rightResult.Entities...)
			}
		} else {
			// For comparison operators, evaluate normally
			result, err = evaluateBinaryNode(node, leftResult, rightResult, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate binary node: %w", err)
			}
		}

	case "variable":
		// For variable nodes, evaluate directly
		var err error
		result, err = evaluateVariableNode(node, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate variable node: %w", err)
		}

	case "literal":
		// For literal nodes, evaluate directly
		var err error
		result, err = evaluateLiteralNode(node)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate literal node: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported node type: %s", node.Type)
	}

	return result, nil
}

// evaluateBinaryNode evaluates a binary operation node
func evaluateBinaryNode(node *ExpressionNode, left, right *IntermediateResult, ctx *EvaluationContext) (*IntermediateResult, error) {
	// First determine if this is a single entity or dual entity comparison
	compType, err := DetectComparisonType(node)
	if err != nil {
		return nil, fmt.Errorf("failed to detect comparison type: %w", err)
	}

	// Get entities involved
	leftEntity, rightEntity, err := getInvolvedEntities(node)
	if err != nil {
		return nil, fmt.Errorf("failed to get involved entities: %w", err)
	}

	var result *IntermediateResult

	// Handle different comparison types
	switch compType {
	case SINGLE_ENTITY:
		eval, err := evaluateSingleEntity(node, leftEntity, ctx)
		if err != nil {
			return nil, err
		}
		result = &IntermediateResult{
			NodeType: node.Type,
			Operator: node.Operator,
			Data:     eval.Data,
			Entities: eval.Entities,
			Err:      eval.Err,
		}

	case DUAL_ENTITY:
		// Check if entities are related
		hasRelation := ctx.RelationshipMap.HasRelationship(leftEntity, rightEntity)

		var eval *EvaluationResult
		if hasRelation {
			eval, err = evaluateRelatedEntities(node, leftEntity, rightEntity, ctx)
		} else {
			eval, err = evaluateUnrelatedEntities(node, leftEntity, rightEntity, ctx)
		}
		if err != nil {
			return nil, err
		}
		result = &IntermediateResult{
			NodeType: node.Type,
			Operator: node.Operator,
			Data:     eval.Data,
			Entities: eval.Entities,
			Err:      eval.Err,
		}

	default:
		return nil, fmt.Errorf("unknown comparison type: %s", compType)
	}

	// Store left and right results
	result.LeftResult = left
	result.RightResult = right

	return result, nil
}

// evaluateVariableNode evaluates a variable node
func evaluateVariableNode(node *ExpressionNode, ctx *EvaluationContext) (*IntermediateResult, error) {
	// Get entity name
	entityName, err := getEntityName(node)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity name: %w", err)
	}

	// Get entity data
	data, ok := ctx.EntityData[entityName]
	if !ok {
		return nil, fmt.Errorf("entity data not found: %s", entityName)
	}

	return &IntermediateResult{
		NodeType: node.Type,
		Data:     data,
		Entities: []string{entityName},
	}, nil
}

// evaluateLiteralNode evaluates a literal node
func evaluateLiteralNode(node *ExpressionNode) (*IntermediateResult, error) {
	return &IntermediateResult{
		NodeType: node.Type,
		Data:     []map[string]interface{}{{"value": node.Value}},
	}, nil
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
func getInvolvedEntities(node *ExpressionNode) (leftEntity, rightEntity string, err error) {
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

// evaluateSingleEntity handles evaluation of expressions involving a single entity type
func evaluateSingleEntity(node *ExpressionNode, entity string, ctx *EvaluationContext) (*EvaluationResult, error) {
	// Get data for the entity
	data, ok := ctx.EntityData[entity]
	if !ok {
		return nil, fmt.Errorf("no data found for entity: %s", entity)
	}

	// Filter data based on the expression
	result := make([]map[string]interface{}, 0)
	for _, item := range data {
		// Evaluate the expression for this item
		matches, err := evaluateNode(node, item)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate node: %w", err)
		}

		// Include item if it matches
		if matches.(bool) {
			result = append(result, item)
		}
	}

	return &EvaluationResult{
		Data:     result,
		Entities: []string{entity},
	}, nil
}

// evaluateRelatedEntities handles evaluation of expressions involving two related entities
func evaluateRelatedEntities(node *ExpressionNode, entity1, entity2 string, ctx *EvaluationContext) (*EvaluationResult, error) {
	// Get data for both entities
	data1, ok1 := ctx.EntityData[entity1]
	data2, ok2 := ctx.EntityData[entity2]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("missing data for entities: %s, %s", entity1, entity2)
	}

	// Perform a join operation based on the relationship
	result := make([]map[string]interface{}, 0)
	for _, item1 := range data1 {
		for _, item2 := range data2 {
			// Check if these items are related (this would depend on your data structure)
			if areItemsRelated(item1, item2, entity1, entity2) {
				// Merge the items
				mergedItem := mergeItems(item1, item2)

				// Evaluate the expression on the merged item
				matches, err := evaluateNode(node, mergedItem)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate node: %w", err)
				}

				// Include item if it matches
				if matches.(bool) {
					result = append(result, mergedItem)
				}
			}
		}
	}

	return &EvaluationResult{
		Data:     result,
		Entities: []string{entity1, entity2},
	}, nil
}

// evaluateUnrelatedEntities handles evaluation of expressions involving two unrelated entities
func evaluateUnrelatedEntities(node *ExpressionNode, entity1, entity2 string, ctx *EvaluationContext) (*EvaluationResult, error) {
	// Get data for both entities
	data1, ok1 := ctx.EntityData[entity1]
	data2, ok2 := ctx.EntityData[entity2]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("missing data for entities: %s, %s", entity1, entity2)
	}

	// Perform a cartesian product
	result := make([]map[string]interface{}, 0)
	for _, item1 := range data1 {
		for _, item2 := range data2 {
			// Merge the items
			mergedItem := mergeItems(item1, item2)

			// Evaluate the expression on the merged item
			matches, err := evaluateNode(node, mergedItem)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate node: %w", err)
			}

			// Include item if it matches
			if matches.(bool) {
				result = append(result, mergedItem)
			}
		}
	}

	return &EvaluationResult{
		Data:     result,
		Entities: []string{entity1, entity2},
	}, nil
}

// areItemsRelated checks if two items are related based on their entity types
func areItemsRelated(item1, item2 map[string]interface{}, entity1, entity2 string) bool {
	// This is a placeholder. The actual implementation would depend on your data structure
	// For example, if entity1 is "class" and entity2 is "method",
	// you might check if item2["class_id"] == item1["id"]

	// For now, we'll assume they're related if they have matching IDs
	id1, ok1 := item1["id"]
	id2, ok2 := item2[entity1+"_id"]
	fmt.Println("Checking relationship:", id1, id2)
	if !ok1 || !ok2 {
		return false
	}

	return id1 == id2
}

// mergeItems merges two items into a single map
func mergeItems(item1, item2 map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy all items from item1
	for k, v := range item1 {
		result["class."+k] = v
	}

	// Copy all items from item2, prefixing keys to avoid conflicts
	for k, v := range item2 {
		result["method."+k] = v
	}
	fmt.Println("Merged item:", result)
	return result
}

// evaluateNode recursively evaluates a single node in the expression tree
// returns interface{} to support different types (bool, string, number)
func evaluateNode(node *ExpressionNode, data map[string]interface{}) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("nil node")
	}

	switch node.Type {
	case "binary":
		left, err := evaluateNode(node.Left, data)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate left node: %w", err)
		}

		right, err := evaluateNode(node.Right, data)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate right node: %w", err)
		}

		// Handle comparison operators
		switch node.Operator {
		case "==":
			return left == right, nil
		case "!=":
			return left != right, nil
		case ">":
			// Convert to float64 for numeric comparison
			leftNum, leftOk := toFloat64(left)
			rightNum, rightOk := toFloat64(right)
			if !leftOk || !rightOk {
				return nil, fmt.Errorf("cannot compare non-numeric values with >")
			}
			return leftNum > rightNum, nil
		case "<":
			leftNum, leftOk := toFloat64(left)
			rightNum, rightOk := toFloat64(right)
			if !leftOk || !rightOk {
				return nil, fmt.Errorf("cannot compare non-numeric values with <")
			}
			return leftNum < rightNum, nil
		case ">=":
			leftNum, leftOk := toFloat64(left)
			rightNum, rightOk := toFloat64(right)
			if !leftOk || !rightOk {
				return nil, fmt.Errorf("cannot compare non-numeric values with >=")
			}
			return leftNum >= rightNum, nil
		case "<=":
			leftNum, leftOk := toFloat64(left)
			rightNum, rightOk := toFloat64(right)
			if !leftOk || !rightOk {
				return nil, fmt.Errorf("cannot compare non-numeric values with <=")
			}
			return leftNum <= rightNum, nil
		case "&&":
			leftBool, leftOk := left.(bool)
			rightBool, rightOk := right.(bool)
			if !leftOk || !rightOk {
				return nil, fmt.Errorf("cannot perform logical AND on non-boolean values")
			}
			return leftBool && rightBool, nil
		case "||":
			leftBool, leftOk := left.(bool)
			rightBool, rightOk := right.(bool)
			if !leftOk || !rightOk {
				return nil, fmt.Errorf("cannot perform logical OR on non-boolean values")
			}
			return leftBool || rightBool, nil
		default:
			return nil, fmt.Errorf("unsupported operator: %s", node.Operator)
		}

	case "variable":
		// Handle entity paths (e.g., "class.name")
		parts := strings.Split(node.Value, ".")
		if len(parts) > 1 {
			// Extract field
			field := parts[1]

			// Get the value from data
			val, ok := data[node.Value]
			if !ok {
				return nil, fmt.Errorf("field not found: %s", field)
			}
			return val, nil
		}

		// Regular variable
		val, ok := data[node.Value]
		if !ok {
			return nil, fmt.Errorf("variable not found: %s", node.Value)
		}
		return val, nil

	case "literal":
		// Try to parse the literal value
		if strings.HasPrefix(node.Value, "\"") && strings.HasSuffix(node.Value, "\"") {
			// String literal
			return strings.Trim(node.Value, "\""), nil
		}

		// Try to parse as number
		if val, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return val, nil
		}

		return node.Value, nil

	default:
		return nil, fmt.Errorf("unsupported node type: %s", node.Type)
	}
}

// toFloat64 converts an interface{} to a float64 if possible
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// nodeToExprString converts an ExpressionNode to an expr-lang expression string
// DetectComparisonType analyzes a binary expression node and determines if it's comparing
// a single entity with a static value or comparing two different entities
func DetectComparisonType(node *ExpressionNode) (ComparisonType, error) {
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

// getEntityName extracts the entity name from a variable path (e.g. "class.name" -> "class")
func getEntityName(node *ExpressionNode) (string, error) {
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
	default:
		return "", fmt.Errorf("unsupported node type: %s", node.Type)
	}
}

func nodeToExprString(node *ExpressionNode) (string, error) {
	switch node.Type {
	case "binary":
		left, err := nodeToExprString(node.Left)
		if err != nil {
			return "", err
		}
		right, err := nodeToExprString(node.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s)", left, node.Operator, right), nil

	case "literal":
		return node.Value, nil

	case "method_call":
		// Format method call with arguments
		args := make([]string, 0, len(node.Args))
		for _, arg := range node.Args {
			argStr, err := nodeToExprString(&arg)
			if err != nil {
				return "", err
			}
			args = append(args, argStr)
		}
		return fmt.Sprintf("%s(%s)", node.Value, strings.Join(args, ", ")), nil

	case "predicate_call":
		// Similar to method_call
		args := make([]string, 0, len(node.Args))
		for _, arg := range node.Args {
			argStr, err := nodeToExprString(&arg)
			if err != nil {
				return "", err
			}
			args = append(args, argStr)
		}
		return fmt.Sprintf("%s(%s)", node.Value, strings.Join(args, ", ")), nil

	case "variable":
		return node.Value, nil

	case "unary":
		right, err := nodeToExprString(node.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%s", node.Operator, right), nil

	default:
		return "", fmt.Errorf("unknown node type: %s", node.Type)
	}
}

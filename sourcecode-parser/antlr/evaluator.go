package parser

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
)

// IntermediateResult represents intermediate evaluation state at each node
type IntermediateResult struct {
	NodeType    string
	Operator    string
	Data        []map[string]interface{}
	Entities    []string
	LeftResult  *IntermediateResult
	RightResult *IntermediateResult
	Value       interface{}
	Err         error
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
	ProxyEnv        map[string][]map[string]interface{}
}

// RelationshipMap represents relationships between entities and their attributes
type RelationshipMap struct {
	// Direct mapping between entities for faster lookups
	// map[EntityName]map[RelatedEntityName]bool
	DirectRelationships map[string]map[string]bool
	// Original relationships for attribute-based queries
	Relationships map[string]map[string][]string
}

// NewRelationshipMap creates a new RelationshipMap
func NewRelationshipMap() *RelationshipMap {
	return &RelationshipMap{
		DirectRelationships: make(map[string]map[string]bool),
		Relationships:       make(map[string]map[string][]string),
	}
}

// AddRelationship adds a relationship between an entity and its related entities through an attribute
func (rm *RelationshipMap) AddRelationship(entity, attribute string, relatedEntities []string) {
	// Store the original relationship structure
	if rm.Relationships[entity] == nil {
		rm.Relationships[entity] = make(map[string][]string)
	}
	rm.Relationships[entity][attribute] = relatedEntities

	// Also store direct entity-to-entity relationships for faster lookups
	for _, related := range relatedEntities {
		// Create entity1 -> entity2 relationship
		if rm.DirectRelationships[entity] == nil {
			rm.DirectRelationships[entity] = make(map[string]bool)
		}
		rm.DirectRelationships[entity][related] = true

		// Create entity2 -> entity1 relationship (bidirectional)
		if rm.DirectRelationships[related] == nil {
			rm.DirectRelationships[related] = make(map[string]bool)
		}
		rm.DirectRelationships[related][entity] = true
	}
}

// HasRelationship checks if two entities are related through any attribute
func (rm *RelationshipMap) HasRelationship(entity1, entity2 string) bool {
	// Use the optimized direct relationship lookup
	if relatedEntities, ok := rm.DirectRelationships[entity1]; ok {
		if _, related := relatedEntities[entity2]; related {
			return true
		}
	}

	return false
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
			var leftData, rightData []map[string]interface{}

			if leftResult != nil && len(leftResult.Data) > 0 {
				leftData = leftResult.Data
			}

			// print leftData
			fmt.Println("leftData:", leftData)

			if rightResult != nil && len(rightResult.Data) > 0 {
				rightData = rightResult.Data
			}

			// print rightData
			fmt.Println("rightData:", rightData)

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
func evaluateBinaryNode(node *ExpressionNode, left, right *IntermediateResult, ctx *EvaluationContext) (*IntermediateResult, error) {
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
			} else if node.Right != nil && node.Right.Entity != "" {
				node.Entity = node.Right.Entity
			}
		}
		entityData, ok := ctx.EntityData[node.Entity]
		if !ok {
			return nil, fmt.Errorf("no data for entity : %s", node.Entity)
		}

		// Filter data based on the expression
		var filteredData []map[string]interface{}
		for i, item := range entityData {
			match, err := evaluateNode(node, ctx.ProxyEnv[node.Entity][i])
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
		leftData, leftOk := ctx.EntityData[node.Left.Entity]
		rightData, rightOk := ctx.EntityData[node.Right.Entity]

		if !leftOk || !rightOk {
			return nil, fmt.Errorf("missing data for entities: %s, %s", node.Left.Entity, node.Right.Entity)
		}

		// Handle related and unrelated entities
		if hasRelation {
			// For related entities, find matching pairs with optimized approach
			var matchedData []map[string]interface{}

			// Build an index of right items by their relationship key to avoid O(n²) complexity
			rightItemIndex := make(map[string][]map[string]interface{})
			for _, rightItem := range rightData {
				if relatedID, ok := rightItem["class_id"].(string); ok {
					rightItemIndex[relatedID] = append(rightItemIndex[relatedID], rightItem)
				}
			}

			// For each left item, directly access related right items using the index
			for i, leftItem := range leftData {
				// fmt.Println("leftItem:", leftItem)
				if id, ok := leftItem["id"].(string); ok {
					// Get only the related items instead of scanning all
					for j, rightItem := range rightItemIndex[id] {
						// Merge the items
						mergedItem := mergeItems(leftItem, rightItem)

						for k, v := range mergedItem {
							// pretty print mergedItem
							fmt.Println(k, v)
						}

						// Create proxy environment for evaluation
						proxyEnv := make(map[string]interface{})
						for k, v := range mergedItem {
							proxyEnv[k] = v
							proxyEnv["cd"] = ctx.ProxyEnv[node.Left.Entity][i]
							proxyEnv["md"] = ctx.ProxyEnv[node.Right.Entity][j]
						}

						// Add "cd." and "md." prefixes only if they're not already present
						if !strings.HasPrefix(node.Left.Value, "cd.") {
							node.Left.Value = fmt.Sprintf("%s.%s", "cd", node.Left.Value)
						}
						if !strings.HasPrefix(node.Right.Value, "md.") {
							node.Right.Value = fmt.Sprintf("%s.%s", "md", node.Right.Value)
						}

						// Evaluate the expression
						match, err := evaluateNode(node, proxyEnv)
						if err != nil {
							fmt.Printf("failed to evaluate expression: %s\n", err)
							//return nil, fmt.Errorf("failed to evaluate expression: %w", err)
						}

						// If it matches, add to matched data
						if matchBool, ok := match.(bool); ok && matchBool {
							matchedData = append(matchedData, mergedItem)
						}
					}
				}
			}

			result.Data = matchedData
		} else {
			// For unrelated entities, use cross product
			var matchedData []map[string]interface{}

			// For each left item, check against each right item
			for _, leftItem := range leftData {
				for _, rightItem := range rightData {
					// Merge the items
					mergedItem := mergeItems(leftItem, rightItem)

					// Create proxy environment for evaluation
					proxyEnv := make(map[string]interface{})
					for k, v := range mergedItem {
						proxyEnv[k] = v
					}

					// Evaluate the expression
					match, err := evaluateNode(node, proxyEnv)
					if err != nil {
						return nil, fmt.Errorf("failed to evaluate expression: %w", err)
					}

					// If it matches, add to matched data
					if matchBool, ok := match.(bool); ok && matchBool {
						matchedData = append(matchedData, mergedItem)
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

// findIntersection finds the intersection of two data sets based on ID
func findIntersection(data1, data2 []map[string]interface{}) []map[string]interface{} {
	if len(data1) == 0 || len(data2) == 0 {
		return []map[string]interface{}{}
	}

	// Create a map for faster lookups
	idMap := make(map[string]map[string]interface{})
	for _, item := range data1 {
		if id, ok := item["id"].(string); ok {
			idMap[id] = item
		}
	}

	// Find items that exist in both sets
	result := []map[string]interface{}{}
	for _, item := range data2 {
		if id, ok := item["id"].(string); ok {
			if _, exists := idMap[id]; exists {
				result = append(result, item)
			}
		}
	}

	return result
}

// findUnion finds the union of two data sets based on ID
func findUnion(data1, data2 []map[string]interface{}) []map[string]interface{} {
	// Create a map to avoid duplicates
	idMap := make(map[string]map[string]interface{})

	// Add all items from first set
	for _, item := range data1 {
		if id, ok := item["id"].(string); ok {
			idMap[id] = item
		}
	}

	// Add all items from second set
	for _, item := range data2 {
		if id, ok := item["id"].(string); ok {
			idMap[id] = item
		}
	}

	// Convert map back to slice
	result := []map[string]interface{}{}
	for _, item := range idMap {
		result = append(result, item)
	}

	return result
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
func evaluateNode(node *ExpressionNode, proxyEnv map[string]interface{}) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("nil node")
	}

	expression := fmt.Sprintf("%s %s %s", node.Left.Value, node.Operator, node.Right.Value)

	fmt.Println("Expression:", expression)
	// cast data to model.Method

	result, err := expr.Compile(expression, expr.Env(proxyEnv))
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", err)
	}
	return expr.Run(result, proxyEnv)
}

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
	case "method_call":
		return node.Entity, nil
	default:
		return "", fmt.Errorf("unsupported node type: %s", node.Type)
	}
}

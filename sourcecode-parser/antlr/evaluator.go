package parser

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
)

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
func EvaluateExpressionTree(tree *ExpressionNode, data []map[string]interface{}) ([]map[string]interface{}, error) {
	if tree == nil {
		return data, nil
	}

	var result []map[string]interface{}
	for _, item := range data {
		matches, err := evaluateNode(tree, item)
		if err != nil {
			return nil, fmt.Errorf("evaluation error: %w", err)
		}

		// Only include items that match the expression
		if matches.(bool) {
			result = append(result, item)
		}
	}
	return result, nil
}

// evaluateNode recursively evaluates a single node in the expression tree
// returns interface{} to support different types (bool, string, number)
func evaluateNode(node *ExpressionNode, data map[string]interface{}) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("nil node")
	}

	// Convert expression node to expr-lang expression string
	exprStr, err := nodeToExprString(node)
	if err != nil {
		return nil, fmt.Errorf("failed to convert node to expr: %w", err)
	}

	// Compile the expression
	program, err := expr.Compile(exprStr, expr.Env(data))
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", err)
	}

	// Run the expression with the data
	result, err := expr.Run(program, data)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return result, nil
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

	// Get entity names from left and right sides
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

// getEntityName extracts the entity name from a node.
// Returns empty string for literals and static values.
func getEntityName(node *ExpressionNode) (string, error) {
	if node == nil {
		return "", fmt.Errorf("nil node")
	}

	switch node.Type {
	case "variable":
		return node.Value, nil
	case "method_call":
		// For method calls, consider the target object as the entity
		parts := strings.Split(node.Value, ".")
		if len(parts) > 0 {
			return parts[0], nil
		}
		return "", nil
	case "literal":
		return "", nil // Literals are static values
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

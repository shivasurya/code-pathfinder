package parser

import (
	"fmt"
	"strings"
	"github.com/expr-lang/expr"
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



package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func GenerateMethodID(methodName string, parameters []string, sourceFile string) string {
	hashInput := fmt.Sprintf("%s-%s-%s", methodName, parameters, sourceFile)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

func GenerateSha256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// Helper function to append a node to a slice only if it's not already present.
func appendUnique(slice []*Node, node *Node) []*Node {
	for _, n := range slice {
		if n == node {
			return slice
		}
	}
	return append(slice, node)
}

func FormatType(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%.2f", val)
	case []interface{}:
		//nolint:all
		jsonBytes, _ := json.Marshal(val)
		return string(jsonBytes)
	default:
		return fmt.Sprintf("%v", val)
	}
}

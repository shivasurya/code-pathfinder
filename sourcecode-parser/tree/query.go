package graph

import (
	"fmt"
	"log"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/analytics"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/db"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/eval"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
)

func QueryEntities(db *db.StorageNode, query parser.Query) (nodes []*model.Node, output [][]interface{}) {
	// Create evaluation context
	ctx := &eval.EvaluationContext{
		RelationshipMap: buildRelationshipMap(),
		ProxyEnv:        make(map[string][]map[string]interface{}),
		EntityModel:     make(map[string][]interface{}),
	}

	// Prepare entity data by using db.StorageNode getter methods
	for _, entity := range query.SelectList {
		analytics.ReportEvent(entity.Entity)

		switch entity.Entity {
		case "method_declaration":
			// Get method declarations from db
			methods := db.GetMethodDecls()
			methodProxyEnv := []map[string]interface{}{}
			entityModel := []interface{}{}
			for _, method := range methods {
				entityModel = append(entityModel, method)
				methodProxyEnv = append(methodProxyEnv, method.GetProxyEnv())
			}
			ctx.ProxyEnv[entity.Entity] = methodProxyEnv
			ctx.EntityModel[entity.Entity] = entityModel
		case "class_declaration":
			// Get class declarations from db
			classes := db.GetClassDecls()
			classProxyEnv := []map[string]interface{}{}
			ctx.EntityModel[entity.Entity] = make([]interface{}, len(classes))
			for _, class := range classes {
				ctx.EntityModel[entity.Entity] = append(ctx.EntityModel[entity.Entity], class)
				classProxyEnv = append(classProxyEnv, class.GetProxyEnv())
			}
			ctx.ProxyEnv[entity.Entity] = classProxyEnv
		case "field":
			// Get field declarations from db
			fields := db.GetFields()
			ctx.EntityModel[entity.Entity] = make([]interface{}, len(fields))
			fieldProxyEnv := []map[string]interface{}{}
			for _, field := range fields {
				ctx.EntityModel[entity.Entity] = append(ctx.EntityModel[entity.Entity], field)
				fieldProxyEnv = append(fieldProxyEnv, field.GetProxyEnv())
			}
			ctx.ProxyEnv[entity.Entity] = fieldProxyEnv
		}
	}

	// Use the expression tree from the query
	if query.ExpressionTree == nil {
		return nil, nil
	}

	// Evaluate the condition
	result, err := eval.EvaluateExpressionTree(query.ExpressionTree, ctx)
	if err != nil {
		// Handle error appropriately
		fmt.Println("Error evaluating expression tree:", err)
		return nil, nil
	}

	// loop over result.Data and print each item
	// for _, data := range result.Data {
	// 	fmt.Println(data)
	// }

	// Convert result data back to nodes
	resultNodes := make([]*model.Node, 0)
	for _, data := range result.Data {
		node := &model.Node{}
		if method, ok := data.(*model.Method); ok {
			node.MethodDecl = method
		}
		resultNodes = append(resultNodes, node)
	}

	output = generateOutput(resultNodes, query)
	return resultNodes, output
}

// buildRelationshipMap creates a relationship map for the entities.
func buildRelationshipMap() *eval.RelationshipMap {
	rm := eval.NewRelationshipMap()
	// Add relationships between entities
	// For example:
	rm.AddRelationship("class_declaration", "class_id", []string{"method_declaration"})
	rm.AddRelationship("method_declaration", "class_id", []string{"class_declaration"})
	return rm
}

func generateOutput(nodes []*model.Node, query parser.Query) [][]interface{} {
	results := make([][]interface{}, 0, len(nodes))

	for _, node := range nodes {
		var result []interface{}
		for _, outputFormat := range query.SelectOutput {
			switch outputFormat.Type {
			case "string":
				// Remove quotes from string literals
				cleanedString := strings.ReplaceAll(outputFormat.SelectEntity, "\"", "")
				result = append(result, cleanedString)

			case "variable", "method_chain":
				// Add toString method for variables if not present
				expression := outputFormat.SelectEntity
				if outputFormat.Type == "variable" && !strings.HasSuffix(expression, ".toString()") {
					expression += ".toString()"
				}

				// Skip invalid method chains
				if outputFormat.Type == "method_chain" && !strings.Contains(expression, ".") {
					continue
				}

				if outputFormat.Type == "method_chain" {
					// remove md.
					expression = strings.ReplaceAll(expression, "md.", "")
				}

				// Evaluate the expression
				response, err := evaluateExpression([]*model.Node{node}, expression)
				if err != nil {
					log.Printf("Error evaluating expression %s: %v", expression, err)
					result = append(result, "") // Add empty string on error
				} else {
					result = append(result, response)
				}
			}
		}
		results = append(results, result)
	}

	return results
}

func evaluateExpression(node []*model.Node, expression string) (interface{}, error) {
	var env map[string]interface{}
	for _, n := range node {
		env = n.MethodDecl.GetProxyEnv()
	}
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

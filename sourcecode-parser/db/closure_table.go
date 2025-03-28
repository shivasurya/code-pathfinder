package db

import (
	"database/sql"
	"fmt"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
)

type ClosureTableRow struct {
	Ancestor   int64
	Descendant int64
	Depth      int64
}

// Convert Tree into Closure Table entries.
func BuildClosureTable(node *model.TreeNode, ancestors []int64, depth int64, closure []ClosureTableRow) []ClosureTableRow {
	if node == nil {
		return closure
	}

	// Store the ancestor-descendant relationship
	for _, ancestor := range ancestors {
		closure = append(closure, ClosureTableRow{
			Ancestor:   ancestor,
			Descendant: node.Node.NodeID,
			Depth:      depth,
		})
	}

	closure = append(closure, ClosureTableRow{
		Ancestor:   node.Node.NodeID,
		Descendant: node.Node.NodeID,
		Depth:      0,
	})

	// Recursively process children
	for _, child := range node.Children {
		newAncestors := append(ancestors, node.Node.NodeID) // Pass the entire chain of ancestors
		closure = BuildClosureTable(child, newAncestors, depth+1, closure)
	}

	return closure
}

func StoreClosureTable(db *sql.DB, closureTable []ClosureTableRow, file string) {
	// Insert closure table relationships
	query := "INSERT INTO closure_table (ancestor, descendant, depth, file) VALUES (?, ?, ?, ?)"
	for _, row := range closureTable {
		_, err := db.Exec(query, row.Ancestor, row.Descendant, row.Depth, file)
		if err != nil {
			fmt.Println("Error inserting closure table row:", err)
		}
	}
}

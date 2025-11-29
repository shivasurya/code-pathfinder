package core

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
)

// TypeInfo represents inferred type information for a variable or expression.
// It tracks the fully qualified type name, confidence level, and how the type was inferred.
type TypeInfo struct {
	TypeFQN    string  // Fully qualified type name (e.g., "builtins.str", "myapp.models.User")
	Confidence float32 // Confidence level from 0.0 to 1.0 (1.0 = certain, 0.5 = heuristic, 0.0 = unknown)
	Source     string  // How the type was inferred (e.g., "literal", "assignment", "annotation")
}

// ClassAttribute represents a single attribute of a class.
type ClassAttribute struct {
	Name       string                 // Attribute name (e.g., "value", "user")
	Type       *TypeInfo              // Inferred type of the attribute
	AssignedIn string                 // Method where assigned (e.g., "__init__", "setup")
	Location   *graph.SourceLocation  // Source location of the attribute
	Confidence float64                // Confidence in type inference (0.0-1.0)
}

// ClassAttributes holds all attributes for a single class.
type ClassAttributes struct {
	ClassFQN   string                        // Fully qualified class name (e.g., "myapp.models.User")
	Attributes map[string]*ClassAttribute    // Map from attribute name to attribute info
	Methods    []string                      // List of method FQNs in this class
	FilePath   string                        // Source file path where class is defined
}

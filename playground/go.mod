module github.com/shivasurya/code-pathfinder/playground

go 1.24.1

replace github.com/shivasurya/code-pathfinder/sourcecode-parser => ../sourcecode-parser

require (
	github.com/google/uuid v1.6.0
	github.com/shivasurya/code-pathfinder/sourcecode-parser v0.0.0-00010101000000-000000000000
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/expr-lang/expr v1.16.9 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/posthog/posthog-go v1.2.24 // indirect
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82 // indirect
	golang.org/x/exp v0.0.0-20250106191152-7588d65b2ba8 // indirect
)

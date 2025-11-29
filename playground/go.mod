module github.com/shivasurya/code-pathfinder/playground

go 1.25.3

replace github.com/shivasurya/code-pathfinder/sast-engine => ../sast-engine

require (
	github.com/google/uuid v1.6.0
	github.com/shivasurya/code-pathfinder/sast-engine v0.0.0-20251019001255-25ad474e19a3
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/expr-lang/expr v1.17.6 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/posthog/posthog-go v1.6.11 // indirect
	golang.org/x/exp v0.0.0-20251017212417-90e834f514db // indirect
)

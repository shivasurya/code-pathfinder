module github.com/shivasurya/code-pathfinder/playground

go 1.23

replace github.com/shivasurya/code-pathfinder/sourcecode-parser => ../sourcecode-parser

require (
	github.com/google/uuid v1.6.0
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
)

require github.com/stretchr/testify v1.10.0 // indirect

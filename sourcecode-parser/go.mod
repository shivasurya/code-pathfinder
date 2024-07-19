module github.com/shivasurya/code-pathfinder/sourcecode-parser

go 1.22.0

require github.com/smacker/go-tree-sitter v0.0.0-20240625050157-a31a98a7c0f6

require (
	github.com/antlr4-go/antlr/v4 v4.13.1
	queryparser v0.0.0-00010101000000-000000000000
)

require github.com/stretchr/testify v1.9.0 // indirect

require golang.org/x/exp v0.0.0-20240716175740-e3f259677ff7 // indirect

replace queryparser => ./queryparser

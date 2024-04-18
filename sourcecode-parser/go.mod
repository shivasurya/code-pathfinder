module sourcecode-parser

go 1.20

require github.com/smacker/go-tree-sitter v0.0.0-20231219031718-233c2f923ac7

require (
	github.com/stretchr/testify v1.7.4
	queryparser v0.0.0-00010101000000-000000000000
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace queryparser => ./queryparser

package cfg

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// BuildCFGForLanguage routes to the language-specific CFG builder.
// lang comes from graph.Node.Language ("go" or "python", set during parsing).
// Returns (nil, nil, error) for unsupported languages; callers fall back to
// flat statement extraction (Tier 2) or line-proximity (Tier 3).
func BuildCFGForLanguage(
	lang string,
	funcFQN string,
	funcNode *sitter.Node,
	srcCode []byte,
) (*ControlFlowGraph, BlockStatements, error) {
	switch lang {
	case "python":
		return BuildCFGFromAST(funcFQN, funcNode, srcCode)
	case "go":
		return BuildGoCFGFromAST(funcFQN, funcNode, srcCode)
	default:
		return nil, nil, fmt.Errorf("cfg: no builder for language %q", lang)
	}
}

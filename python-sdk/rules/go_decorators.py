"""
Backward-compatibility shim. go_decorators has moved to the codepathfinder package.
Import from: from codepathfinder.go_decorators import go_rule
"""
from codepathfinder.go_decorators import (  # noqa: F401
    GoRuleMetadata,
    GoRuleDefinition,
    go_rule,
    get_go_rules,
    clear_go_rules,
)

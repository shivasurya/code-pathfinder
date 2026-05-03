"""
Backward-compatibility shim. c_decorators has moved to the codepathfinder package.
Import from: from codepathfinder.c_decorators import c_rule
"""
from codepathfinder.c_decorators import (  # noqa: F401
    CRuleMetadata,
    CRuleDefinition,
    c_rule,
    get_c_rules,
    clear_c_rules,
)

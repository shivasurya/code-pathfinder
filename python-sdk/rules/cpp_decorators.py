"""
Backward-compatibility shim. cpp_decorators has moved to the codepathfinder package.
Import from: from codepathfinder.cpp_decorators import cpp_rule
"""
from codepathfinder.cpp_decorators import (  # noqa: F401
    CppRuleMetadata,
    CppRuleDefinition,
    cpp_rule,
    get_cpp_rules,
    clear_cpp_rules,
)

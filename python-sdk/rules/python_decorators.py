"""
Backward-compatibility shim. python_decorators has moved to the codepathfinder package.
Import from: from codepathfinder.python_decorators import python_rule
"""
from codepathfinder.python_decorators import (  # noqa: F401
    PythonRuleMetadata,
    PythonRuleDefinition,
    python_rule,
    get_python_rules,
    clear_rules,
)

"""
Backward-compatibility shim. go_ir has moved to the codepathfinder package.
"""
from codepathfinder.go_ir import (  # noqa: F401
    compile_go_rules,
    compile_all_rules,
)

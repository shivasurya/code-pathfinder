"""
Backward-compatibility shim. c_ir has moved to the codepathfinder package.
"""
from codepathfinder.c_ir import (  # noqa: F401
    compile_c_rules,
    compile_all_rules,
)

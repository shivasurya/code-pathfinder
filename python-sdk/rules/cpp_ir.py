"""
Backward-compatibility shim. cpp_ir has moved to the codepathfinder package.
"""
from codepathfinder.cpp_ir import (  # noqa: F401
    compile_cpp_rules,
    compile_all_rules,
)

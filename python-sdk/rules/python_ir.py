"""
Backward-compatibility shim. python_ir has moved to the codepathfinder package.
Import from: from codepathfinder.python_ir import compile_python_rules
"""
from codepathfinder.python_ir import (  # noqa: F401
    compile_python_rules,
    compile_all_rules,
    compile_to_json,
    write_ir_file,
)

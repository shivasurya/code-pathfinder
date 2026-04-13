"""
Backward-compatibility shim. container_ir has moved to the codepathfinder package.
"""
from codepathfinder.container_ir import (  # noqa: F401
    compile_dockerfile_rules,
    compile_compose_rules,
    compile_all_rules,
    compile_to_json,
    write_ir_file,
)

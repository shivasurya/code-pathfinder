"""
Backward-compatibility shim. container_combinators has moved to the codepathfinder package.
Import from: from codepathfinder.container_combinators import all_of, any_of
"""
from codepathfinder.container_combinators import (  # noqa: F401
    CombinatorMatcher,
    all_of,
    any_of,
    none_of,
    instruction_after,
    instruction_before,
    stage,
    final_stage_has,
)

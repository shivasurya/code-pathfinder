"""
Backward-compatibility shim. container_matchers has moved to the codepathfinder package.
Import from: from codepathfinder.container_matchers import instruction, missing
"""
from codepathfinder.container_matchers import (  # noqa: F401
    Matcher,
    instruction,
    missing,
    service_has,
    service_missing,
)

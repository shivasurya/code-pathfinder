"""
Backward-compatibility shim. container_programmatic has moved to the codepathfinder package.
Import from: from codepathfinder.container_programmatic import custom_check
"""
from codepathfinder.container_programmatic import (  # noqa: F401
    ProgrammaticMatcher,
    custom_check,
    DockerfileAccess,
    ComposeAccess,
)
